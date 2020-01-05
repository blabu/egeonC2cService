package parser

import (
	"bytes"
	"fmt"

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
		log.Warning("Undefined start symbol")
		return nil, fmt.Errorf("Undefined start symbol in message BYTE_ARRAY: %v", receivedData)
	}
	receivedData = receivedData[poz:]
	if len(receivedData) < 3 {
		return nil, fmt.Errorf("Data is too short")
	}
	log.Trace("Create c2c parser")
	return new(C2cParser), nil
}
