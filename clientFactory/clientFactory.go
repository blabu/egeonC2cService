package clientFactory

import (
	"blabu/c2cService/client"
	"blabu/c2cService/client/c2cService"
	"blabu/c2cService/client/savemsgservice"
	"blabu/c2cService/client/trafficclient"
	conf "blabu/c2cService/configuration"
	c2cData "blabu/c2cService/data/c2cdata"
	"blabu/c2cService/parser"
	"strconv"
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
		// peerClient := s2sservice.NewDecorator(p, db, uint32(m), client)
		msgClient := savemsgservice.NewDecorator(db, client)
		return trafficclient.GetNewTraficCounterWrapper(db, msgClient)
	default:
		return nil
	}
}
