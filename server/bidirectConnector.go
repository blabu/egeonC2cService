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

//readHandler - Поток дял чтения данных из интернета (всегда ждем данных), проверяем полное ли сообщение, если полное, отправляем дальше в канал readData
func (c *BidirectConnection) readHandler(Connect *net.Conn, kill <-chan bool, finishRead chan<- bool) {
	defer func() {
		close(finishRead)
		log.Trace("Finish readHandler")
	}()
	res, _ := strconv.ParseUint(conf.GetConfigValueOrDefault("ReceiveBufferSize", "512"), 10, 32)
	request := make([]byte, 0, res)
	resp := make([]byte, res/4)
	bufferdReader := bufio.NewReader(*Connect)
	var p parser.Parser
	for {
		select {
		case <-kill:
			log.Info("Close read handler in BidirectConnection")
			return
		default:
			(*Connect).SetReadDeadline(time.Now().Add(c.Duration))
			n, err := bufferdReader.Read(resp) // Читаем!!!
			if err != nil {
				log.Infof("Error when try read from conection: %v\n", err)
				return
			}
			request = append(request, resp[:n]...) // Добавляем к сообщению прочтенное
			if p != nil {
				ok, err := p.IsFullReceiveMsg(request)
				if err != nil {
					log.Error(err.Error())
					return // Пришедший пакет не соответсвует протоколу
				}
				if !ok {
					log.Debug("Message not full")
					continue
				}
				c.updateWatchDogTimer()
				if len(request) < 127 {
					log.Debug("Receive Full message ", string(request))
				} else {
					log.Debug("Receive Full message is more than 127 bytes")
				}
				if er := c.logic.Get().Write(request); er != nil {
					log.Error(er.Error())
					return // TODO Выполнять обработку ошибок
				}
				request = request[:0] // Читстим буфер, данные отправлены
			} else { // Если парсер еще не инициализирован передаем принятые данные на инициализацию парсера
				log.Debug("Parser still not initialize")
				if p, err = parser.InitParser(request); err != nil {
					return
				}
				var messageIsFull bool
				var e error
				if messageIsFull, e = p.IsFullReceiveMsg(request); e != nil {
					log.Error("Message invalid, for this parser type ", e.Error())
					return
				}
				c.logic.Set(CreateReadWriteMainLogic(p, time.Second))
				defer c.logic.Get().Close()
				go c.logic.Get().Read(func(data []byte, err error) {
					if err == io.EOF {
						log.Debug(err.Error())
						(*Connect).Close()
						return
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
				if !messageIsFull {
					log.Trace("Message not full")
					continue
				}
				if err := c.logic.Get().Write(request); err != nil {
					log.Error(err.Error())
					return // TODO Выполнять обработку ошибок
				}
				request = request[:0] // Чистим старые данные. Если с первого сообщения проинициализировать парсер не удалось то выбрасываем это сообщение
			}
		}
	}
}

// ManageSession - is a function that manage new connection.
// Инициализирует парсер по первому сообщению.
// Инициализирует и запускает клиентскую логику.
// Контролирует с помощью парсера полноту сообщения и передает это сообщение клиентской логики
// If connection is finished or some error net.Connection
func (c *BidirectConnection) ManageSession(Connect net.Conn, st *stat.Statistics) {
	st.NewConnection() // Регистрируем новое соединение

	kill := make(chan bool)
	defer func(start time.Time) { // Если с из вне произошло отключение
		st.CloseConnection()
		st.SetConnectionTime(time.Since(start))
		close(kill)
		Connect.Close() // Соединение необходимо разрушить в случае конца сессии
		log.Info("Finish connector")
	}(time.Now()) // Фиксируем конец сесии во времени

	readNetworkStoped := make(chan bool) // Канал для остановки логики работ с соединением
	go c.readHandler(&Connect, kill, readNetworkStoped)
	select {
	case <-c.Tm.C:
		log.Info("Timeout close connector")
		return
	case <-readNetworkStoped:
		log.Info("Close connector finish network read operation")
		return
	}
}
