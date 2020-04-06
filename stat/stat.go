package stat

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	log "blabu/c2cService/logWrapper"
)

// S_VERSION - Версия сервера
const S_VERSION = "v2.3.0"

// Statistics - базовые метрики работы сервера
type Statistics struct {
	ServerVersion           string           `json:"version"`
	MaxTimeForOneConnection time.Duration    `json:"oneConnectionTimeout"`
	MaxResponceTime         int64            `json:"maxResponce"`
	TimeUP                  time.Time        `json:"timeUP"`
	NowConnected            int32            `json:"nowConnected"`
	MaxCuncurentConnection  int32            `json:"maxConcurentConnection"`
	AllConnection           int32            `json:"allConnection"`
	IPAddresses             map[string]AllIP `json:"allIP"`
	rwM                     sync.RWMutex
}

// SetResponceTime - Передает в статистику максимальное значение времени ответа и команда на которую было потрачено столько времени
func (s *Statistics) SetResponceTime(responceTime time.Duration) {
	t := responceTime.Nanoseconds()
	if s.MaxResponceTime < t {
		atomic.StoreInt64(&s.MaxResponceTime, t)
	}
}

// NewConnection - атомарно добавляет в статистику новое соединение
func (s *Statistics) NewConnection() {
	atomic.AddInt32(&(s.AllConnection), 1)
	atomic.AddInt32(&(s.NowConnected), 1)
	now := atomic.LoadInt32(&s.NowConnected)
	if now > s.MaxCuncurentConnection {
		atomic.StoreInt32(&s.MaxCuncurentConnection, now)
	}
}

//CloseConnection - отображет в статистеке закрытие соединения
func (s *Statistics) CloseConnection() {
	log.Info("Close connection")
	atomic.AddInt32(&s.NowConnected, -1)
}

//SetConnectionTime - Проверяет и сохраняет при необходимости максимальное время сессии
func (s *Statistics) SetConnectionTime(dt time.Duration) {
	if s.MaxTimeForOneConnection < dt {
		atomic.StoreInt64((*int64)(&s.MaxTimeForOneConnection), dt.Nanoseconds())
	}
}

// CreateStatistics - создает объект со статистикой
func CreateStatistics() Statistics {
	ipAddrInit()
	return Statistics{
		ServerVersion: S_VERSION,
		TimeUP:        time.Now(),
		IPAddresses:   make(map[string]AllIP, 1),
	}
}

func (s *Statistics) GetJsonStat() []byte {
	res, err := json.Marshal(*s)
	if err != nil {
		log.Warning(err.Error())
		return []byte{}
	}
	return res
}
