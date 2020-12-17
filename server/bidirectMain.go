package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	log "github.com/blabu/egeonC2cService/logWrapper"

	"github.com/blabu/egeonC2cService/client"
	"github.com/blabu/egeonC2cService/clientFactory"
	"github.com/blabu/egeonC2cService/dto"
	"github.com/blabu/egeonC2cService/parser"
)

// bidirectMain - двунаправленная реализация MainLogicIO для независимого чтения и записи информации
// Реализовано:
// 1. чтение с клиента и запись в сеть метод Read()
// 2. запись в клиента метод Write()
type bidirectMain struct {
	sessionID uint32
	p         parser.Parser
	c         client.ReadWriteCloser
}

//CreateReadWriteMainLogic - Создаем новый интерфейс для MainLogicIO (логики взаимодействия сервера и клиентской логики)
//!!!НИКОГДА НЕ ВОЗРАЩАЕТ NIL!!!
func CreateReadWriteMainLogic(p parser.Parser, readTimeout time.Duration) MainLogicIO {
	sesID := atomic.AddUint32(&lastSessionID, 1)
	return &bidirectMain{
		sessionID: sesID,
		p:         p,
		c:         clientFactory.CreateClientLogic(p, sesID),
	}
}

// Write - синхронный вызов парсит сообщение полученное с сети и пишет данные в клиентскую логику
func (s *bidirectMain) Write(data []byte) (int, error) {
	if s.c == nil {
		return 0, errors.New("Nil error")
	}
	m, err := s.p.ParseMessage(data)
	if err != nil {
		log.Warningf("Can not parse message in session %d. Error %s", s.sessionID, err.Error())
		return 0, err
	}
	return len(data), s.c.Write(&m)
}

//Read - читает из системы и передает данные обработчику handler
func (s *bidirectMain) Read(ctx context.Context, handler dto.ServerReadHandler) {
	if s.c == nil {
		handler(nil, errors.New("Parser or client is nil"))
		return
	}
	s.c.Read(ctx, func(msg dto.Message, systemError error) error {
		if systemError != nil {
			if systemError == io.EOF { // Читать больше нечего
				handler(nil, io.EOF) // Закрываем соединение
			}
			log.Info(systemError.Error())
			return systemError
		}
		log.Trace("Received data from client logic fine")
		return handler(s.p.FormMessage(msg)) //Передаем данные для отправки в интернет
	})
}

// Close - закрываем соединения с клиентской логикой
func (s *bidirectMain) Close() error {
	log.Infof("Close bidirectMain and client logic in session %d", s.sessionID)
	if s.c != nil {
		s.c.Close()
		return nil
	}
	return fmt.Errorf("Client logic is nil")
}
