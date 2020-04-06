package trafficClient

import (
	"blabu/c2cService/client"
	"blabu/c2cService/data/c2cData"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"errors"
	"io"
	"time"
)

type traficCounterWrapper struct {
	storage        c2cData.DB
	client         client.ReadWriteCloser
	stat           dto.ClientLimits
	initialBalance float64
	validate       func(st dto.ClientLimits) (dto.ClientLimits, error)
}

//GetNewTraficCounterWrapper - вернет обертку, которая реализует подсчет трафика принятых и отправленных байт
func GetNewTraficCounterWrapper(storage c2cData.DB, cl client.ReadWriteCloser) client.ReadWriteCloser {
	return &traficCounterWrapper{
		storage:  storage,
		client:   cl,
		stat:     dto.ClientLimits{},
		validate: updateLimits,
	}
}

func (c *traficCounterWrapper) Write(msg *dto.Message) error {
	var er error
	if c.stat.ID == 0 {
		log.Tracef("Try init clinet %s stat in write method", msg.From)
		rc := c.stat.ReceiveBytes
		tr := c.stat.TransmiteBytes
		c.stat, _ = initStat(msg.From, c.storage)
		c.initialBalance = c.stat.Balance
		c.stat.TransmiteBytes += tr
		c.stat.ReceiveBytes += rc
	} else if c.stat, er = c.validate(c.stat); er != nil {
		log.Error(er.Error())
		return er
	}
	c.stat.TransmiteBytes += uint64(len(msg.Content))
	return c.client.Write(msg)
}

func (c *traficCounterWrapper) Read(dt time.Duration, handler func(msg dto.Message, err error)) {
	var er error
	c.client.Read(dt, func(msg dto.Message, err error) {
		if c.stat.ID == 0 {
			log.Tracef("Try init clinet %s stat in read method", msg.From)
			rc := c.stat.ReceiveBytes
			tr := c.stat.TransmiteBytes
			c.stat, _ = initStat(msg.From, c.storage)
			c.initialBalance = c.stat.Balance
			c.stat.TransmiteBytes += tr
			c.stat.ReceiveBytes += rc
		} else if err == nil {
			if c.stat, er = c.validate(c.stat); er != nil {
				log.Error(er.Error())
				handler(dto.Message{}, io.EOF)
				return
			}
		}
		handler(msg, err)
		c.stat.ReceiveBytes += uint64(len(msg.Content))
	})
}

func (c *traficCounterWrapper) Close() error {
	if err := c.storage.UpdateIfNotModified(&c.stat); err != nil {
		log.Error(err.Error())
		client, err := c.storage.GetStat(c.stat.ID)
		if err != nil {
			log.Error(err.Error())
			return errors.New("Can not save client")
		}
		c.stat.Balance = client.Balance - (c.initialBalance - c.stat.Balance)
		c.stat.MaxReceivedBytes = client.MaxReceivedBytes
		c.stat.MaxTransmittedBytes = client.MaxTransmittedBytes
		c.stat.Rate = client.Rate
		c.stat.TimePeriod = client.TimePeriod
		c.storage.UpdateStat(&c.stat)
	}
	return c.client.Close()
}
