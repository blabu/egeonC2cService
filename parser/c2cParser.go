package parser

import (
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

/*
Протокол.
$Vx..x;yy...yy;zz...zz;tt...tt;jj...jj;ss...ss###................
Все сообщение разделено на заголовок (выполняется ИСКЛЮЧИТЕЛЬНО в текстовом виде латиница в кодировке ASCI)
И сам данные (произволный формат)
Заголовок состоит из следующих полей разделенных символом ';':
$V - "магическая последовательность" начало посылки
x..x - версия протокола (число в шестнадцатиричном представлении)
yy...yy - от кого посылка (строка ASCII)
zz...zz - кому посылка (строка ASCII)
tt...tt - тип сообщения (число в шестнадцатиричном представлении)
jj...jj - кол-во прыжков (после которого вернем ошибку)
ss...ss - размер передаваемых данных (число в шестнадцатиричном представлении)
### - Конец заголовка

Пример: $V1;от кого;кому;тип сообщения(команда);размер сообщения###\0САМО СООБЩЕНИЕ
*/
// maxPackageSzie - максимальный размер одного пакета в байтах
const maxPackageSzie = 8 * 1048576 /*(8 * 1Mb)*/
const beginHeader string = "$V"
const endHeader string = "###"
const delimStr string = ";"
const minHeader string = beginHeader + delimStr + delimStr + delimStr + delimStr + delimStr + endHeader
const headerParamSize = 6

type header struct {
	headerSize  int
	from        string
	to          string
	mType       uint64
	jumpCnt     uint64
	contentSize int // Размер данных
}

// C2cParser - Парсер разбирает сообщения по протоколу
// 1 - клиент-клиент
type C2cParser struct {
	protocolVer uint64
}

//FormMessage - from - Content[0], to - Content[1], data - Content[2]
func (c2c *C2cParser) FormMessage(msg dto.Message) ([]byte, error) {
	if len(msg.Content) < 4 {
		return nil, fmt.Errorf("Error. Incorrect input message")
	}
	res := make([]byte, 0, 128+len(msg.Content[0].Data))
	res = append(res, []byte(beginHeader)...)
	res = append(res, []byte(strconv.FormatUint(c2c.protocolVer, 16))...)
	res = append(res, ';')
	res = append(res, msg.Content[0].Data...)
	res = append(res, ';')
	res = append(res, msg.Content[1].Data...)
	res = append(res, ';')
	res = append(res, []byte(strconv.FormatUint(uint64(msg.Command), 16))...)
	res = append(res, ';')
	res = append(res, []byte(strconv.FormatUint(binary.LittleEndian.Uint64(msg.Content[3].Data), 16))...)
	res = append(res, ';')
	res = append(res, []byte(strconv.FormatUint(uint64(len(msg.Content[2].Data)), 16))...)
	res = append(res, []byte(endHeader)...)
	res = append(res, msg.Content[2].Data...)
	return res, nil
}

// return size of header and error if not find header or parsing error
func (c2c *C2cParser) parseHeader(data []byte) (int, header, error) {
	var resHeader header
	if data == nil || len(data) < len(minHeader) {
		return 0, resHeader, fmt.Errorf("Input is empty, nothing to be parsed")
	}
	index := bytes.IndexByte(data, '$')
	if index < 0 {
		return 0, resHeader, fmt.Errorf("Undefined start symb of package")
	}
	start := string(data[index : index+2])
	if !strings.EqualFold(start, beginHeader) {
		return index, resHeader, fmt.Errorf("Package must be started from %s but %s", beginHeader, start)
	}
	resHeader.headerSize = bytes.Index(data, []byte(endHeader)) // Поиск конца заголовка
	if resHeader.headerSize < index || resHeader.headerSize >= len(data) {
		return index, resHeader, fmt.Errorf("Undefined end header %s", endHeader)
	}
	parsed := strings.Split(string(data[index+2:resHeader.headerSize]), delimStr)
	if len(parsed) < headerParamSize {
		return index, resHeader, fmt.Errorf("Incorrect header")
	}
	var err error
	if c2c.protocolVer, err = strconv.ParseUint(parsed[0], 16, 64); err != nil { //Версия протокола
		log.Tracef("Can not parse number in %s error %s", parsed[0], err.Error())
		return index, resHeader, fmt.Errorf("Icorrect protocol version, it must be a number")
	}
	switch c2c.protocolVer {
	case 0: // Для сервер-сервер соединения
		fallthrough
	case 1: // Для клиент-сервер соединения
		resHeader.from = parsed[1]                                                   // от кого
		resHeader.to = parsed[2]                                                     //кому
		if resHeader.mType, err = strconv.ParseUint(parsed[3], 16, 64); err != nil { //тип сообщения (команда)
			return index, resHeader, fmt.Errorf("Icorrect message type, it must be a number")
		}
		if resHeader.jumpCnt, err = strconv.ParseUint(parsed[4], 16, 64); err != nil {
			return index, resHeader, fmt.Errorf("Incorrect message jump type")
		}
		if resHeader.jumpCnt == 0 {
			return index, resHeader, fmt.Errorf("Jump count is zero")
		}
		var s uint64
		if s, err = strconv.ParseUint(parsed[5], 16, 64); err != nil { //размер сообщения
			return index, resHeader, fmt.Errorf("Icorrect message size, it must be a number")
		}
		if s > maxPackageSzie {
			err := fmt.Errorf("Income package is too big. Overflow internal buffer")
			log.Error(err.Error())
			return index, resHeader, err
		}
		resHeader.contentSize = int(s)
		resHeader.headerSize += len(endHeader) // Add endHeader
		return index, resHeader, nil
		// TODO implement another version of protocol
	default:
		return index, resHeader, fmt.Errorf("Error usuported porotocol")
	}
}

//ParseMessage - from - Content[0], to - Content[1], data - Content[2]
func (c2c *C2cParser) ParseMessage(data []byte) (dto.Message, error) {
	var err error
	var head header
	var i int
	if i, head, err = c2c.parseHeader(data); err != nil {
		log.Trace(err.Error())
		return dto.Message{}, err
	}
	log.Tracef("Message from %s to %s type %d jump %d and content size %d", head.from, head.to, head.mType, head.jumpCnt, head.contentSize)
	head.jumpCnt--
	jmp := make([]byte, 8)
	binary.LittleEndian.PutUint64(jmp, head.jumpCnt)
	return dto.Message{
		Command: uint16(head.mType),
		Content: []dto.Content{
			dto.Content{
				Data: []byte(head.from),
			},
			dto.Content{
				Data: []byte(head.to),
			},
			dto.Content{
				Data: data[i+head.headerSize : i+head.headerSize+head.contentSize],
			},
			dto.Content{
				Data: jmp,
			},
		},
	}, nil
}

// IsFullReceiveMsg - Проверка пришел полный пакет или нет
// TODO каждый раз парсить заголовок не эффективно надо будет переписать
func (c2c *C2cParser) IsFullReceiveMsg(data []byte) (bool, error) {
	var err error
	var head header
	var i int
	if i, head, err = c2c.parseHeader(data); err != nil {
		log.Trace(err.Error())
		return false, err
	}
	if len(data) >= i+head.contentSize+head.headerSize {
		return true, nil
	}
	return false, nil
}

func (c2c *C2cParser) GetParserType() uint64 {
	return C2cParserType
}
