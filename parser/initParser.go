package parser

const (
	startSymb        byte = '$'
	versionAttribute byte = 'V'
)

//CreateEmptyParser - создает интерфейс парсера с ограничением максимального размера сообщения maxSize
// Кусок принятого сообщения нужен для создания других видов парсера в будущем
func CreateEmptyParser(receivedChank []byte, maxSize uint64) (Parser, error) {
	c2c := new(C2cParser)
	c2c.maxPackageSize = maxSize
	return c2c, nil
}
