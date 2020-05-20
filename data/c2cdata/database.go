package c2cdata

import (
	cf "blabu/c2cService/configuration"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	bolt "github.com/etcd-io/bbolt"
	// "github.com/boltdb/bolt"
)

type ClientType = uint16

const sizeOfClientsType = 16

//ClientGenerator - Функции генерации
type IClientGenerator interface {
	GenerateRandomClient(T ClientType, hash string) (*dto.ClientDescriptor, error)
	GenerateClient(T ClientType, name, hash string) (*dto.ClientDescriptor, error)
}

type DB interface {
	IClientGenerator
	IClient
	IC2cLimits
	IMessage
	IPerm
	ForEach(tableName string, callBack func(key []byte, value []byte) error)
}

type boltC2cDatabase struct {
	db *bolt.DB
	ClientImpl
	C2cLimitsImpl
	Messages
	PermImpl
}

var database boltC2cDatabase

// GetBoltDbInstance - Вернет реализацию интерфейса C2cDB реализованную на базе boltDB
func GetBoltDbInstance() DB {
	return &database
}

// InitC2cDB - create bolt database
func InitC2cDB() *bolt.DB {
	res := cf.GetConfigValueOrDefault("C2cStore", "./c2c.db")
	var err error
	database.db, err = bolt.Open(res, 0600, nil)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	// Create bucket if not exist
	database.db.Update(func(tx *bolt.Tx) error {
		getBucket(tx, Names)
		getBucket(tx, Clients)
		getBucket(tx, MaxClientID)
		getBucket(tx, ClientLimits)
		getBucket(tx, Permission)
		return nil
	})
	database.clientStorage = database.db
	database.limitStorage = database.db
	database.messageStorage = database.db
	database.permStorage = database.db
	log.Info("Init database finished fine")
	return database.db
}

func (d *boltC2cDatabase) ForEach(tableName string, callBack func(key []byte, value []byte) error) {
	d.db.View(
		func(tx *bolt.Tx) error {
			if buck, err := getBucket(tx, tableName); err == nil {
				buck.ForEach(callBack)
			}
			return nil
		})
}

func (d *boltC2cDatabase) getMaxID(T ClientType) uint64 {
	log.Tracef("Try find maxID device for %d", T)
	tx, err := d.db.Begin(true)
	if err != nil {
		log.Error(err.Error())
		return 0
	}
	defer func() {
		if tx.DB() != nil {
			tx.Rollback()
		}
	}()
	buck, err := getBucket(tx, MaxClientID)
	if err != nil {
		log.Error(err.Error())
		return 0
	}
	maxID := uint64(T)<<(64-sizeOfClientsType) | 1
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, T)
	if bID := buck.Get(buf); bID != nil {
		mxID := bytesToUint64(bID)
		if mxID >= maxID {
			maxID = mxID + 1
			log.Tracef("Max ID finded %d, %v", maxID, bID)
		} else {
			log.Errorf("Incorrect max ID %d, set default value %d", mxID, maxID)
		}
	}
	err = buck.Put(buf, uint64ToBytes(maxID))
	if err != nil {
		log.Error(err.Error())
		return 0
	}
	tx.Commit()
	return maxID
}

// GenerateRandomClient - Генерируем нового клиента, имя которого будет совпадать с его идентификационным номером
func (d *boltC2cDatabase) GenerateRandomClient(T ClientType, hash string) (*dto.ClientDescriptor, error) {
	if len(hash) < 2 {
		return nil, errors.New("hash password is to small")
	}
	max := d.getMaxID(T)
	if max != 0 {
		return &dto.ClientDescriptor{
			ID:        max,
			Name:      strconv.FormatUint(max, 16),
			SecretKey: hash,
		}, nil
	}
	return nil, fmt.Errorf("Can not generate new client undefined maxID for client type %d", T)
}

// GenerateClient - Генерируем нового клиента по его имени и паролю
func (d *boltC2cDatabase) GenerateClient(T ClientType, name, hash string) (*dto.ClientDescriptor, error) {
	log.Tracef("Generate new client for type %d", T)
	if _, er := d.getIdByName(name); er == nil {
		return nil, fmt.Errorf("Client with name %s already exist", name)
	}
	max := d.getMaxID(T)
	log.Tracef("New ID is %d", max)
	if max != 0 {
		return &dto.ClientDescriptor{
			ID:        max,
			Name:      name,
			SecretKey: hash,
		}, nil
	}
	return nil, errors.New("Can not generate new client undefined maxID")
}
