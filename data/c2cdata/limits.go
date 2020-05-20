package c2cdata

import (
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
)

// IC2cLimits - Интерфейс для статистики по клиенту
type IC2cLimits interface {
	GetStat(ID uint64) (dto.ClientLimits, error)
	UpdateStat(cl *dto.ClientLimits) error
	UpdateIfNotModified(cl *dto.ClientLimits) error
}

type C2cLimitsImpl struct {
	limitStorage *bolt.DB
}

func (d *C2cLimitsImpl) UpdateIfNotModified(cl *dto.ClientLimits) error {
	tx, e := d.limitStorage.Begin(true)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	buck, err := getBucket(tx, ClientLimits)
	if err != nil {
		return err
	}
	var oldClient dto.ClientLimits
	err = json.Unmarshal(buck.Get(uint64ToBytes(cl.ID)), &oldClient)
	if err != nil {
		return err
	}
	if oldClient.ModifiedDate.After(cl.ModifiedDate) {
		return errors.New("Client already modified")
	}
	cl.ModifiedDate = time.Now()
	data, err := json.Marshal(cl)
	if err != nil {
		return err
	}
	buck.Put(uint64ToBytes(cl.ID), data)
	return tx.Commit()
}

func (d *C2cLimitsImpl) GetStat(ID uint64) (dto.ClientLimits, error) {
	var res []byte
	er := d.limitStorage.View(
		func(tx *bolt.Tx) error {
			buck, err := getBucket(tx, ClientLimits)
			if err != nil {
				return err
			}
			res = buck.Get(uint64ToBytes(ID))
			if res == nil {
				return fmt.Errorf("Undefine client %x in stat storage", ID)
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

func (d *C2cLimitsImpl) UpdateStat(cl *dto.ClientLimits) error {
	cl.ModifiedDate = time.Now()
	data, err := json.Marshal(cl)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return d.limitStorage.Update(
		func(tx *bolt.Tx) error {
			buck, err := getBucket(tx, ClientLimits)
			if err != nil {
				return err
			}
			return buck.Put(uint64ToBytes(cl.ID), data)
		})
}
