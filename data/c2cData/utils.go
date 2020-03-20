package c2cData

import (
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
)

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
