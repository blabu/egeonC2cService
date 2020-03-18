package dto

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
