package server

import (
	conf "blabu/c2cService/configuration"
	log "blabu/c2cService/logWrapper"
	"blabu/c2cService/parser"
	"io"

	"bufio"
	"net"
	"strconv"
	"sync"
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

//BidirectConnection - структура, которая управляет соединением реализует интерфейс Connector
//У сервера два независимых процесса чтения и записи могут происходить одновременно
type BidirectSession struct {
	Tm       *time.Timer
	Duration time.Duration
	netReq   []byte
	logic    atomicMainLog
}

func (c *BidirectSession) updateWatchDogTimer() {
	if c.Duration != 0 {
		c.Tm.Reset(c.Duration)
	}
}

//readHandler - Поток для чтения данных из интернета (всегда ждем данных),
// проверяем полное ли сообщение, если полное, отправляем дальше
func (c *BidirectSession) readHandler(Connect *net.Conn, kill <-chan bool, finishRead chan<- bool, p parser.Parser) {
	defer func() {
		close(finishRead)
		log.Trace("Finish readHandler")
	}()
	maxPacketSize, _ := strconv.ParseUint(conf.GetConfigValueOrDefault("MaxPacketSize", "512"), 10, 32)
	maxPacketSize *= 1024
	bufferdReader := bufio.NewReader(*Connect)

	for {
		select {
		case <-kill:
			log.Info("Close read handler in BidirectConnection")
			return
		default:
			c.updateWatchDogTimer()
			n, err := p.IsFullReceiveMsg(c.netReq)
			if err != nil {
				log.Error(err.Error())
				return
			}
			if uint64(n+len(c.netReq)) > maxPacketSize {
				log.Errorf("Message %s is to big %d and max %d", string(c.netReq), n+len(c.netReq), maxPacketSize)
				return
			}
			if n == 0 {
				log.Debug(string(c.netReq))
				if err := c.logic.Get().Write(c.netReq); err != nil {
					log.Error(err.Error())
					return // TODO Выполнять обработку ошибок
				}
				c.netReq = c.netReq[:128]
				(*Connect).SetReadDeadline(time.Now().Add(c.Duration))
				n, err = bufferdReader.Read(c.netReq) // Читаем!!!
				if err != nil {
					log.Infof("Error when try read from conection: %v\n", err)
					return
				}
				c.netReq = c.netReq[:n]
				continue
			}
			log.Tracef("Try read last %d bytes", n)
			temp := make([]byte, n)
			(*Connect).SetReadDeadline(time.Now().Add(c.Duration))
			n, err = bufferdReader.Read(temp) // Читаем!!!
			if err != nil {
				log.Infof("Error when try read from conection: %v\n", err)
				return
			}
			c.netReq = append(c.netReq, temp[:n]...) // Добавляем к сообщению прочтенное
		}
	}
}

// Run - is a function that manage new connection.
// Инициализирует парсер по первому сообщению.
// Инициализирует и запускает клиентскую логику.
// Контролирует с помощью парсера полноту сообщения и передает это сообщение клиентской логики
// If connection is finished or some error net.Connection
func (c *BidirectSession) Run(Connect net.Conn, p parser.Parser) {
	kill := make(chan bool)
	defer close(kill)

	go c.logic.Get().Read(func(data []byte, err error) {
		if err == io.EOF {
			log.Debug(err.Error())
			Connect.Close()
			return
		} else if err == nil && data != nil {
			Connect.SetWriteDeadline(time.Now().Add(time.Duration(len(data)) * time.Millisecond)) // timeout for write data 1 millisecond for every bytes
			if _, err := Connect.Write(data); err != nil {                                        // Отправляем в сеть
				log.Debug("Write ok")
			}
			return
		}
		log.Info("Data to transmit is nil")
		return
	})

	readNetworkStoped := make(chan bool) // Канал для остановки логики работ с соединением
	go c.readHandler(&Connect, kill, readNetworkStoped, p)
	select {
	case <-c.Tm.C:
		log.Info("Timeout close connector")
		return
	case <-readNetworkStoped:
		log.Info("Close connector finish network read operation")
		return
	}
}
