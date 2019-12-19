package c2cService

import (
	"blabu/c2cService/client"
	"blabu/c2cService/data/c2cData"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"fmt"
	"io"
	"sync"
	"time"
)

var connection client.ConnectionCache

func init() {
	connection = client.NewConnectionCache()
}

// C2cDevice - Сущность реализующая интерфейс клиента для двустороннего обмена сообщениями
// и интерфейс ClientListenerInterface для добавления его в кеш
type C2cDevice struct {
	sessionID    uint32
	storage      c2cData.C2cDB
	device       dto.ClientDescriptor // Номер устройства
	readChan     chan dto.Message
	listenerList map[uint64]*chan dto.Message // Список каналов устройств слушающих отправляемые сообщения этого клиента
	listenerMtx  sync.RWMutex                 // Для защиты списка каналов устройств слушающих сообщения этого клиента
}

func (c *C2cDevice) AddListener(from uint64, ch *chan dto.Message) {
	if ch != nil {
		c.listenerMtx.Lock()
		c.listenerList[from] = ch
		c.listenerMtx.Unlock()
		log.Tracef("Add channel from client %d to %s", from, c.device.Name)
	}
}

func (c *C2cDevice) DelListener(from uint64) {
	c.listenerMtx.Lock()
	delete(c.listenerList, from)
	c.listenerMtx.Unlock()
	log.Tracef("Delete channel from client %d for %s", from, c.device.Name)
}

func (c *C2cDevice) GetListenerChan() *chan dto.Message {
	return &c.readChan
}

// NewC2cDevice - Конструктор нового клеинта
func NewC2cDevice(s c2cData.C2cDB, sessionID uint32, maxCONNECTION uint32) client.ClientInterface {
	var c = new(C2cDevice)
	c.sessionID = sessionID
	c.storage = s
	c.readChan = make(chan dto.Message, maxCONNECTION) // Делаем его буферизированным, чтобы много узлов смогли отпраить ему сообщение
	c.listenerList = make(map[uint64]*chan dto.Message)
	return c
}

// Write - если сессия открыта выполняет передачу данных
// если сессия закрыта ищет в сообщении возможные функций выполненяет их и формирует ответ
func (c *C2cDevice) Write(msg *dto.Message) error {
	if msg == nil {
		return fmt.Errorf("Message is nil in session %d", c.sessionID)
	}
	if len(msg.Content) < 4 { // От кого, кому, данные (может быть пустым), и счетчик прыжков (счетчик прыжков нужен для перезапроса на этом уровне не используется)
		return fmt.Errorf("Not enough arguments in message in session %d", c.sessionID)
	}
	switch msg.Command {
	case errorCOMMAND:
		return c.errorHandler(msg)
	case pingCOMMAND:
		return c.ping(msg.Content)
	case connectByIDCOMMAND: // Content[0] - from ID, Content[1] - to ID
		return c.connectByID(msg.Content)
	case connectByNameCOMMAND: // Content[0] - from name, Content[1] - to name
		return c.connectByName(msg.Content)
	case initByIDCOMMAND: // Content[0] - from ID, Content[1] - to (server always "0")
		return c.initByID(msg.Content)
	case initByNameCOMMAND: // Content[0] - from name, Content[1] - to (server always "0")
		return c.initByName(msg.Content)
	case registerCOMMAND:
		return c.registerNewDevice(msg.Content) // Content[0] - from name, Content[1] - to (server always "0") , Content[2] - BASE64(SHA256(name+password))
	case generateCOMMAND:
		return c.generateNewDevice(msg.Content) // Content[0] - is empty, Content[1] - to (server always "0"), Content[2] - BASE64 string password hash
	case dataCOMMAND:
		return c.sendNewMessage(msg)
	case destroyConCOMMAND: // Разорвать соединения без отключения от сервера
		return c.destroyConnection(msg) //Content[0] - from: local ID or Name, Content[1] - destroy connection from who.
	case propertiesCOMMAND:
		return c.setProperies(msg) //Content[0] - from: local ID or Name, Content[1] - to
	default:
		return fmt.Errorf("Unsupported command %d in session %d", msg.Command, c.sessionID)
	}
}

//Read - читаем ответ от бизнес логики или стороннего клиента
//Ждущая функция, вернет управления если:
// 1. Приготовлен ответ
// 2. Истекло время ожидания ответа
// 3. Произшла ошибка чтения
func (c *C2cDevice) Read(dt time.Duration, handler func(msg dto.Message, err error)) {
	t := time.NewTimer(dt)
	for {
		select {
		case m, ok := <-c.readChan:
			if !ok {
				log.Tracef("Read channel is closed for device %d %s for session %d", c.device.ID, c.device.Name, c.sessionID)
				handler(dto.Message{}, io.EOF)
				return
			}
			handler(m, nil)
		case <-t.C:
			err := fmt.Errorf("Read timeout in session %d", c.sessionID)
			handler(dto.Message{}, err)
		}
	}
}

// Close - информирует бизнес логику про разрыв соединения
func (c *C2cDevice) Close() {
	connection.DelClientFromCashe(c.device.ID)
	close(c.readChan)
	log.Infof("Close client with id %d in session %d", c.device.ID, c.sessionID)
}
