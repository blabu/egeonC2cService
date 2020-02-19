package client

import (
	cf "blabu/c2cService/configuration"
	"blabu/c2cService/dto"
	"errors"
	"io"
	"strconv"
	"time"
)

type traficCounterWrapper struct {
	receivedBytes       uint64
	maxReceivedBytes    uint64
	transmittedBytes    uint64
	maxTransmittedBytes uint64
	client              ReadWriteCloser
}

//GetNewTraficCounterWrapper - вернет обертку, которая реализует подсчет трафика принятых и отправленных байт
func GetNewTraficCounterWrapper(cl ReadWriteCloser) ReadWriteCloser {
	maxTransmittedBytes := cf.GetConfigValueOrDefault("MaxSessionTransmit", "0") // Не ограничено
	maxReceivedBytes := cf.GetConfigValueOrDefault("MaxSessionReceive", "0")     // не ограничено
	tr, _ := strconv.ParseUint(maxTransmittedBytes, 10, 64)
	rc, _ := strconv.ParseUint(maxReceivedBytes, 10, 64)
	return &traficCounterWrapper{
		receivedBytes:       0,
		transmittedBytes:    0,
		maxTransmittedBytes: tr,
		maxReceivedBytes:    rc,
		client:              cl,
	}
}

func (c *traficCounterWrapper) Write(msg *dto.Message) error {
	c.receivedBytes += uint64(len(msg.Content))
	if c.maxReceivedBytes != 0 && c.receivedBytes > c.maxReceivedBytes {
		return errors.New("Overflow receive bytes limit")
	}
	return c.client.Write(msg)
}

func (c *traficCounterWrapper) Read(dt time.Duration, handler func(msg dto.Message, err error)) {
	c.client.Read(dt, func(msg dto.Message, err error) {
		if err == nil {
			c.transmittedBytes += uint64(len(msg.Content))
			if c.maxTransmittedBytes != 0 && c.transmittedBytes > c.maxTransmittedBytes {
				handler(dto.Message{}, io.EOF)
				return
			}
		}
		handler(msg, err)
	})
}

func (c *traficCounterWrapper) Close() error {
	return c.client.Close()
}
