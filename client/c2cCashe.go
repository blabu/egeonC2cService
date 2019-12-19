package client

import (
	log "blabu/c2cService/logWrapper"
	"fmt"
	"sync"
)

const maxCONNECTION = 16 /*Максимальное кол-во коннектов к одному клиенту*/

// Структура клиента (для хранения его в онлайн кеше)
type cachedClient struct {
	base             ClientListenerInterface   // Указатель на сам клиент.
	connectedReaders []ClientListenerInterface // Список указателей на всех клиентов, которые читают нужен для удаления данного клиента как публикующего данные у его читателей
	mtx              *sync.RWMutex             // Для модификации connectedReaders
}

//ConnectionCache - кеш всех подключений
type ConnectionCache struct {
	onlineClientsCashe map[uint64]cachedClient
	ml                 sync.RWMutex
}

// NewConnectionCache - Создает новый потокобезопасный кеш соединений
func NewConnectionCache() ConnectionCache {
	return ConnectionCache{
		onlineClientsCashe: make(map[uint64]cachedClient),
	}
}

// AddClientToCache - check if client does not exist create all needed meta data and add him to online cache store
func (con ConnectionCache) AddClientToCache(devID uint64, cl ClientListenerInterface) error {
	if cl != nil {
		con.ml.Lock()
		defer con.ml.Unlock()
		_, ok := con.onlineClientsCashe[devID]
		if ok {
			log.Warning("Can not append new abonent with diviceID ", devID, " abonent exist")
			return fmt.Errorf("Abonent exist")
		}
		allReaders := make([]ClientListenerInterface, 0, maxCONNECTION) // Для избежания случайных переалокаций
		con.onlineClientsCashe[devID] = cachedClient{
			base:             cl,
			connectedReaders: allReaders,
			mtx:              new(sync.RWMutex),
		}
		return nil
	}
	return fmt.Errorf("Client is nil")
}

// DelClientFromCashe - delete this client from all connected to him valid devices
// and than delete this client from online cache store
func (con ConnectionCache) DelClientFromCashe(devID uint64) {
	con.ml.Lock()
	defer con.ml.Unlock()
	clientBase, ok := con.onlineClientsCashe[devID]
	if ok { // If abonent exist
		clientBase.mtx.RLock() // iterate by connectedReaders
		for _, val := range clientBase.connectedReaders {
			if val != nil {
				val.DelListener(devID)
			}
		}
		clientBase.mtx.RUnlock()
		delete(con.onlineClientsCashe, devID)
	} else {
		log.Errorf("Client with id %d not find in cache for delete it", devID)
	}
}

// DisconnectClient - close connection from devTo and devFrom
func (con ConnectionCache) DisconnectClient(devTo, devFrom uint64) error {
	con.ml.RLock()
	cTo, ok1 := con.onlineClientsCashe[devTo]
	cFrom, ok2 := con.onlineClientsCashe[devFrom]
	con.ml.RUnlock()
	if !ok1 || !ok2 || cTo.base == nil || cFrom.base == nil {
		err := fmt.Errorf("Some of clients is undefined %d, %d", devTo, devFrom)
		log.Warning(err.Error())
		return err
	}
	if cFrom.connectedReaders == nil || cTo.connectedReaders == nil {
		err := fmt.Errorf("Some of clients is invalid %d, %d", devTo, devFrom)
		log.Warning(err.Error())
		return err
	}
	cFrom.base.DelListener(devTo)
	cTo.base.DelListener(devFrom)
	return nil
}

// ConnectClients - Создает соединение и регистрирует его в кеше
func (con ConnectionCache) ConnectClients(devTo, devFrom uint64) error {
	con.ml.RLock()
	cTo, ok1 := con.onlineClientsCashe[devTo]
	cFrom, ok2 := con.onlineClientsCashe[devFrom]
	con.ml.RUnlock()
	if !ok1 || cTo.base == nil || cFrom.connectedReaders == nil {
		if !ok1 {
			log.Warningf("Client %d not find in online cache", devTo)
		} else if cTo.base == nil {
			log.Errorf("Client %d in online cache is nil", devTo)
		} else if cFrom.connectedReaders == nil {
			log.Errorf("Client %d in online cache is not correct", devFrom)
		}
		return fmt.Errorf("Can not create connection to device %d", devTo)
	}
	if !ok2 || cFrom.base == nil || cTo.connectedReaders == nil {
		if !ok2 {
			log.Warningf("Client %d not find in online cache", devFrom)
		} else if cFrom.base == nil {
			log.Errorf("Client %d in online cache is nil", devFrom)
		} else if cFrom.connectedReaders == nil {
			log.Errorf("Client from %d in online cache is not correct", devFrom)
		}
		return fmt.Errorf("Can not create connection from device %d", devTo)
	}
	cFrom.base.AddListener(devTo, cTo.base.GetListenerChan())
	cTo.base.AddListener(devFrom, cFrom.base.GetListenerChan())
	cTo.mtx.Lock()
	cTo.connectedReaders = append(cTo.connectedReaders, cFrom.base)
	cTo.mtx.Unlock()
	cFrom.mtx.Lock()
	cFrom.connectedReaders = append(cFrom.connectedReaders, cTo.base)
	cFrom.mtx.Unlock()
	con.ml.Lock()
	con.onlineClientsCashe[devTo] = cTo
	con.onlineClientsCashe[devFrom] = cFrom
	con.ml.Unlock()
	return nil
}
