package c2cdata

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/blabu/c2cLib/dto"
	log "github.com/blabu/egeonC2cService/logWrapper"
	bolt "go.etcd.io/bbolt"
)

func getBucket(tx *bolt.Tx, bucketName string) (*bolt.Bucket, error) {
	var buck *bolt.Bucket
	if buck = tx.Bucket([]byte(bucketName)); buck == nil {
		log.Info("Can not find bucket ", bucketName)
		log.Trace("Try create client bucket ", bucketName)
		var er error
		if buck, er = tx.CreateBucket([]byte(bucketName)); er != nil {
			log.Error(er.Error())
			return nil, fmt.Errorf("Can not create bucket for clients")
		}
		log.Tracef("Bucket %s created", bucketName)
	}
	return buck, nil
}

func update(buckName []byte, db *bolt.DB, handler func(*bolt.Bucket) error) error {
	return db.Update(
		func(tx *bolt.Tx) error {
			buck, err := tx.CreateBucketIfNotExists(buckName)
			if err != nil {
				return err
			}
			return handler(buck)
		})
}

func view(buckName []byte, db *bolt.DB, handler func(*bolt.Bucket) error) error {
	return db.View(
		func(tx *bolt.Tx) error {
			buck := tx.Bucket(buckName)
			if buck == nil {
				return errors.New("Bucket does not exist")
			}
			return handler(buck)
		})
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
