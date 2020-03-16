package dto

import (
	"time"
)

// Content - Содержание сообщения
// type Content struct {
// 	Data []byte
// }

// Message - это данные от устройства прошедшие валидацию и разделенные на содержимое и команду
type Message struct {
	messageID uint32
	Proto     uint16
	Jmp       uint16
	Command   uint16
	From      string
	To        string
	Content   []byte
}

// ClientDescriptor - base entity for client to client messanger
type ClientDescriptor struct {
	ID           uint64    `json:"ID"`
	Name         string    `json:"Name"` /*Начинается ОБЯЗАТЕЛЬНО с буквы латинского алфавита*/
	SecretKey    string    `json:"Key"`
	RegisterDate time.Time `json:"Registered"`
	AllowedUsers []string  `json:"Alloved"` /*Список разрешений для пользователя*/
}

//ClientStat - client base statistics
type ClientStat struct {
	ID                  uint64        `json:"ID"`
	Name                string        `json:"Name"`
	LastActivity        time.Time     `json:"LastActivity"`
	TransmiteBytes      uint64        `json:"Transmit"`
	ReceiveBytes        uint64        `json:"Receive"`
	MaxReceivedBytes    uint64        `json:"MaxRx"`
	MaxTransmittedBytes uint64        `json:"MaxTx"`
	LimitExpiration     time.Time     `json:"LimitExpiration"`
	TimePeriod          time.Duration `json:"Period"`
}
