package clientFactory

import (
	"github.com/blabu/egeonC2cService/client"
	"github.com/blabu/egeonC2cService/client/c2cService"
	"github.com/blabu/egeonC2cService/client/savemsgservice"
	cf "github.com/blabu/egeonC2cService/configuration"
	c2cData "github.com/blabu/egeonC2cService/data/c2cdata"
	"github.com/blabu/egeonC2cService/parser"
)

//CreateClientLogic - create client for c2c or s2s communication
func CreateClientLogic(p parser.Parser, sessionID uint32) client.ReadWriteCloser {
	m := cf.Config.MaxQueuePacketSize
	switch p.GetParserType() {
	case parser.C2cParserType:
		db := c2cData.GetBoltDbInstance()
		client := c2cService.NewC2cDevice(db, sessionID, m)
		return savemsgservice.NewDecorator(db, client)
	default:
		return nil
	}
}
