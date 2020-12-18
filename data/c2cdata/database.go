package c2cdata

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"unsafe"

	"github.com/blabu/c2cLib/dto"
	cf "github.com/blabu/egeonC2cService/configuration"
	"github.com/blabu/egeonC2cService/data"
	log "github.com/blabu/egeonC2cService/logWrapper"

	bolt "go.etcd.io/bbolt"
)

type boltC2cDatabase struct {
	db *bolt.DB
	ClientImpl
	Messages
}

var database boltC2cDatabase

// GetBoltDbInstance - Вернет реализацию интерфейса DB реализованную на базе boltDB
func GetBoltDbInstance() data.DB {
	return &database
}

// InitC2cDB - create bolt database
func InitC2cDB() *bolt.DB {
	res := cf.Config.C2cStore
	if len(res) == 0 {
		res = "./c2c.db"
	}
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
		return nil
	})
	database.clientStorage = database.db
	database.messageStorage = database.db
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

func (d *boltC2cDatabase) getMaxID(T data.ClientType) uint64 {
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
	maxID := uint64(T)<<(64-(unsafe.Sizeof(T)*8)) | 1
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(T))
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
func (d *boltC2cDatabase) GenerateRandomClient(T data.ClientType, hash string) (*dto.ClientDescriptor, error) {
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
func (d *boltC2cDatabase) GenerateClient(T data.ClientType, name, hash string) (*dto.ClientDescriptor, error) {
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
