package client

import "blabu/c2cService/dto"

//ClientListenerInterface - интерфейс, который позволяет реализовать систему подписки
// на рассылку от устройства устройству
type ClientListenerInterface interface {
	AddListener(from uint64, ch *chan dto.Message)
	DelListener(from uint64)
	GetListenerChan() *chan dto.Message
}
