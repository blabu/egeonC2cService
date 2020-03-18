package clientFactory

import (
	"blabu/c2cService/client"
	"blabu/c2cService/client/s2sService"
	"blabu/c2cService/client/trafficClient"
	conf "blabu/c2cService/configuration"
	"blabu/c2cService/data/c2cData"
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
		peerClient := s2sService.NewDecorator(p, db, sessionID, uint32(m))
		return trafficClient.GetNewTraficCounterWrapper(db, peerClient)
		//return c2cService.NewC2cDevice(c2cData.GetBoltDbInstance(), sessionID, uint32(m))
	default:
		return nil
	}
}
