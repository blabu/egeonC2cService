/*
Package client - Содержит всю бизнес логику системы
Имеет послойную архитектуру. Каждый ВЕРХНИЙ слой зависит ИСКЛЮЧИТЕЛЬНО от следующего за ним нижестоящего.
Содержит такие слои:
	ClientLogic и ClientInterface - логика работы с клиентом. Здесь создается девайс
	Пакет deviceLogic - предоставляет доступ к данным девайса.*/
package client

import (
	"blabu/c2cService/dto"
	"io"
	"time"
)

//ListenerInterface - интерфейс, который позволяет реализовать систему подписки
// на рассылку от устройства устройству
type ListenerInterface interface {
	AddListener(from uint64, ch *chan dto.Message)
	DelListener(from uint64)
	GetListenerChan() *chan dto.Message
}

//ReadWriteCloser - создает интерфейс работы с клиентом
type ReadWriteCloser interface {
	// GetID - идентификатор клиента
	GetID() uint64

	// Write - Передаем данные полученные из сети бизнес логике
	Write(msg *dto.Message) error

	//Read - читаем ответ бизнес логики return io.EOF if client never answer
	Read(dt time.Duration, handler func(msg dto.Message, err error))

	// Close - информирует бизнес логику про разрыв соединения
	io.Closer
}

// CachedClientInterface - агрегация клиентского интерфейса
type CachedClientInterface interface {
	ListenerInterface
	ReadWriteCloser
}
