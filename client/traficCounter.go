package client

import (
	"blabu/c2cService/dto"
	"errors"
	"time"
)

type traficCounterWrapper struct {
	receivedBytes       uint64
	maxReceivedBytes    uint64
	transmittedBytes    uint64
	maxTransmittedBytes uint64
	client              CachedClientInterface
}

func (c *traficCounterWrapper) Write(msg *dto.Message) error {
	c.receivedBytes += uint64(len(msg.Content))
	if c.maxReceivedBytes != 0 && c.receivedBytes > c.maxReceivedBytes {
		return errors.New("Overflow bytes limit")
	}
	return c.client.Write(msg)
}

func (c *traficCounterWrapper) Read(dt time.Duration, handler func(msg dto.Message, err error)) {
	c.client.Read(dt, func(msg dto.Message, err error) {
		if err == nil {
			c.transmittedBytes += uint64(len(msg.Content))
		}
		handler(msg, err)
	})
}

func (c *traficCounterWrapper) Close() error {
	return c.client.Close()
}
