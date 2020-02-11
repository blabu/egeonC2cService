package server

import (
	conf "blabu/c2cService/configuration"
	log "blabu/c2cService/logWrapper"
	"blabu/c2cService/parser"
	"blabu/c2cService/stat"
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
type BidirectConnection struct {
	Tm       *time.Timer
	Duration time.Duration
	logic    atomicMainLog
}

func NewBidirectConnector(dT time.Duration) Connector {
	timer := time.NewTimer(dT)
	return &BidirectConnection{
		Tm:       timer,
		Duration: dT,
	}
}

func (c *BidirectConnection) updateWatchDogTimer() {
	if c.Duration != 0 {
		c.Tm.Reset(c.Duration)
	}
}

func (c *BidirectConnection) initParser(r io.Reader, resp *[]byte) (parser.Parser, error) {
	if conn, ok := r.(net.Conn); ok {
		log.Trace("Set timeout read operation")
		conn.SetReadDeadline(time.Now().Add(c.Duration))
	}
	n, e := r.Read(*resp)
	if e != nil {
		return nil, e
	}
	*resp = (*resp)[:n]
	return parser.InitParser(*resp)
}

//readHandler - Поток для чтения данных из интернета (всегда ждем данных), проверяем полное ли сообщение, если полное, отправляем дальше в канал readData
func (c *BidirectConnection) readHandler(Connect *net.Conn, stopConnectionFromNet <-chan bool, stopConnectionFromClient chan<- bool) {
	defer func() {
		close(stopConnectionFromClient)
		log.Trace("Finish readHandler")
	}()
	maxPacketSize, _ := strconv.ParseUint(conf.GetConfigValueOrDefault("MaxPacketSize", "512"), 10, 32)
	maxPacketSize *= 1024
	resp := make([]byte, 128)
	bufferdReader := bufio.NewReader(*Connect)
	p, err := c.initParser(bufferdReader, &resp)
	if err != nil {
		log.Error(err.Error())
		return
	}
	c.logic.Set(CreateReadWriteMainLogic(p, time.Second))
	defer c.logic.Get().Close()
	go c.logic.Get().Read(func(data []byte, err error) {
		if err == io.EOF {
			log.Debug(err.Error())
			(*Connect).Close()
			return
		}
		if err != nil {
			log.Info(err.Error())
		} else if err == nil && data != nil {
			(*Connect).SetWriteDeadline(time.Now().Add(time.Duration(len(data)) * time.Millisecond)) // timeout for write data 1 millisecond for every bytes
			if _, err := (*Connect).Write(data); err != nil {                                        // Отправляем в сеть
				log.Debug("Write ok")
			}
			return
		}
		log.Info("Data to transmit is nil")
		return
	})
	for {
		select {
		case <-stopConnectionFromNet:
			log.Info("Close read handler in BidirectConnection from network")
			return
		default:
			c.updateWatchDogTimer()
			n, err := p.IsFullReceiveMsg(resp)
			if err != nil {
				log.Error(err.Error())
				return
			}
			if uint64(n+len(resp)) > maxPacketSize {
				log.Errorf("Message %s is to big %d and max %d", string(resp), n+len(resp), maxPacketSize)
				return
			}
			if n == 0 {
				if err := c.logic.Get().Write(resp); err != nil {
					log.Error(err.Error())
					return // TODO Выполнять обработку ошибок
				}
				resp = resp[:128]
				(*Connect).SetReadDeadline(time.Now().Add(c.Duration))
				n, err = bufferdReader.Read(resp) // Читаем!!!
				if err != nil {
					log.Infof("Error when try read from conection: %v\n", err)
					return
				}
				resp = resp[:n]
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
			resp = append(resp, temp[:n]...) // Добавляем к сообщению прочтенное
		}
	}
}

// SessionHandler - is a function that manage new connection.
// Инициализирует парсер по первому сообщению.
// Инициализирует и запускает клиентскую логику.
// Контролирует с помощью парсера полноту сообщения и передает это сообщение клиентской логики
// If connection is finished or some error net.Connection
func (c *BidirectConnection) SessionHandler(Connect net.Conn, st *stat.Statistics) {
	st.NewConnection() // Регистрируем новое соединение

	stopConnectionFromNet := make(chan bool)
	defer func(start time.Time) { // Если с из вне произошло отключение
		st.CloseConnection()
		st.SetConnectionTime(time.Since(start))
		close(stopConnectionFromNet)
		Connect.Close() // Соединение необходимо разрушить в случае конца сессии
		log.Info("Finish connector")
	}(time.Now()) // Фиксируем конец сесии во времени

	stopConnectionFromLogic := make(chan bool) // Канал для остановки логики работ с соединением
	go c.readHandler(&Connect, stopConnectionFromNet, stopConnectionFromLogic)
	select {
	case <-c.Tm.C:
		log.Info("Timeout close connector")
		return
	case <-stopConnectionFromLogic:
		log.Info("Close connector finish network read operation")
		return
	}
}
