package clientFactory

import (
	"blabu/c2cService/client"
	"blabu/c2cService/client/s2sService"
	conf "blabu/c2cService/configuration"
	"blabu/c2cService/data/c2cData"
	"blabu/c2cService/parser"
	"strconv"
)

//CreateClientLogic - create client for c2c or s2s communication
func CreateClientLogic(p parser.Parser, sessionID uint32) client.ClientInterface {
	m, e := strconv.ParseUint(conf.GetConfigValueOrDefault("maxConnectionBuffer", "64"), 10, 32)
	if e != nil {
		m = 64
	}
	switch p.GetParserType() {
	case parser.C2cParserType:
		return s2sService.NewDecorator(p, c2cData.GetBoltDbInstance(), sessionID, uint32(m))
		//return c2cService.NewC2cDevice(c2cData.GetBoltDbInstance(), sessionID, 16)
	default:
		return nil
	}
}
