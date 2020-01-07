package c2cService

import (
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

/*
Приватные функции для клиент-клиент взаимодействия
Предполагается что параметры всех функций проверены на nil и размер
*/

const answerInitByNameOk string = "INIT OK"
const answerConnectByNameOk string = "CONNECT OK"

const answerInitByIDOk string = "0"
const answerConnectByIDOk string = "0"

const saltUniqCount = 3 /*Количество раз использования случайной соли при авторизации*/

func (c *C2cDevice) ping(m *dto.Message) error {
	if c.device.ID != 0 {
		currTimeStr := strconv.FormatInt(time.Now().Unix(), 16)
		c.readChan <- dto.Message{
			Command: pingCOMMAND,
			Proto:   m.Proto,
			Jmp:     m.Jmp,
			Content: []dto.Content{
				dto.Content{Data: []byte("0")},
				m.Content[0], // To abonent
				dto.Content{Data: []byte(strings.ToUpper(currTimeStr))},
			},
		}
		return nil
	}
	log.Errorf("PING error. Undefined client in session %d", c.sessionID)
	return fmt.Errorf("Undefined client. Initialize first in session %d", c.sessionID)
}

func (c *C2cDevice) connectByID(m *dto.Message) error {
	if c.device.ID == 0 {
		return fmt.Errorf("Initialize device at first")
	}
	from, err := strconv.ParseUint(string(m.Content[0].Data), 16, 64)
	if err != nil {
		return err
	}
	if c.device.ID != uint64(from) {
		log.Warningf("session %d client ID in request command is incorrect originID %d != requestedID %d", c.sessionID, c.device.ID, from)
		return fmt.Errorf("Incorrect client id")
	}
	to, err := strconv.ParseUint(string(m.Content[1].Data), 16, 64)
	if err != nil {
		return err
	}
	log.Tracef("SessionID %d Connect command from device %s to device %s", c.sessionID, from, to)
	if err = connection.ConnectClients(to, from); err != nil {
		log.Warning(err.Error())
		return fmt.Errorf("Can not create connection whith abonnent %d", to)
	}
	c.readChan <- dto.Message{
		Command: connectByIDCOMMAND,
		Jmp:     m.Jmp,
		Proto:   m.Proto,
		Content: []dto.Content{
			m.Content[1], // FROM remote client
			m.Content[0], // TO  local client
			dto.Content{Data: []byte(answerConnectByIDOk)},
		},
	}
	return nil
}

func (c *C2cDevice) connectByName(m *dto.Message) error {
	if c.device.ID == 0 {
		return fmt.Errorf("Initialize device at first in session %d", c.sessionID)
	}
	from := string(m.Content[0].Data)
	if c.device.Name != from {
		return fmt.Errorf("Incorrect device name %s != %s in session %d", c.device.Name, from, c.sessionID)
	}
	toClient, err := c.storage.GetClientByName(string(m.Content[1].Data))
	if err != nil {
		return err
	}
	if err := connection.ConnectClients(toClient.ID, c.device.ID); err != nil {
		log.Warning(err.Error())
		return fmt.Errorf("Can not create connection with abonnent %s in session %d", toClient, c.sessionID)
	}
	log.Tracef("Connect command from device %s to device %s in session %d", from, toClient, c.sessionID)
	c.readChan <- dto.Message{
		Command: connectByNameCOMMAND,
		Jmp:     m.Jmp,
		Proto:   m.Proto,
		Content: []dto.Content{
			m.Content[1],
			m.Content[0],
			dto.Content{Data: []byte(answerConnectByNameOk)},
		},
	}
	return nil
}

// For init by ID you need send ID (Content[0]), (salt ; signature)-(Content[2]) signature-base64(SHA256(ID + salt + base64(SHA256(name+password))))
func (c *C2cDevice) initByID(m *dto.Message) error {
	id, err := strconv.ParseUint(string(m.Content[0].Data), 16, 64)
	if err != nil {
		return fmt.Errorf("Can not find corect ID in session %d", c.sessionID)
	}
	credentials := strings.Split(string(m.Content[2].Data), ";") // Разделим соль от подписи
	if len(credentials) < 2 {
		err := fmt.Errorf("Undefined signature for initialize in session %d", c.sessionID)
		log.Error(err.Error())
		return err
	}
	if CheckSalt(credentials[0]) > saltUniqCount {
		err := fmt.Errorf("Salt already been used %d times in session %d", saltUniqCount, c.sessionID)
		log.Error(err.Error())
		return err
	}
	if c.device.ID == 0 {
		device, err := c.storage.GetClientByID(id)
		if err != nil {
			log.Warning(err.Error())
			return err
		}
		c.device = *device
	}
	if c.device.ID == id {
		temp := sha256.Sum256([]byte(string(m.Content[0].Data) + credentials[0] + c.device.SecretKey))
		origin := base64.StdEncoding.EncodeToString(temp[:])
		if origin != credentials[1] {
			log.Warningf("Incorrect signature %s != %s in session %d", origin, credentials[1], c.sessionID)
			c.device.ID = 0
			c.device.Name = ""
			return fmt.Errorf("Initialize fail session %d", c.sessionID)
		}
		if er := connection.AddClientToCache(c.device.ID, c); er == nil {
			c.readChan <- dto.Message{
				Command: initByIDCOMMAND,
				Jmp:     m.Jmp,
				Proto:   m.Proto,
				Content: []dto.Content{
					dto.Content{Data: []byte("0")}, // FROM SERVER
					m.Content[0],                   // TO sended client
					dto.Content{Data: []byte(answerInitByIDOk)},
				},
			}
			return nil
		}
		log.Warningf("Credentials is equals TODO destroy old session with client %s id: %d", c.device.Name, c.device.ID)
		er := fmt.Errorf("Can not create abonent session %d", c.sessionID)
		log.Error(er.Error())
		return er
	}
	c.device.ID = 0
	c.device.Name = ""
	return fmt.Errorf("Incorrect ID in session %d", c.sessionID)
}

// For init by name you need send name (Content[0]), (salt ; signature)-(Content[2]) signature - base64(SHA256(name + salt + base64(SHA256(name+password))))
func (c *C2cDevice) initByName(m *dto.Message) error {
	name := string(m.Content[0].Data)
	credentials := strings.Split(string(m.Content[2].Data), ";") // Разделим соль от подписи
	if len(credentials) < 2 {
		err := fmt.Errorf("Undefined signature for initialize in session %d", c.sessionID)
		log.Error(err.Error())
		return err
	}
	if CheckSalt(credentials[0]) > saltUniqCount {
		err := fmt.Errorf("Salt already been used %d times", saltUniqCount)
		log.Error(err.Error())
		return err
	}
	if c.device.ID == 0 {
		device, err := c.storage.GetClientByName(name)
		if err != nil {
			log.Warning(err.Error())
			return err
		}
		c.device = *device
	}
	if c.device.Name == name {
		t := string(m.Content[0].Data) + credentials[0] + c.device.SecretKey
		temp := sha256.Sum256([]byte(t))
		origin := base64.StdEncoding.EncodeToString(temp[:])
		if origin != credentials[1] {
			log.Errorf("Origin credentials: %s", t)
			log.Errorf("SHA256: %x", temp)
			log.Errorf("Incorrect signature %s != %s in session %d", origin, credentials[1], c.sessionID)
			c.device.ID = 0
			c.device.Name = ""
			return fmt.Errorf("Initialize fail in session %d", c.sessionID)
		}
		if er := connection.AddClientToCache(c.device.ID, c); er == nil {
			c.readChan <- dto.Message{
				Command: initByNameCOMMAND,
				Jmp:     m.Jmp,
				Proto:   m.Proto,
				Content: []dto.Content{
					dto.Content{Data: []byte("0")},
					m.Content[0],
					dto.Content{Data: []byte(answerInitByNameOk)},
				},
			}
			return nil
		}
		log.Warningf("Credentials is equals TODO destroy old session with client %s id: %d", c.device.Name, c.device.ID)
		er := fmt.Errorf("Can not create abonent in session %d", c.sessionID)
		log.Error(er.Error())
		c.device.ID = 0
		c.device.Name = ""
		return er
	}
	c.device.ID = 0
	c.device.Name = ""
	err := fmt.Errorf("Incorrect name in session %d", c.sessionID)
	log.Error(err.Error())
	return err
}

// For registration new device you need send an unique name, and base64(sha256(name+password))
func (c *C2cDevice) registerNewDevice(m *dto.Message) error {
	if c.device.ID != 0 {
		err := fmt.Errorf("Client already exist error in session %d", c.sessionID)
		log.Error(err.Error())
		return err
	}
	dev, err := c.storage.GenerateClient(string(m.Content[0].Data), string(m.Content[2].Data))
	if err != nil {
		log.Error(err.Error())
		return fmt.Errorf("Client with name %s already exicst in session %d", string(m.Content[0].Data), c.sessionID)
	}
	c.device = *dev
	log.Tracef("Generat client with name %s and id %d", dev.Name, dev.ID)
	if err = c.storage.SaveClient(dev); err != nil {
		log.Error(err.Error())
		return fmt.Errorf("Can not save new client with name %s in session %d", string(m.Content[0].Data), c.sessionID)
	}
	thisID := strconv.FormatUint(dev.ID, 16)
	c.readChan <- dto.Message{
		Command: registerCOMMAND,
		Jmp:     m.Jmp,
		Proto:   m.Proto,
		Content: []dto.Content{
			dto.Content{Data: []byte("0")},
			m.Content[0],
			dto.Content{Data: []byte(thisID)},
		},
	}
	connection.AddClientToCache(dev.ID, c)
	return nil
}

func (c *C2cDevice) generateNewDevice(m *dto.Message) error {
	if c.device.ID != 0 {
		err := fmt.Errorf("Client already exist error in session %d", c.sessionID)
		log.Warning(err.Error())
		return err
	}
	if dev, err := c.storage.GenerateRandomClient(string(m.Content[2].Data)); err == nil {
		if err = c.storage.SaveClient(dev); err != nil {
			log.Error(err.Error())
			return fmt.Errorf("Can not save new client with name %s in session %d", string(m.Content[0].Data), c.sessionID)
		}
		c.device = *dev
		c.readChan <- dto.Message{
			Command: generateCOMMAND,
			Jmp:     m.Jmp,
			Proto:   m.Proto,
			Content: []dto.Content{
				dto.Content{Data: []byte("0")},
				dto.Content{Data: []byte(c.device.Name)},
			},
		}
		connection.AddClientToCache(c.device.ID, c)
		return nil
	}
	return fmt.Errorf("Can not generate new client in session %d", c.sessionID)
}

func (c *C2cDevice) findID(arg string) uint64 {
	var toID uint64
	var err error
	if toID, err = strconv.ParseUint(arg, 16, 64); err != nil {
		if v, err := c.storage.GetClientByName(arg); err == nil { //  Может "кому" у нас имя, а нам надо ID
			return v.ID
		}
		// Если "кому" - не ясно => вернем 0
		return 0
	}
	return toID
}

func (c *C2cDevice) sendNewMessage(msg *dto.Message) error {
	toID := c.findID(string(msg.Content[1].Data))
	c.listenerMtx.RLock()
	defer c.listenerMtx.RUnlock()
	if toID == 0 {
		for id, ch := range c.listenerList {
			if ch != nil {
				to := strconv.FormatUint(id, 16)
				msg.Content[1] = dto.Content{Data: []byte(to)}
				*ch <- *msg
			}
		}
	} else {
		if val, ok := c.listenerList[toID]; ok {
			if val != nil {
				*val <- *msg
			}
		} else {
			return fmt.Errorf("Client with id %d undefined in session %d", toID, c.sessionID)
		}
	}
	return nil
}

//TODO not tested yet
//Content[0] - from: local ID or Name, Content[1] - destroy connection from who if == '0' destroy connection from all
func (c *C2cDevice) destroyConnection(msg *dto.Message) error {
	// check if name or id from is equal to local name or id
	if !strings.EqualFold(string(msg.Content[0].Data), c.device.Name) {
		localID := strconv.FormatUint(c.device.ID, 16)
		if !strings.EqualFold(string(msg.Content[0].Data), localID) {
			err := fmt.Errorf("User name or id %s is not equal to local name %s or id %s in session %d", string(msg.Content[0].Data), c.device.Name, localID, c.sessionID)
			log.Warning(err.Error())
			return err
		}
	}
	toID := c.findID(string(msg.Content[1].Data))
	if toID == 0 { // disconnect from all connected devices
		log.Tracef("Close all connection for client %s: %d in session %d", c.device.Name, c.device.ID, c.sessionID)
		c.listenerMtx.Lock()
		for id, ch := range c.listenerList {
			if ch != nil {
				to := strconv.FormatUint(id, 16)
				msg.Content[1] = dto.Content{Data: []byte(to)}
				*ch <- *msg                // Передаем сообщение об отключении себя от них
				delete(c.listenerList, id) // Удаляем у себя подписанные устройства
			}
		}
		c.listenerMtx.Unlock()
		connection.DelClientFromCashe(c.device.ID)  // Удаляем в подписанных устройствах себя
		connection.AddClientToCache(c.device.ID, c) // Заново добавляем себя в кеш
		return nil
	}
	c.listenerMtx.Lock()
	if ch, ok := c.listenerList[toID]; ok {
		log.Tracef("Destroy connection with on client %d in session %d", toID, c.sessionID)
		*ch <- *msg                  // Send destroy connection message to the remote device
		delete(c.listenerList, toID) // Удаляем у себя подписанное устройство
	}
	c.listenerMtx.Unlock()
	connection.DisconnectClient(c.device.ID, toID) // Удялем в подписанных устройствах себя
	return nil
}

// setProperies - дополнительная команда, призванная передавать настроечные параметры (обговоренные участниками общения)
// Аппаратно специфичные штуки. Сделана как отдельная команда только для удобства.
// На самом деле для сервера все равно какие настройки (данные) будут в аргументах
// В качестве параметром "Кого" (msg.Content[0].Data) и "Кому" (msg.Content[1].Data) не может быть обобщенных данных. Все должно быть конкретно!!!
func (c *C2cDevice) setProperies(msg *dto.Message) error {
	toID := c.findID(string(msg.Content[1].Data))
	if toID == 0 {
		err := fmt.Errorf("To client must be specified")
		log.Warning(err.Error())
		return err
	}
	c.listenerMtx.RLock()
	if ch, ok := c.listenerList[toID]; ok {
		if ch != nil {
			*ch <- *msg
		}
	}
	c.listenerMtx.RUnlock()
	return nil
}

// errorHandler - пока затычка с логированием
//TODO Обработка ошибок от телеметрии
func (c *C2cDevice) errorHandler(msg *dto.Message) error {
	errMsg := "Error "
	for i, v := range msg.Content {
		if i == 0 {
			errMsg += "From "
		} else if i == 1 {
			errMsg += "To "
		}
		errMsg += string(v.Data) + " "
	}
	log.Error(errMsg)
	return nil
}
