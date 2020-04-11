package httpGateway

import (
	cf "blabu/c2cService/configuration"
	"blabu/c2cService/data/c2cData"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

func getClientPermission(key string) (*dto.ClientPermission, error) {
	if rootKey, err := cf.GetConfigValue("RootKey"); err == nil {
		if rootKey == key {
			return &dto.ClientPermission{
				Name: "root",
				Key:  rootKey,
				Perm: []dto.Permission{
					dto.Permission{
						URL:        "{any}",
						IsWritable: true,
					}},
			}, nil
		}
	}
	db := c2cData.GetBoltDbInstance()
	if Perm, ok := db.(c2cData.IPerm); !ok {
		return &dto.ClientPermission{}, errors.New("Can not find permission")
	} else {
		return Perm.GetPermission(key)
	}
}

func checkKey(key, path string) (dto.Permission, error) {
	p, err := getClientPermission(key)
	if err != nil {
		return dto.Permission{}, err
	}
	if len(p.Perm) > 0 && p.Perm[0].URL == "{any}" && p.Name == "root" {
		return dto.Permission{
			URL:        path,
			IsWritable: true,
		}, nil
	}
	url := strings.TrimPrefix(path, apiLevel)
	for _, v := range p.Perm {
		if v.URL == url {
			return v, nil
		}
		log.Tracef("Url %s not equal requested %s", v.URL, url)
	}
	return dto.Permission{}, errors.New("Operation not permitted")
}

func readNotExpireFile(path string, expire time.Duration) ([]byte, error) {
	baseDir, err := os.Open(path)
	if err != nil {
		log.Info("File read error ", err.Error())
		return nil, err
	}
	defer baseDir.Close()
	if info, er := baseDir.Stat(); er == nil {
		if time.Now().After(info.ModTime().Add(expire)) {
			return nil, errors.New("Time expire")
		}
	}
	return ioutil.ReadAll(baseDir)
}

func readFile(path string) ([]byte, error) {
	baseDir, err := os.Open(path)
	if err != nil {
		log.Info("File read error ", err.Error())
		return nil, err
	}
	defer baseDir.Close()
	return ioutil.ReadAll(baseDir)
}

var fileSystemMtx sync.Mutex

func writeToFile(data []byte, pathToFile string) error {
	fileSystemMtx.Lock()
	defer fileSystemMtx.Unlock()
	dir := path.Dir(pathToFile)
	er := os.MkdirAll(dir, os.ModePerm)
	if er != nil {
		log.Debugf("Error %s when try create new folder %s", er.Error(), dir)
		return er
	}
	f, err := os.Create(pathToFile)
	if err != nil {
		return err
	}
	log.Debug("Create new tile file at path ", pathToFile)
	f.Write(data)
	f.Close()
	return nil
}

func getClientLimit(r *http.Request) (bool, dto.ClientLimits, error) {
	key := r.URL.Query().Get("key")
	perm, err := checkKey(key, limits)
	if err != nil {
		return false, dto.ClientLimits{}, errors.New("Operation not permitted")
	}
	storage := c2cData.GetBoltDbInstance()
	var id uint64
	idStr := r.URL.Query().Get("id")
	if len(idStr) == 0 {
		name := r.URL.Query().Get("name")
		if id, err = storage.GetClientID(name); err != nil {
			return perm.IsWritable, dto.ClientLimits{}, err
		}
	} else {
		if id, err = strconv.ParseUint(idStr, 10, 64); err != nil {
			return perm.IsWritable, dto.ClientLimits{}, err
		}
	}
	l, e := storage.GetStat(id)
	return perm.IsWritable, l, e
}
