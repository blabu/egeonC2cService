package c2cdata

import (
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"errors"
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
)

//IClient - БАЗОВЫЙ интерфейс для клиент-клиент взаимодействия (Сделан для тестов)
type IClient interface {
	GetClient(ID uint64) (*dto.ClientDescriptor, error)
	DelClient(ID uint64) error
	GetClientID(name string) (uint64, error)
	SaveClient(cl *dto.ClientDescriptor) error
}

type ClientImpl struct {
	clientStorage *bolt.DB
}

func (d *ClientImpl) delClient(id []byte) error {
	return d.clientStorage.Update(
		func(tx *bolt.Tx) error {
			Clients, e1 := getBucket(tx, Clients)
			if e1 != nil {
				return e1
			}
			Names, e2 := getBucket(tx, Names)
			if e2 != nil {
				return e2
			}
			if err := Names.Delete(id); err != nil {
				return err
			}
			return Clients.Delete(id)
		})
}

func (d *ClientImpl) getIdByName(name string) (uint64, error) {
	if d.clientStorage == nil {
		log.Fatal("Client storage is nill")
	}
	var res []byte
	er := d.clientStorage.View(
		func(tx *bolt.Tx) error {
			buck, err := getBucket(tx, Names)
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

func (d *ClientImpl) getClient(id []byte) (*dto.ClientDescriptor, error) {
	var result []byte
	er := d.clientStorage.View(
		func(tx *bolt.Tx) error {
			buck, err := getBucket(tx, Clients)
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

func (d *ClientImpl) GetClient(ID uint64) (*dto.ClientDescriptor, error) {
	return d.getClient(uint64ToBytes(ID))
}

func (d *ClientImpl) GetClientID(name string) (uint64, error) {
	return d.getIdByName(name)
}

func (d *ClientImpl) DelClient(ID uint64) error {
	return d.delClient(uint64ToBytes(ID))
}

//SaveClient - Сохраняем нового клиента на диск.
func (d *ClientImpl) SaveClient(cl *dto.ClientDescriptor) error {
	if cl == nil {
		return errors.New("Incorrect client data")
	}
	cl.RegisterDate = time.Now()
	if cl.ID == 0 {
		er := errors.New("Can not save client with id = 0")
		log.Error(er)
		return er
	}
	er := d.clientStorage.Update(
		func(tx *bolt.Tx) error {
			Clients, er := getBucket(tx, Clients)
			if er != nil {
				return er
			}
			Names, er := getBucket(tx, Names)
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
