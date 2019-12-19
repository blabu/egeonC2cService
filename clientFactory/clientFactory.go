package clientFactory

import (
	"blabu/c2cService/client"
	"blabu/c2cService/client/s2sService"
	"blabu/c2cService/data/c2cData"
	"blabu/c2cService/parser"
)

func CreateClientLogic(p parser.Parser, sessionID uint32) client.ClientInterface {
	switch p.GetParserType() {
	case parser.C2cParserType:
		return s2sService.NewDecorator(p, c2cData.GetBoltDbInstance(), sessionID, 16)
		// return c2cService.NewC2cDevice(c2cData.GetBoltDbInstance(), sessionID, 16)
	default:
		return nil
	}
}
