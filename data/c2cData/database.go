package c2cData

import (
	cf "blabu/c2cService/configuration"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
)

//C2cDB - БАЗОВЫЙ интерфейс для клиент-клиент взаимодействия (Сделан для тестов)
type C2cDB interface {
	GetClientByName(name string) (*dto.ClientDescriptor, error)
	GetClientByID(ID uint64) (*dto.ClientDescriptor, error)
	DelClient(ID uint64, name string) error
	GenerateRandomClient(hash string) (*dto.ClientDescriptor, error)
	GenerateClient(name, hash string) (*dto.ClientDescriptor, error)
	SaveClient(cl *dto.ClientDescriptor) error
}

type boltC2cDatabase struct {
	clientStorage *bolt.DB
}

var database boltC2cDatabase

const (
	clientsByNameBucket = "clientsByName" // Клиенты с ключем по имени
	clientsByIDBucket   = "clientsByID"   // Клиенты с ключем по ID
	maxClientID         = "maxClientID"
	// clientMessages      = "client%sMessages" // Шаблон для таблиц с сообщениями пользователей
)

// GetBoltDbInstance - Вернет реализацию интерфейса C2cDB реализованную на базе boltDB
func GetBoltDbInstance() C2cDB {
	return &database
}

// InitC2cDB - create bolt database
func InitC2cDB() *bolt.DB {
	res := cf.GetConfigValueOrDefault("c2cStore", "./c2c.db")
	var err error
	database.clientStorage, err = bolt.Open(res, 0600, nil)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	// Create bucket if not exist
	database.clientStorage.Update(func(tx *bolt.Tx) error {
		database.getBucket(tx, clientsByNameBucket)
		database.getBucket(tx, clientsByIDBucket)
		database.getBucket(tx, maxClientID)
		return nil
	})
	return database.clientStorage
}

func (d *boltC2cDatabase) delClient(id, name []byte) error {
	er := d.clientStorage.View(
		func(tx *bolt.Tx) error {
			clientsByName, e1 := d.getBucket(tx, clientsByNameBucket)
			if e1 != nil {
				return e1
			}
			clientsByID, e2 := d.getBucket(tx, clientsByIDBucket)
			if e2 != nil {
				return e2
			}
			if err := clientsByID.Delete(id); err != nil {
				return err
			}
			return clientsByName.Delete(name)
		})
	return er
}

func (d *boltC2cDatabase) getClient(key []byte, bucketName string) (*dto.ClientDescriptor, error) {
	var result []byte
	er := d.clientStorage.View(
		func(tx *bolt.Tx) error {
			buck, err := d.getBucket(tx, bucketName)
			if err != nil {
				return err
			}
			result = buck.Get(key)
			if result == nil || len(result) == 0 {
				err := fmt.Errorf("Undefined client")
				log.Warning(err.Error())
				return err
			}
			return nil
		})
	if er != nil {
		return nil, er
	}
	return deserialize(result), nil
}

func (d *boltC2cDatabase) GetClientByName(name string) (*dto.ClientDescriptor, error) {
	return d.getClient([]byte(name), clientsByNameBucket)
}

func (d *boltC2cDatabase) GetClientByID(ID uint64) (*dto.ClientDescriptor, error) {
	return d.getClient(uint64ToBytes(ID), clientsByIDBucket)
}

func (d *boltC2cDatabase) DelClient(ID uint64, name string) error {
	return d.delClient(uint64ToBytes(ID), []byte(name))
}

func (d *boltC2cDatabase) getMaxID(T byte) uint64 {
	tx, err := d.clientStorage.Begin(true)
	if err != nil {
		log.Error(err.Error())
		return 0
	}
	defer func() {
		if tx.DB() != nil {
			tx.Rollback()
		}
	}()
	buck, err := d.getBucket(tx, maxClientID)
	if err != nil {
		log.Error(err.Error())
		return 0
	}
	maxID := uint64(T)<<56 | 1
	if bID := buck.Get([]byte{T}); bID != nil {
		maxID = bytesToUint64(bID)
		log.Tracef("Max ID finded %d, %v", maxID, bID)
		maxID++
	}
	err = buck.Put([]byte{T}, uint64ToBytes(maxID))
	if err != nil {
		log.Error(err.Error())
		return 0
	}
	tx.Commit()
	return maxID
}

