package client

import (
	"blabu/c2cService/data/c2cData"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"errors"
	"io"
	"strconv"
	"time"
)

type traficCounterWrapper struct {
	storage c2cData.DB
	client  ReadWriteCloser
	stat    dto.ClientStat
}

//GetNewTraficCounterWrapper - вернет обертку, которая реализует подсчет трафика принятых и отправленных байт
func GetNewTraficCounterWrapper(storage c2cData.DB, cl ReadWriteCloser) ReadWriteCloser {
	return &traficCounterWrapper{
		storage: storage,
		client:  cl,
		stat:    dto.ClientStat{},
	}
}

func checkLimits(stat *dto.ClientStat) bool {
	if stat.ID != 0 && stat.LimitExpiration.Before(time.Now()) {
		stat.LimitExpiration = time.Now().Add(stat.TimePeriod)
		stat.ReceiveBytes = 0
		stat.TransmiteBytes = 0
		return true
	}
	if stat.MaxReceivedBytes != 0 && stat.MaxTransmittedBytes != 0 {
		return stat.ReceiveBytes < stat.MaxReceivedBytes && stat.TransmiteBytes < stat.MaxTransmittedBytes
	}
	return true
}

func initStat(from string, storage c2cData.DB) dto.ClientStat {
	var e error
	var stat dto.ClientStat
	if stat.ID, e = strconv.ParseUint(from, 16, 64); e != nil {
		if stat.ID, e = storage.GetClientID(from); e != nil {
			stat.ID = 0
			return stat
		}
	}
	if stat, e = storage.GetStat(stat.ID); e != nil {
		stat.ID = 0
	}
	return stat
}

func (c *traficCounterWrapper) Write(msg *dto.Message) error {
	if c.stat.ID == 0 {
		rc := c.stat.ReceiveBytes
		tr := c.stat.TransmiteBytes
		c.stat = initStat(msg.From, c.storage)
		c.stat.TransmiteBytes += tr
		c.stat.ReceiveBytes += rc
	}
	c.stat.ReceiveBytes += uint64(len(msg.Content))
	if !checkLimits(&c.stat) {
		return errors.New("Overflow receive bytes limit")
	}
	return c.client.Write(msg)
}

func (c *traficCounterWrapper) Read(dt time.Duration, handler func(msg dto.Message, err error)) {
	c.client.Read(dt, func(msg dto.Message, err error) {
		if c.stat.ID == 0 {
			rc := c.stat.ReceiveBytes
			tr := c.stat.TransmiteBytes
			c.stat = initStat(msg.From, c.storage)
			c.stat.TransmiteBytes += tr
			c.stat.ReceiveBytes += rc
		}
		if err == nil {
			c.stat.TransmiteBytes += uint64(len(msg.Content))
			if !checkLimits(&c.stat) {
				log.Error("Overflow transmite limit")
				handler(dto.Message{}, io.EOF)
				return
			}
		}
		handler(msg, err)
	})
}

func (c *traficCounterWrapper) Close() error {
	c.storage.UpdateStat(&c.stat)
	return c.client.Close()
}
