package data

import "github.com/blabu/c2cLib/dto"

//IClient - БАЗОВЫЙ интерфейс для клиент-клиент взаимодействия (Сделан для тестов)
type IClient interface {
	GetClient(ID uint64) (*dto.ClientDescriptor, error)
	DelClient(ID uint64) error
	GetClientID(name string) (uint64, error)
	SaveClient(cl *dto.ClientDescriptor) error
}

//ClientType - первые байты в идентиифкаторе клиента
type ClientType uint16

//IClientGenerator - Функции генерации нового клиента
type IClientGenerator interface {
	// GenerateRandomClient - Генерируем нового клиента, имя которого будет совпадать с его идентификационным номером
	GenerateRandomClient(T ClientType, hash string) (*dto.ClientDescriptor, error)
	// GenerateClient - Генерируем нового клиента по его имени и паролю
	GenerateClient(T ClientType, name, hash string) (*dto.ClientDescriptor, error)
}

//IMessage - интерфейс для сохранения сообщений
type IMessage interface {
	IsSended(userID uint64, messageID uint64)
	Add(userID uint64, msg dto.UnSendedMsg) (uint64, error)
	GetNext(userID uint64) (dto.UnSendedMsg, error)
}

//DB - интерфейс базы данных работы платформы сообщений
type DB interface {
	IClientGenerator
	IClient
	IMessage
	ForEach(tableName string, callBack func(key []byte, value []byte) error)
}
