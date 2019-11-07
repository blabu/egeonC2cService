package s2sService

import (
	"blabu/c2cService/client"
	"blabu/c2cService/client/c2cService"
	"blabu/c2cService/configuration"
	"blabu/c2cService/data/c2cData"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"blabu/c2cService/parser"
	"errors"
	"io"
	"net"
	"strings"
	"time"
)

/*
Пакет реализует ClientInterface
для межсерверного и клиент-серверного взаимодействия в p2p сети
Оборачивая в себе пакет клиент-клиент взаимодействия.
Выполняя обработку ошибок
*/

// NewDecorator - создает новый клиент обертку для поиска клиентов по сети из серверов
func NewDecorator(p parser.Parser, s c2cData.C2cDB, sessionID uint32, maxCONNECTION uint32) client.ClientInterface {
	client := c2cService.NewC2cDevice(s, sessionID, maxCONNECTION)
	srvListString := configuration.GetConfigValueOrDefault("PeerSrv", "")
	srvList := strings.Split(srvListString, ";")
	service := C2cDecorate{
		p:              p,
		serverLists:    srvList,
		client:         client,
		serverReadChan: make(chan dto.Message, maxCONNECTION),
		//kill:           make(chan bool, 1),
		conn: nil,
	}
	return &service
}

// C2cDecorate - Декоратор с2с соединения реализующий обмен между двумя клиентами в p2p сети
type C2cDecorate struct {
	p              parser.Parser
	serverLists    []string
	client         client.ClientInterface
	serverReadChan chan dto.Message
	conn           *net.Conn
}

func readFromConnection(p parser.Parser, conn net.Conn, handler func(dto.Message, error)) {
	readBuffer := make([]byte, 0, 2048)
	for { // Пытаемся прочитать полный ответ разпарсить его и подготовить ответ
		tempRead := make([]byte, 256)
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, er := conn.Read(tempRead)
		if er != nil { // Удаленный сервер разорвал соединение
			log.Trace(er.Error())
			handler(dto.Message{}, er)
			return
		}
		tempRead = tempRead[:n]
		readBuffer = append(readBuffer, tempRead...)
		isFull, err := p.IsFullReceiveMsg(readBuffer)
		if err != nil { // Сообщение не корректное
			log.Trace(er.Error())
			handler(dto.Message{}, err)
			continue
		}
		if isFull {
			m, er := p.ParseMessage(readBuffer)
			if er != nil {
				log.Trace(er.Error())
				handler(dto.Message{}, er)
				continue
			}
			handler(m, nil)
			readBuffer = readBuffer[:0]
		}
	}
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
	defer func() { log.Trace("Client delegate write finish") }()
	if s.conn != nil {
		if er := s.writeToRemoteServerHandler(msg, *s.conn); er != nil {
			(*s.conn).Close()
			s.conn = nil
		} else {
			return nil
		}
	}
	err := s.client.Write(msg)
	if err == nil {
		return nil
	}
	log.Trace("Try find new connection")
	if s.conn == nil {
		for _, addr := range s.serverLists {
			log.Trace("Try connect to ", addr)
			conn, e := net.Dial("tcp", addr)
			if e != nil {
				log.Trace("Connection fail")
				continue
			}
			if e := s.writeToRemoteServerHandler(msg, conn); e == nil {
				go readFromConnection(s.p, conn, func(m dto.Message, err error) {
					if err == nil {
						s.conn = &conn
						s.serverReadChan <- m // Отправляем ответ
					} else {
						if s.conn != nil {
							close(s.serverReadChan)
						}
						conn.Close()
						log.Error(err.Error())
					}
				})
				return nil
			}
			conn.Close()
		}
		return errors.New("Not find server")
	}
	return err
}

func (s *C2cDecorate) clientRead(clientReadChan chan<- dto.Message, kill <-chan bool, dt time.Duration) {
	delegateFinish := make(chan bool, 1)
	defer func() {
		log.Trace("Client deleget readhandler finish")
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
		}
	}
}

// Close - информирует бизнес логику про разрыв соединения
func (s *C2cDecorate) Close() {
	if s.conn != nil {
		(*s.conn).Close()
	}
	log.Trace("Close client decorator")
	s.client.Close()
}
