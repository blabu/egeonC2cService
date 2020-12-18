package clientFactory

import (
	"github.com/blabu/egeonC2cService/client"
	"github.com/blabu/egeonC2cService/client/c2cService"
	"github.com/blabu/egeonC2cService/client/savemsgservice"
	cf "github.com/blabu/egeonC2cService/configuration"
	c2cData "github.com/blabu/egeonC2cService/data/c2cdata"
)

//CreateClientLogic - create client for c2c or s2s communication
func CreateClientLogic(sessionID uint32) client.ReadWriteCloser {
	m := cf.Config.MaxQueuePacketSize
	db := c2cData.GetBoltDbInstance()
	client := c2cService.NewC2cDevice(db, sessionID, m)
	return savemsgservice.NewDecorator(db, client)
}
