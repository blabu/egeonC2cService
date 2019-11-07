package server

import (
	"blabu/c2cService/stat"
	"net"
)

var lastSessionID uint32

/*
Connector - основной интерфйес для создания соединений
Управляет своим соединением.
Отключает соединение по таймоуту,
или в случае критической ошибки чтения из соединения, или записи туда
Инициализирует парсер для соединения по первому сообщению (не обязательно полному)
Создает сущность для работ с соединением (MainLogic)
Проверяет целостность сообщения и передает его в MainLogic (где осуществляется парсинг его и передача в бизнес логику)
*/
type Connector interface {
	//ManageSession - функция управления сеансом с пользователем
	ManageSession(Connect net.Conn, stat *stat.Statistics)
}

// ClientReader - базовый интерфейс для чтения из логики (вывод данных наружу)
type ClientReader interface {
	Read(handler func([]byte, error))
}

// ClientWriter - базовый интерфейс для записи в логику (ввод данных из вне)
type ClientWriter interface {
	Write([]byte) error
}

// MainLogicIO - основоной интерфейс логики взаимодействия сервера с логикой приложения
type MainLogicIO interface {
	ClientReader
	ClientWriter
	Close()
}
