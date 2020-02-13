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
	"sync"
	"sync/atomic"
	"time"
)

type atomicMainLog struct {
	mtx  sync.RWMutex
	main MainLogicIO
}

func (a *atomicMainLog) Get() MainLogicIO {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.main
}

func (a *atomicMainLog) Set(m MainLogicIO) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.main = m
}

// bidirectMainLogic - двунаправленная реализация MainLogicIO для независимого чтения и записи информации
// Реализовано:
// 1. чтение с клиента и запись в сеть метод Read()
// 2. запись в клиента метод Write()
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

// Write - синхронный вызов парсит сообщение полученное с сети и пишет данные в клиентскую логику
func (s *bidirectMainLogic) Write(data []byte) (int, error) {
	if s.p == nil {
		return 0, fmt.Errorf("Error parser is nil in session %d", s.sessionID)
	}
	if s.c == nil {
		return 0, fmt.Errorf("Error client logic is nil in session %d", s.sessionID)
	}
	log.Tracef("Session %d Try write to client logic", s.sessionID)
	m, err := s.p.ParseMessage(data)
	if err != nil {
		log.Warningf("Can not parse message in session %d. Error %s", s.sessionID, err.Error())
		return 0, err
	}
	return len(data), s.c.Write(&m)
}

//Read - читает из бизнес логики и передает данные обработчику handler
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
func (s *bidirectMainLogic) Close() error {
	log.Infof("Close bidirectMainLogic and client logic in session %d", s.sessionID)
	if s.c != nil {
		s.c.Close()
		return nil
	}
	return fmt.Errorf("Client logic is nil")
}
