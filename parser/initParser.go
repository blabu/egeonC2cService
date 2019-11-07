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
// Если после стартового байта (обязательно идет первым в посылке) следует байт указания на версию протокола,
// то будет инициалиизирован парсер реализующий данный протокол, или выдана ошибка если такого нет.
// Возможные протоколы
// 1. $00 - будет иниициализирован парсер ParseV0
// 2. $qqID=x или $qqTY=x, где qq - любое шестнадцатиричное число x = 1,2,3,4,6,7 - инициализирует прокси парсер
// 3. $qqID=x или $qqTY=x, где qq - любое шестнадцатиричное число x = 5 - ГРП парсер
// 4. $V1 - Инициализирует нешифрованное текстовое соединение клиент - клиент
// 5. $V2 - Инициализирует нешифрованное бинарное соединение клиент - клиент с кодировкой base64
func InitParser(receivedData []byte) (Parser, error) {
	if receivedData == nil {
		return nil, fmt.Errorf("nil received data")
	}
	poz := bytes.IndexByte(receivedData, startSymb)
	if poz < 0 {
		log.Warning("Undefined start symbol")
		return nil, fmt.Errorf("Undefined start symbol in message BYTE_ARRAY: %v", receivedData) // Бинарний протокол (для опросчика) не реализован
	}
	receivedData = receivedData[poz:]
	if len(receivedData) < 3 {
		return nil, fmt.Errorf("Data is too short")
	}
	log.Trace("Create c2c parser")
	return new(C2cParser), nil
}
