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
			From:    "0",
			To:      m.From,
			Content: []byte(strings.ToUpper(currTimeStr)),
		}
		log.Tracef("Ping command from device %s, id %d", c.device.Name, c.device.ID)
		return nil
	}
	log.Errorf("PING error. Undefined client in session %d", c.sessionID)
	return NewC2cError(BadCommandError, "Initialize device at first")
}

func (c *C2cDevice) connectByID(m *dto.Message) error {
	if c.device.ID == 0 {
		return NewC2cError(BadCommandError, "Initialize device at first")
	}
	from, err := strconv.ParseUint(m.From, 16, 64)
	if err != nil {
		log.Warning(err.Error())
		return Errorf(InvalidCredentials, "\"%s\" must be a number", m.From)
	}
	if c.device.ID != uint64(from) {
		log.Warningf("session %d client ID in request command is incorrect originID %d != requestedID %d", c.sessionID, c.device.ID, from)
		return NewC2cError(InvalidCredentials, "Incorrect client id")
	}
	to, err := strconv.ParseUint(m.To, 16, 64)
	if err != nil {
		log.Warning(err.Error())
		return Errorf(InvalidCredentials, "\"%s\" must be a number", m.To)
	}
	if err = connection.ConnectClients(to, from); err != nil {
		log.Warning(err.Error())
		return Errorf(ClientNotFindError, "Can not create connection from %d whith abonnent %d", from, to)
	}
	c.readChan <- dto.Message{
		Command: connectByIDCOMMAND,
		Jmp:     m.Jmp,
		Proto:   m.Proto,
		From:    m.To,
		To:      m.From,
		Content: []byte(answerConnectByIDOk),
	}
	log.Infof("Connect by ID command from device %d to device %d finished fine", from, to)
	return nil
}

func (c *C2cDevice) connectByName(m *dto.Message) error {
	if c.device.ID == 0 {
		return NewC2cError(BadCommandError, "Initialize device at first")
	}
	if c.device.Name != m.From {
		log.Warningf("Incorrect device name %s != %s in session %d", c.device.Name, m.From, c.sessionID)
		return NewC2cError(InvalidCredentials, "Incorrect client name")
	}
	toClientID, err := c.storage.GetClientID(m.To)
	if err != nil {
		log.Warning(err.Error())
		return NewC2cError(ClientNotFindError, "Undefined target client")
	}
	if err := connection.ConnectClients(toClientID, c.device.ID); err != nil {
		log.Warning(err.Error())
		return Errorf(ClientNotFindError, "Can not create connection from %s with abonnent %s", m.From, m.To)
	}
	c.readChan <- dto.Message{
		Command: connectByNameCOMMAND,
		Jmp:     m.Jmp,
		Proto:   m.Proto,
		From:    m.To,
		To:      m.From,
		Content: []byte(answerConnectByNameOk),
	}
	log.Infof("Connect by name command from device %s to device %s finished fine", m.From, m.To)
	return nil
}

// For init by ID you need send ID (Content[0]), (salt ; signature)-(Content[2]) signature-base64(SHA256(ID + salt + base64(SHA256(name+password))))
func (c *C2cDevice) initByID(m *dto.Message) error {
	id, err := strconv.ParseUint(m.From, 16, 64)
	if err != nil {
		log.Warningf("Can not find corect ID in session %d %s", c.sessionID, err.Error())
		return NewC2cError(InvalidCredentials, "ID must be a number")
	}
	credentials := strings.Split(string(m.Content), ";") // Разделим соль от подписи
	if len(credentials) < 2 {
		err := Errorf(InvalidCredentials, "Client %d undefined signature for initialize in session %d", id, c.sessionID)
		log.Warning(err.Error())
		return err
	}
	if CheckSaltByID(id, credentials[0]) > saltUniqCount {
		err := Errorf(InvalidCredentials, "Client %d salt already been used %d times in session %d", id, saltUniqCount, c.sessionID)
		log.Warning(err.Error())
		return err
	}
	if c.device.ID == 0 {
		device, err := c.storage.GetClient(id)
		if err != nil {
			log.Warning(err.Error())
			return NewC2cError(ClientNotFindError, err.Error())
		}
		c.device = *device
	}
	if c.device.ID == id {
		temp := sha256.Sum256([]byte(string(m.From) + credentials[0] + c.device.SecretKey))
		origin := base64.StdEncoding.EncodeToString(temp[:])
		if origin != credentials[1] {
			log.Warningf("Incorrect signature %s != %s in session %d", origin, credentials[1], c.sessionID)
			c.device.ID = 0
			c.device.Name = ""
			return Errorf(InvalidCredentials, "Client %d initialize fail session %d", id, c.sessionID)
		}
		if er := connection.AddClientToCache(c.device.ID, c); er == nil {
			c.readChan <- dto.Message{
				Command: initByIDCOMMAND,
				Jmp:     m.Jmp,
				Proto:   m.Proto,
				From:    "0",
				To:      m.From,
				Content: []byte(answerInitByIDOk),
			}
			log.Infof("Client %d init by id ok", c.device.ID)
			return nil
		}
		log.Infof("Credentials is equals TODO destroy old session with client %s id: %d", c.device.Name, c.device.ID)
		er := fmt.Errorf("Client %d can not create in session %d", id, c.sessionID)
		log.Error(er.Error())
		return er
	}
	c.device.ID = 0
	c.device.Name = ""
	return Errorf(BadCommandError, "Incorrect ID in session %d", c.sessionID)
}

