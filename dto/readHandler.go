package dto

// ReadHandler - стандартный обработчик для чтения пакета
type ReadHandler func(msg Message, err error) error
