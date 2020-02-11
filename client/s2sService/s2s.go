package s2sService

import (
	"blabu/c2cService/client"
	"blabu/c2cService/client/c2cService"
	"blabu/c2cService/configuration"
	"blabu/c2cService/data/c2cData"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"blabu/c2cService/parser"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

/*
Пакет реализует ClientInterface
для межсерверного и клиент-серверного взаимодействия в p2p сети
Оборачивая в себе пакет клиент-клиент взаимодействия.
Выполняя обработку ошибок.
Выполняет поиск клиента в других серверах в случае если локальный сервер этого клиента не нашел
*/

// NewDecorator - создает новый клиент обертку для поиска клиентов по сети из серверов
func NewDecorator(p parser.Parser, s c2cData.C2cDB, sessionID uint32, maxCONNECTION uint32) client.ClientInterface {
	client := c2cService.NewC2cDevice(s, sessionID, maxCONNECTION)
	srvListString := configuration.GetConfigValueOrDefault("PeerList", "")
	srvList := strings.Split(srvListString, ",")
	service := C2cDecorate{
		p:              p,
		serverLists:    srvList,
		client:         client,
		serverReadChan: make(chan dto.Message, maxCONNECTION),
		conn:           nil,
		timeout:        10 * time.Second,
	}
	return &service
}

// C2cDecorate - Декоратор с2с соединения реализующий обмен между двумя клиентами в p2p сети
type C2cDecorate struct {
	p              parser.Parser
	serverLists    []string
	client         client.ClientInterface
	serverReadChan chan dto.Message
	conMtx         sync.Mutex
	conn           net.Conn
	timeout        time.Duration
}

func (s *C2cDecorate) readFromConnection(reader io.Reader, handler func(dto.Message, error)) error {
	readBuffer := make([]byte, 1, 2048) // Размер буфера выставляем равным 1 байту для попытки успешного чтения хотябы одного байта
	if c, ok := reader.(net.Conn); ok {
		c.SetReadDeadline(time.Now().Add(10 * time.Second))
	}
	if _, er := reader.Read(readBuffer); er != nil { // Пытаемся прочитать хотябы один байт
		return er
	}
	go func(readBuffer []byte) { // Если успешно прочли хотябы один байт, читаем остальное
		tempRead := make([]byte, 256)
		for { // Пытаемся прочитать полный ответ разпарсить его и подготовить ответ
			if c, ok := reader.(net.Conn); ok {
				c.SetReadDeadline(time.Now().Add(s.timeout))
			}
			n, er := reader.Read(tempRead)
			if er != nil { // Удаленный сервер разорвал соединение
				log.Trace(er.Error())
				handler(dto.Message{}, er)
				return
			}
			readBuffer = append(readBuffer, tempRead[:n]...)
			sz, err := s.p.IsFullReceiveMsg(readBuffer)
			if err != nil { // Сообщение не корректное
				log.Trace(er.Error())
				handler(dto.Message{}, err)
				continue
			}
			if sz == 0 {
				m, er := s.p.ParseMessage(readBuffer)
				if er != nil {
					log.Trace(er.Error())
					handler(dto.Message{}, er)
					continue
				}
				handler(m, nil)
				readBuffer = readBuffer[:0]
			}
		}
	}(readBuffer)
	return nil
}

func (s *C2cDecorate) writeToRemoteServerHandler(msg *dto.Message, conn net.Conn) error {
	buf, err := s.p.FormMessage(*msg)
	if err != nil {
		return err
	}
	if _, err := conn.Write(buf); err != nil {
		return err
	}
	return nil
}

// Write - Передаем данные полученные из сети бизнес логике
func (s *C2cDecorate) Write(msg *dto.Message) error {
	s.conMtx.Lock()
	if s.conn != nil {
		if er := s.writeToRemoteServerHandler(msg, s.conn); er != nil {
			s.conn.Close()
			s.conn = nil
		} else {
			s.conMtx.Unlock()
			return nil
		}
	}
	s.conMtx.Unlock()
	err := s.client.Write(msg)
	if err == nil {
		return nil
	}
	if er, ok := err.(c2cService.C2cError); ok {
		if er.ErrType > c2cService.DisableConnectionErrorLimit {
			return er
		}
	} else {
		return err
	}
	log.Trace("Try find new connection")
	if s.conn == nil {
		for _, addr := range s.serverLists {
			log.Trace("Try connect to ", addr)
			conf := tls.Config{
				InsecureSkipVerify: true,
			}
			conn, e := tls.Dial("tcp", addr, &conf)
			if e != nil {
				log.Trace("Connecction fail")
				continue
			}
			if e := s.writeToRemoteServerHandler(msg, conn); e != nil {
				conn.Close()
				continue
			}
			s.conMtx.Lock()
			s.conn = conn
			s.conMtx.Unlock()
			if er := s.readFromConnection(conn, func(m dto.Message, err error) {
				if err != nil {
					s.conMtx.Lock()
					s.conn.Close()
					s.conn = nil
					s.conMtx.Unlock()
					close(s.serverReadChan)
					return
				}
				s.serverReadChan <- m
			}); er != nil {
				s.conMtx.Lock()
				s.conn.Close()
				s.conn = nil
				s.conMtx.Unlock()
				continue
			}
			return nil
		}
		return errors.New("Not find server")
	}
	return err
}

func (s *C2cDecorate) clientRead(clientReadChan chan<- dto.Message, kill <-chan bool, dt time.Duration) {
	delegateFinish := make(chan bool, 1)
	defer func() {
		log.Trace("Client delegat readhandler finish")
		close(clientReadChan)
	}()
	for {
		select {
		case <-kill:
			return
		case <-delegateFinish:
			return
		default:
			s.client.Read(dt, func(msg dto.Message, err error) {
				if err == io.EOF {
					log.Trace(err.Error())
					delegateFinish <- true
					return
				}
				if err == nil {
					clientReadChan <- msg
				}
			})
		}
	}
}

//Read - читаем ответ бизнес логики return io.EOF if client never answer
func (s *C2cDecorate) Read(dt time.Duration, handler func(msg dto.Message, err error)) {
	timer := time.NewTimer(dt)
	kill := make(chan bool, 1)
	clientReadChan := make(chan dto.Message)
	defer func() {
		kill <- true
	}()
	go s.clientRead(clientReadChan, kill, dt)
	for {
		select {
		case m, ok := <-s.serverReadChan:
			if !ok {
				log.Trace("Server read error")
				handler(dto.Message{}, io.EOF)
				return
			}
			handler(m, nil)
		case m, ok := <-clientReadChan:
			if !ok {
				log.Trace("Client read error")
				handler(dto.Message{}, io.EOF)
				return
			}
			handler(m, nil)
		case <-timer.C:
			handler(dto.Message{}, errors.New("Timeout"))
			timer.Reset(dt)
		}
	}
}

// Close - информирует бизнес логику про разрыв соединения
func (s *C2cDecorate) Close() {
	s.conMtx.Lock()
	defer s.conMtx.Unlock()
	if s.conn != nil {
		s.conn.Close()
	}
	log.Trace("Close client decorator")
	s.client.Close()
}
