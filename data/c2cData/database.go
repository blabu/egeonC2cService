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
type C2cStat interface {
	GetStat(ID uint64) (dto.ClientStat, error)
	UpdateStat(cl *dto.ClientStat) error
}

type DB interface {
	C2cDB
	ClientGenerator
	C2cStat
	AddNewLimitation(ID uint64, timePeriod time.Duration, maxTxBytes, maxRxBytes uint64)
}

type boltC2cDatabase struct {
	clientStorage *bolt.DB
}

var database boltC2cDatabase

const (
	names       = "nameByID" // список имен с ключем по ID
	clients     = "clients" // Непосредственно сами клиенты с ключем по ID
	maxClientID = "maxClientID" // Максимально выданный в системе идентификатор
	clientStat  = "clientStatistics" // Все отправленные сообщения от клиентов
)

// GetBoltDbInstance - Вернет реализацию интерфейса C2cDB реализованную на базе boltDB
func GetBoltDbInstance() DB {
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
		database.getBucket(tx, names)
		database.getBucket(tx, clients)
		database.getBucket(tx, maxClientID)
		database.getBucket(tx,clientStat)
		return nil
	})
	return database.clientStorage
}

func (d *boltC2cDatabase) GetStat(ID uint64) (dto.ClientStat, error) {
	var res []byte
	er := d.clientStorage.View(
		func(tx *bolt.Tx)error {
			buck, err := d.getBucket(tx, clientStat)
			if err != nil {
				return err
			}
			res = buck.Get(uint64ToBytes(ID))
			return nil
		})
	if er != nil {
		return dto.ClientStat{}, er
	}
	var stCl dto.ClientStat
	er = json.Unmarshal(res, &stCl)
	return stCl, er
}

func (d *boltC2cDatabase) UpdateStat(cl *dto.ClientStat) error {
	data, err := json.Marshal(cl)
	if err != nil {
		return err
	}
	return d.clientStorage.Update(
		func(tx *bolt.Tx) error {
			buck, err := d.getBucket(tx, clientStat)
			if err != nil {
				return err
			}
			return buck.Put(uint64ToBytes(cl.ID), data)
		})
}

func (d* boltC2cDatabase) AddNewLimitation(ID uint64, timePeriod time.Duration, maxTxBytes, maxRxBytes uint64) {
	d.clientStorage.Update(
		func(tx *bolt.Tx) error {
			buck, err := d.getBucket(tx, clientStat)
			if err != nil {
				return err
			}
			var res dto.ClientStat
			if err = json.Unmarshal(buck.Get(uint64ToBytes(ID)), &res); err != nil {
				return err
			}
			res.TimePeriod = timePeriod
			res.MaxReceivedBytes = maxRxBytes
			res.MaxTransmittedBytes = maxTxBytes
			res.LimitExpiration = time.Now().Add(res.TimePeriod)
			if data, err := json.Marshal(res); err != nil {
				return err
			} else {
				return buck.Put(uint64ToBytes(ID), data)
			}
		})
}

func (d *boltC2cDatabase) delClient(id []byte) error {
	return d.clientStorage.Update(
		func(tx *bolt.Tx) error {
			Clients, e1 := d.getBucket(tx, clients)
			if e1 != nil {
				return e1
			}
			Names, e2 := d.getBucket(tx, names)
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
			buck, err := d.getBucket(tx, names)
			if err != nil {
				return err
			}
			res = buck.Get([]byte(name))
			if res == nil || len(res) == 0 {
				err := fmt.Errorf("Undefined client")
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
			buck, err := d.getBucket(tx, clients)
			if err != nil {
				return err
			}
			result = buck.Get(id)
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
		Clients, er := d.getBucket(tx, clients)
		if er != nil {
			return er
		}
		Names, er := d.getBucket(tx, names)
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