// For init by name you need send name (m.From), (salt ; signature)-(m.Content) signature - base64(SHA256(name + salt + base64(SHA256(name+password))))
func (c *C2cDevice) initByName(m *dto.Message) error {
	credentials := strings.Split(string(m.Content), ";") // Разделим соль от подписи
	if len(credentials) < 2 {
		err := Errorf(InvalidCredentials, "Client %s undefined signature for initialize in session %d", m.From, c.sessionID)
		log.Warning(err.Error())
		return err
	}
	if CheckSaltByUserName(m.From, credentials[0]) > saltUniqCount {
		err := Errorf(InvalidCredentials, "Client %s salt already been used %d times", m.From, saltUniqCount)
		log.Warning(err.Error())
		return err
	}
	if c.device.ID == 0 {
		id, err := c.storage.GetClientID(m.From)
		if err != nil {
			log.Warning(err.Error())
			return NewC2cError(ClientNotFindError, err.Error())
		}
		device, err := c.storage.GetClient(id)
		if err != nil {
			log.Warning(err.Error())
			return NewC2cError(ClientNotFindError, err.Error())
		}
		c.device = *device
	}
	if c.device.Name == m.From {
		t := m.From + credentials[0] + c.device.SecretKey
		temp := sha256.Sum256([]byte(t))
		origin := base64.StdEncoding.EncodeToString(temp[:])
		if origin != credentials[1] {
			log.Errorf("Origin credentials: %s", t)
			log.Errorf("SHA256: %x", temp)
			log.Errorf("Incorrect signature %s != %s in session %d", origin, credentials[1], c.sessionID)
			c.device.ID = 0
			c.device.Name = ""
			return Errorf(InvalidCredentials, "client %s finded and initialize fail in session %d", m.From, c.sessionID)
		}
		if er := connection.AddClientToCache(c.device.ID, c); er != nil {
			log.Warning(er.Error())
			log.Infof("Credentials is equals TODO destroy old session with client %s id: %d", c.device.Name, c.device.ID)
			er = fmt.Errorf("Can not create abonent in session %d", c.sessionID)
			log.Error(er.Error())
			c.device.ID = 0
			c.device.Name = ""
			return er
		}
		c.readChan <- dto.Message{
			Command: initByNameCOMMAND,
			Jmp:     m.Jmp,
			Proto:   m.Proto,
			From:    "0",
			To:      m.From,
			Content: []byte(answerInitByNameOk),
		}
		log.Infof("Client %s init by name ok", c.device.Name)
		return nil
	}
	c.device.ID = 0
	c.device.Name = ""
	err := Errorf(BadCommandError, "Incorrect name in session %d", c.sessionID)
	log.Warning(err.Error())
	return err
}

// For registration new device you need send an unique name, and base64(sha256(name+password))
func (c *C2cDevice) registerNewDevice(m *dto.Message) error {
	if c.device.ID != 0 {
		err := Errorf(BadCommandError, "Client %d already exist error in session %d", c.device.ID, c.sessionID)
		log.Warning(err.Error())
		return err
	}
	dev, err := c.storage.GenerateClient(c.clientType, m.From, string(m.Content))
	if err != nil {
		log.Warning(err.Error())
		return Errorf(BadCommandError, "Client with name %s already exicst in session %d", m.From, c.sessionID)
	}
	if err = c.storage.SaveClient(dev); err != nil {
		log.Warning(err.Error())
		return Errorf(InternalError, "Can not save new client with name %s in session %d", m.From, c.sessionID)
	}
	c.device = *dev
	thisID := strconv.FormatUint(dev.ID, 16)
	c.readChan <- dto.Message{
		Command: registerCOMMAND,
		Jmp:     m.Jmp,
		Proto:   m.Proto,
		From:    "0",
		To:      m.From,
		Content: []byte(thisID),
	}
	log.Infof("Registered new client %s with ID %d", c.device.Name, c.device.ID)
	return connection.AddClientToCache(dev.ID, c)
}

