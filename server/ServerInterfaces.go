package server

import (
	"context"
	"io"
	"net"

	"github.com/blabu/egeonC2cService/dto"
)

var lastSessionID uint32

/*
Session - основной интерфйес для создания соединений
Управляет своим соединением.
Отключает соединение по таймоуту,
или в случае критической ошибки чтения из соединения, или записи туда
Инициализирует парсер для соединения по первому сообщению (не обязательно полному)
Создает сущность для работ с соединением (MainLogic)
Проверяет целостность сообщения и передает его в MainLogic (где осуществляется парсинг его и передача в бизнес логику)
*/
type Session interface {
	//Run - функция управления сеансом с пользователем
	Run(Connect net.Conn)
}

// ClientReader - базовый интерфейс для чтения из логики (вывод данных наружу)
type ClientReader interface {
	Read(context.Context, dto.ServerReadHandler)
}

// MainLogicIO - основоной интерфейс логики взаимодействия сервера с логикой приложения
type MainLogicIO interface {
	ClientReader
	io.WriteCloser
}