// GenerateRandomClient - Генерируем нового клиента, имя которого будет совпадать с его идентификационным номером
func (d *boltC2cDatabase) GenerateRandomClient(hash string) (*dto.ClientDescriptor, error) {
	if len(hash) < 2 {
		return nil, fmt.Errorf("hash password is to small")
	}
	max := d.getMaxID(1)
	if max != 0 {
		return &dto.ClientDescriptor{
			ID:        max,
			Name:      strconv.FormatUint(max, 16),
			SecretKey: hash,
		}, nil
	}
	return nil, fmt.Errorf("Can not generate new client undefined maaxID")
}

// GenerateClient - Генерируем нового клиента по его имени и паролю
func (d *boltC2cDatabase) GenerateClient(name, hash string) (*dto.ClientDescriptor, error) {
	if _, er := d.GetClientByName(name); er == nil {
		return nil, fmt.Errorf("Client with name %s already exist", name)
	}
	max := d.getMaxID(1)
	log.Tracef("New ID is %d", max)
	if max != 0 {
		return &dto.ClientDescriptor{
			ID:        max,
			Name:      name,
			SecretKey: hash,
		}, nil
	}
	return nil, fmt.Errorf("Can not generate new client undefined maxID")
}

//SaveClient - Сохраняем нового клиента на диск.
func (d *boltC2cDatabase) SaveClient(cl *dto.ClientDescriptor) error {
	cl.RegisterDate = time.Now()
	if cl.ID == 0 {
		er := fmt.Errorf("Can not save client with id = 0")
		log.Error(er)
		return er
	}
	er := d.clientStorage.Update(func(tx *bolt.Tx) error {
		clientByID, er := d.getBucket(tx, clientsByIDBucket)
		if er != nil {
			return er
		}
		clientsByName, er := d.getBucket(tx, clientsByNameBucket)
		if er != nil {
			return er
		}
		if er = clientByID.Put(uint64ToBytes(cl.ID), serialize(cl)); er != nil {
			return fmt.Errorf("Can not save client ID, incorrect %d", cl.ID)
		}
		if er = clientsByName.Put([]byte(cl.Name), serialize(cl)); er != nil {
			return fmt.Errorf("Can not save client Name incorrect %s", cl.Name)
		}
		return nil
	})
	return er
}

func (d *boltC2cDatabase) getBucket(tx *bolt.Tx, bucketName string) (*bolt.Bucket, error) {
	var buck *bolt.Bucket
	if buck = tx.Bucket([]byte(bucketName)); buck == nil {
		log.Warning("Can not find bucket for clients")
		log.Trace("Try create client bucket")
		var er error
		if buck, er = tx.CreateBucket([]byte(bucketName)); er != nil {
			log.Error(er.Error())
			return nil, fmt.Errorf("Can not create bucket for clients")
		}
		log.Tracef("Bucket %s created", bucketName)
	}
	return buck, nil
}

func uint64ToBytes(val uint64) []byte {
	res := make([]byte, 8)
	binary.LittleEndian.PutUint64(res, val)
	return res
}

func bytesToUint64(bytes []byte) uint64 {
	if len(bytes) != 8 {
		log.Errorf("Invalid uin64 type %v", bytes)
		return 0
	}
	return binary.LittleEndian.Uint64(bytes)
}

func serialize(c *dto.ClientDescriptor) []byte {
	res, _ := json.Marshal(c)
	return res
}

func deserialize(dat []byte) *dto.ClientDescriptor {
	var cl dto.ClientDescriptor
	json.Unmarshal(dat, &cl)
	return &cl
}
