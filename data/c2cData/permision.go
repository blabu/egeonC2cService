package c2cData

import (
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
)

type IPerm interface {
	GetPermission(key string) (*dto.ClientPermission, error)
	UpdatePermission(dto.ClientPermission) error
}

func (b *boltC2cDatabase) GetPermission(key string) (*dto.ClientPermission, error) {
	var perm dto.ClientPermission
	err := b.clientStorage.View(
		func(tx *bolt.Tx) error {
			if buck, err := b.getBucket(tx, Permission); err != nil {
				return err
			} else {
				data := buck.Get([]byte(key))
				return json.Unmarshal(data, &perm)
			}
		})
	if err != nil {
		log.Warningf(err.Error())
		err = fmt.Errorf("Undefine token %s", key)
		return nil, err
	}
	return &perm, nil
}

func (b *boltC2cDatabase) UpdatePermission(perm dto.ClientPermission) error {
	data, err := json.Marshal(perm)
	if err != nil {
		return err
	}
	return b.clientStorage.Update(
		func(tx *bolt.Tx) error {
			if buck, err := b.getBucket(tx, Permission); err != nil {
				return err
			} else {
				return buck.Put([]byte(perm.Key), data)
			}
		})
}
