package clientFactory

import (
	"strconv"

	"github.com/blabu/egeonC2cService/client"
	"github.com/blabu/egeonC2cService/client/c2cService"
	conf "github.com/blabu/egeonC2cService/configuration"
	c2cData "github.com/blabu/egeonC2cService/data/c2cdata"
	"github.com/blabu/egeonC2cService/parser"
	"github.com/blabu/egeonC2cService/savemsgservice"
)

//CreateClientLogic - create client for c2c or s2s communication
func CreateClientLogic(p parser.Parser, sessionID uint32) client.ReadWriteCloser {
	m, e := strconv.ParseUint(conf.GetConfigValueOrDefault("MaxQueuePacketSize", "64"), 10, 32)
	if e != nil {
		m = 64
	}
	switch p.GetParserType() {
	case parser.C2cParserType:
		db := c2cData.GetBoltDbInstance()
		client := c2cService.NewC2cDevice(db, sessionID, uint32(m))
		return savemsgservice.NewDecorator(db, client)
	default:
		return nil
	}
}