// generateNewDevice - генерирует имя и  идентификатор для указанного в m.Content пароля минимум три символа
func (c *C2cDevice) generateNewDevice(m *dto.Message) error {
	if c.device.ID != 0 {
		err := Errorf(BadCommandError, "Client %d already exist error in session %d", c.device.ID, c.sessionID)
		log.Warning(err.Error())
		return err
	}
	if dev, err := c.storage.GenerateRandomClient(c.clientType, string(m.Content)); err == nil {
		if err = c.storage.SaveClient(dev); err != nil {
			log.Warning(err.Error())
			return Errorf(InternalError, "Can not save new client with name %s in session %d", m.From, c.sessionID)
		}
		c.device = *dev
		c.readChan <- dto.Message{
			Command: generateCOMMAND,
			Jmp:     m.Jmp,
			Proto:   m.Proto,
			From:    "0",
			To:      c.device.Name,
		}
		log.Infof("Generate new client %s with ID %d", c.device.Name, c.device.ID)
		return connection.AddClientToCache(c.device.ID, c)
	}
	return Errorf(InternalError, "Can not generate new client in session %d", c.sessionID)
}

// findID - вернет идентификатор клиента. На вход подается либо идентификатор либо имя
func (c *C2cDevice) findID(arg string) uint64 {
	if toID, err := strconv.ParseUint(arg, 16, 64); err == nil {
		return toID
	}
	if toID, err := c.storage.GetClientID(arg); err == nil {
		return toID
	}
	return 0
}

func (c *C2cDevice) sendNewMessage(msg *dto.Message) error {
	toID := c.findID(msg.To)
	c.listenerMtx.RLock()
	defer c.listenerMtx.RUnlock()
	if toID == 0 {
		for id, ch := range c.listenerList {
			if ch != nil {
				msg.To = strconv.FormatUint(id, 16)
				*ch <- *msg
			}
		}
	} else {
		if val, ok := c.listenerList[toID]; ok {
			if val != nil {
				*val <- *msg
			}
		} else {
			return Errorf(ClientNotFindError, "Client with id %d undefined in session %d", toID, c.sessionID)
		}
	}
	return nil
}

//TODO not tested yet
//m.From - from: local ID or Name, m.To - destroy connection from who if == '0' destroy connection from all
func (c *C2cDevice) destroyConnection(msg *dto.Message) error {
	// check if name or id from is equal to local name or id
	if !strings.EqualFold(msg.From, c.device.Name) {
		localID := strconv.FormatUint(c.device.ID, 16)
		if !strings.EqualFold(msg.From, localID) {
			err := Errorf(BadCommandError, "User name or id %s is not equal to local name %s or id %s in session %d",
				msg.From, c.device.Name, localID, c.sessionID)
			log.Warning(err.Error())
			return err
		}
	}
	toID := c.findID(msg.To)
	if toID == 0 { // disconnect from all connected devices
		log.Infof("Close all connection for client %s: %d in session %d", c.device.Name, c.device.ID, c.sessionID)
		c.listenerMtx.Lock()
		for id, ch := range c.listenerList {
			if ch != nil {
				to := strconv.FormatUint(id, 16)
				msg.To = to
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
	ch, ok := c.listenerList[toID]
	if !ok {
		c.listenerMtx.Unlock()
		return Errorf(ClientNotFindError, "Undefined client %s when try destroy session whith %s", msg.To, c.device.Name)
	}
	log.Tracef("Destroy connection with on client %d in session %d", toID, c.sessionID)
	*ch <- *msg                  // Send destroy connection message to the remote device
	delete(c.listenerList, toID) // Удаляем у себя подписанное устройство
	c.listenerMtx.Unlock()
	connection.DisconnectClient(c.device.ID, toID) // Удялем в подписанных устройствах себя
	return nil
}

// setProperies - дополнительная команда, призванная передавать настроечные параметры (обговоренные участниками общения)
// Аппаратно специфичные штуки. Сделана как отдельная команда только для удобства.
// На самом деле для сервера все равно какие настройки (данные) будут в аргументах
// В качестве параметром "Кого" (msg.Content[0].Data) и "Кому" (msg.Content[1].Data) не может быть обобщенных данных. Все должно быть конкретно!!!
func (c *C2cDevice) setProperies(msg *dto.Message) error {
	toID := c.findID(msg.To)
	if toID == 0 {
		err := NewC2cError(BadCommandError, "To client must be specified")
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
	log.Error(fmt.Sprintf("Error from %s, to %s", msg.From, msg.To))
	return nil
}
