package server

import (
	"context"
	"io"

	"github.com/blabu/egeonC2cService/parser"

	conf "github.com/blabu/egeonC2cService/configuration"
	log "github.com/blabu/egeonC2cService/logWrapper"

	"bufio"
	"net"
	"time"
)

//BidirectConnection - структура, которая управляет соединением реализует интерфейс Connector
//У сервера два независимых процесса чтения и записи могут происходить одновременно
type BidirectSession struct {
	Tm       *time.Timer
	Duration time.Duration
	netReq   []byte
	logic    MainLogicIO
}

func (c *BidirectSession) updateWatchDogTimer() {
	if c.Duration != 0 {
		c.Tm.Reset(c.Duration)
	}
}

//readHandler - Поток для чтения данных из интернета (всегда ждем данных),
// проверяем полное ли сообщение, если полное, отправляем дальше
func (c *BidirectSession) readHandler(
	Connect *net.Conn,
	stopConnectionFromNet <-chan bool,
	stopConnectionFromClient chan<- bool,
	p parser.Parser) {

	defer close(stopConnectionFromClient)
	maxPacketSize := uint64(conf.Config.MaxPacketSize) * 1024
	bufferdReader := bufio.NewReader(*Connect)
	for {
		select {
		case <-stopConnectionFromNet:
			log.Info("Close read handler in BidirectConnection from network")
			return
		default:
			c.updateWatchDogTimer()
			leftBytes, err := p.IsFullReceiveMsg(c.netReq)
			if err != nil {
				log.Warning(err.Error())
				return
			}
			if uint64(leftBytes+len(c.netReq)) > maxPacketSize {
				log.Errorf("Message %s is to big %d and max %d", string(c.netReq), leftBytes+len(c.netReq), maxPacketSize)
				return
			}
			if leftBytes == 0 { // Если все байты получены
				if _, err := c.logic.Write(c.netReq); err != nil {
					log.Warning(err.Error())
					return // TODO Выполнять обработку ошибок
				}
				c.netReq = c.netReq[:minHeaderSize]
				(*Connect).SetReadDeadline(time.Now().Add(c.Duration))
				numb, err := bufferdReader.Read(c.netReq) // Читаем!!!
				if err != nil {
					log.Infof("Error when try read from conection: %v", err)
					return
				}
				c.netReq = c.netReq[:numb]
				continue
			}
			log.Tracef("Try read last %d bytes", leftBytes)
			temp := make([]byte, leftBytes)
			(*Connect).SetReadDeadline(time.Now().Add(c.Duration))
			leftBytes, err = io.ReadFull(bufferdReader, temp) // Читаем остальное!!!
			if err != nil {
				log.Infof("Error when try read from conection: %v", err)
				return
			}
			c.netReq = append(c.netReq, temp[:leftBytes]...) // Добавляем к сообщению прочтенное
		}
	}
}

// Run - is a function that manage new connection.
// Инициализирует и запускает клиентскую логику.
// Контролирует с помощью парсера полноту сообщения и передает это сообщение клиентской логики
// If connection is finished or some error net.Connection
func (c *BidirectSession) Run(Connect net.Conn, p parser.Parser) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.logic.Read(ctx, func(data []byte, systemError error) error { //Читаем из системы пишем в интернет
		if data != nil && systemError == nil {
			c.updateWatchDogTimer()
			Connect.SetWriteDeadline(time.Now().Add(time.Duration(len(data)) * 10 * time.Millisecond))
			_, err := Connect.Write(data)
			return err
		} else if systemError == io.EOF { //Если ошибка из системы это конец потока. Закрываем соединение
			log.Info("Close connection by read operation")
			return Connect.Close()
		}
		return nil
	})

	stopConnectionFromNet := make(chan bool)
	defer close(stopConnectionFromNet)
	stopConnectionFromClient := make(chan bool) // Канал для остановки логики работ с соединением

	go c.readHandler(&Connect, stopConnectionFromNet, stopConnectionFromClient, p) //Читаем данные из интернета, отправляем в систему
	select {
	case <-c.Tm.C:
		log.Info("Timeout close connector")
		return
	case <-stopConnectionFromClient:
		log.Info("Close connector finish network read operation")
		return
	}
}
