package server

import (
	"blabu/c2cService/client"
	"blabu/c2cService/clientFactory"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"blabu/c2cService/parser"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// bidirectMainLogic - двунаправленная реализация MainLogicIO для независимого чтения и записи информации
type bidirectMainLogic struct {
	sessionID uint32
	dt        time.Duration
	p         parser.Parser
	c         client.ClientInterface
}

//CreateReadWriteMainLogic - Создаем новый интерфейс для MainLogicIO (логики взаимодействия сервера и клиентской логики)
//!!!НИКОГДА НЕ ВОЗРАЩАЕТ NIL!!!
func CreateReadWriteMainLogic(p parser.Parser, readTimeout time.Duration) MainLogicIO {
	if p == nil {
		return new(bidirectMainLogic)
	}
	sesID := atomic.AddUint32(&lastSessionID, 1)
	return &bidirectMainLogic{
		sessionID: sesID,
		dt:        readTimeout,
		p:         p,
		c:         clientFactory.CreateClientLogic(p, sesID),
	}
}

// Write - синхронный вызов парсит согласно парсеру и пишет данные внутрь сервера (в клиентскую логику)
func (s *bidirectMainLogic) Write(data []byte) error {
	if s.p == nil {
		return fmt.Errorf("Error parser is nil in session %d", s.sessionID)
	}
	if s.c == nil {
		return fmt.Errorf("Error client logic is nil in session %d", s.sessionID)
	}
	log.Tracef("Session %d Try write to client logic", s.sessionID)
	m, err := s.p.ParseMessage(data)
	if err != nil {
		log.Warningf("Can not parse message in session %d. Error %s", s.sessionID, err.Error())
		return err
	}
	m.SessionID = s.sessionID
	return s.c.Write(&m)
}

func (s *bidirectMainLogic) Read(handler func([]byte, error)) {
	if s.p == nil {
		handler(nil, errors.New("Parser is nil"))
		return
	}
	if s.c == nil {
		handler(nil, errors.New("Client is nil"))
		return
	}
	s.c.Read(s.dt, func(msg dto.Message, err error) {
		if err != nil {
			if err == io.EOF { // Читать больше нечего
				log.Debug(err.Error())
				handler(nil, io.EOF)
			}
			return
		}
		log.Trace("Received data from client logic fine")
		handler(s.p.FormMessage(msg))
	})
}

// Close - закрываем соединения с клиентской логикой
func (s *bidirectMainLogic) Close() {
	log.Infof("Close bidirectMainLogic and client logic in session %d", s.sessionID)
	if s.c != nil {
		s.c.Close()
	}
}
