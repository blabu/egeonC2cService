package parser

import (
	"bytes"
	"fmt"
	"strconv"

	conf "blabu/c2cService/configuration"
	log "blabu/c2cService/logWrapper"
)

const (
	startSymb        byte = '$'
	versionAttribute byte = 'V'
)

const (
	C2cParserType = iota + 1
)

// InitParser - инициалиизирует парсер исходя из заголовка сообщения по версии программы
func InitParser(receivedData []byte) (Parser, error) {
	if receivedData == nil {
		return nil, fmt.Errorf("nil received data")
	}
	poz := bytes.IndexByte(receivedData, startSymb)
	if poz < 0 {
		log.Warning("Undefined start symbol in ", string(receivedData))
		return nil, fmt.Errorf("Undefined start symbol in message BYTE_ARRAY: %v", receivedData)
	}
	receivedData = receivedData[poz:]
	if len(receivedData) < 3 {
		return nil, fmt.Errorf("Data is too short")
	}
	log.Trace("Create c2c parser")
	p := new(C2cParser)
	maxPackageSize, _ := strconv.ParseUint(conf.GetConfigValueOrDefault("MaxPacketSize", "512"), 10, 32)
	p.maxPackageSize = maxPackageSize * 1024
	return p, nil
}
