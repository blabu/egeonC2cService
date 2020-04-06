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

type ClientType = uint16
const sizeOfClientsType = 16

//ClientGenerator - Функции генерации
type ClientGenerator interface {
	GenerateRandomClient(T ClientType, hash string) (*dto.ClientDescriptor, error)
	GenerateClient(T ClientType, name, hash string) (*dto.ClientDescriptor, error)
}

//C2cDB - БАЗОВЫЙ интерфейс для клиент-клиент взаимодействия (Сделан для тестов)
type C2cDB interface {
	GetClient(ID uint64) (*dto.ClientDescriptor, error)
	DelClient(ID uint64) error
	GetClientID(name string) (uint64, error)
	SaveClient(cl *dto.ClientDescriptor) error
}

// C2cStat - Интерфейс для статистики по клиенту
type C2cLimits interface {
	GetStat(ID uint64) (dto.ClientLimits, error)
	UpdateStat(cl *dto.ClientLimits) error
	UpdateIfNotModified(cl *dto.ClientLimits) error
}

type DB interface {
	C2cDB
	ClientGenerator
	C2cLimits
	ForEach(tableName string, callBack func(key []byte, value []byte)error)
}

type boltC2cDatabase struct {
	clientStorage *bolt.DB
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
	database.clientStorage, err = bolt.Open(res, 0600, nil)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	// Create bucket if not exist
	database.clientStorage.Update(func(tx *bolt.Tx) error {
		database.getBucket(tx, Names)
		database.getBucket(tx, Clients)
		database.getBucket(tx, MaxClientID)
		database.getBucket(tx,ClientLimits)
		database.getBucket(tx,Permission)
		return nil
	})
	return database.clientStorage
}

func (d *boltC2cDatabase) ForEach(tableName string, callBack func(key []byte, value []byte)error) {
	d.clientStorage.View(
		func(tx *bolt.Tx) error {
			if buck,err := d.getBucket(tx, tableName); err == nil {
				buck.ForEach(callBack)
			}
			return nil
		})
}

func (d *boltC2cDatabase) UpdateIfNotModified(cl *dto.ClientLimits) error {
	tx, e := d.clientStorage.Begin(true)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	buck, err := d.getBucket(tx, ClientLimits)
	if err != nil {
		return err
	}
	var oldClient dto.ClientLimits
	err = json.Unmarshal(buck.Get(uint64ToBytes(cl.ID)), &oldClient)
	if err != nil {
		return err
	}
	if oldClient.ModifiedDate.After(cl.ModifiedDate) {
		return fmt.Errorf("Client already modified")
	}
	cl.ModifiedDate = time.Now()
	data, err := json.Marshal(cl)
	if err != nil {
		return err
	}
	buck.Put(uint64ToBytes(cl.ID), data)
	return tx.Commit()
}

func (d *boltC2cDatabase) GetStat(ID uint64) (dto.ClientLimits, error) {
	var res []byte
	er := d.clientStorage.View(
		func(tx *bolt.Tx)error {
			buck, err := d.getBucket(tx, ClientLimits)
			if err != nil {
				return err
			}
			res = buck.Get(uint64ToBytes(ID))
			if res == nil {
				return fmt.Errorf("Undefine client %d in stat storage", ID)
			}
			return nil
		})
	if er != nil {
		return dto.ClientLimits{ID: ID}, er
	}
	var stCl dto.ClientLimits
	er = json.Unmarshal(res, &stCl)
	return stCl, er
}

func (d *boltC2cDatabase) UpdateStat(cl *dto.ClientLimits) error {
	cl.ModifiedDate = time.Now()
	data, err := json.Marshal(cl)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return d.clientStorage.Update(
		func(tx *bolt.Tx) error {
			buck, err := d.getBucket(tx, ClientLimits)
			if err != nil {
				return err
			}
			return buck.Put(uint64ToBytes(cl.ID), data)
		})
}

func (d *boltC2cDatabase) delClient(id []byte) error {
	return d.clientStorage.Update(
		func(tx *bolt.Tx) error {
			Clients, e1 := d.getBucket(tx, Clients)
			if e1 != nil {
				return e1
			}
			Names, e2 := d.getBucket(tx, Names)
			if e2 != nil {
				return e2
			}
			if err := Names.Delete(id); err != nil {
				return err
			}
			return Clients.Delete(id)
	})
}

func (d *boltC2cDatabase) getIdByName(name string) (uint64, error) {
	var res []byte
	er := d.clientStorage.View(
		func(tx*bolt.Tx) error {
			buck, err := d.getBucket(tx, Names)
			if err != nil {
				return err
			}
			res = buck.Get([]byte(name))
			if res == nil || len(res) == 0 {
				err := fmt.Errorf("Undefined client with name %s", name)
				log.Warning(err.Error())
				return err
			}
			return nil
		})
		if er != nil {
			return 0, er
		}
		return bytesToUint64(res), nil
}

func (d *boltC2cDatabase) getClient(id []byte) (*dto.ClientDescriptor, error) {
	var result []byte
	er := d.clientStorage.View(
		func(tx *bolt.Tx) error {
			buck, err := d.getBucket(tx, Clients)
			if err != nil {
				return err
			}
			result = buck.Get(id)
			if result == nil || len(result) == 0 {
				err := fmt.Errorf("Undefined client with id %d", bytesToUint64(id))
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

func (d *boltC2cDatabase) GetClient(ID uint64) (*dto.ClientDescriptor, error) {
	return d.getClient(uint64ToBytes(ID))
}

func (d *boltC2cDatabase) GetClientID(name string) (uint64, error) {
	return d.getIdByName(name)
}

func (d *boltC2cDatabase) DelClient(ID uint64) error {
	return d.delClient(uint64ToBytes(ID))
}

func (d *boltC2cDatabase) getMaxID(T ClientType) uint64 {
	log.Tracef("Try find maxID device for %d", T)
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
	buck, err := d.getBucket(tx, MaxClientID)
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
			maxID = mxID+1
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
		return nil, fmt.Errorf("hash password is to small")
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
	er := d.clientStorage.Update(
		func(tx *bolt.Tx) error {
		Clients, er := d.getBucket(tx, Clients)
		if er != nil {
			return er
		}
		Names, er := d.getBucket(tx, Names)
		if er != nil {
			return er
		}
		if er = Clients.Put(uint64ToBytes(cl.ID), serialize(cl)); er != nil {
			return fmt.Errorf("Can not save client ID, incorrect %d", cl.ID)
		}
		if er = Names.Put([]byte(cl.Name), uint64ToBytes(cl.ID)); er != nil {
			return fmt.Errorf("Can not save client Name incorrect %s", cl.Name)
		}
		return nil
	})
	return er
}
