package dto

import (
	"time"
)

// Content - Содержание сообщения
type Content struct {
	Data []byte
}

// Message - это данные от устройства прошедшие валидацию и разделенные на содержимое и команду
type Message struct {
	SessionID uint32
	Command   uint16
	Content   []Content
}

// // FormMessageAnswer - Формирует сообщение из входных аргументов
// func FormMessageAnswer(command uint16, content ...string) Message {
// 	var res []Content
// 	for _, str := range content {
// 		t := Content{Data: []byte(str)}
// 		res = append(res, t)
// 	}
// 	return Message{
// 		Command: command,
// 		Content: res,
// 	}
// }

// // Copy - копирует одно сообщение в другое
// func Copy(dst, src *Message) error {
// 	if dst == nil || src == nil {
// 		return fmt.Errorf("Message is nil")
// 	}
// 	dst.Command = src.Command
// 	dst.SessionID = src.SessionID
// 	copy(dst.Content, src.Content)
// 	return nil
// }

// // Clear - очищает сообщение
// func Clear(m *Message) {
// 	m.Command = 0
// 	m.SessionID = 0
// 	m.Content = m.Content[:0]
// }

// ClientDescriptor - base entity for client to client messanger
type ClientDescriptor struct {
	ID           uint64    `json:"ID"`
	Name         string    `json:"Name"`
	SecretKey    string    `json:"Key"`
	RegisterDate time.Time `json:"Registered"`
	AllowedUsers []string  `json:"Alloved"` /*Список разрешений для пользователя*/
}
