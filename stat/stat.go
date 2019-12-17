package stat

import (
	"html/template"
	"sync/atomic"
	"time"

	log "blabu/c2cService/logWrapper"
)

// S_VERSION - Версия сервера
const S_VERSION = "v2.0.1"

// Statistics - базовые метрики работы сервера
type Statistics struct {
	maxTimeForOneConnection time.Duration `json `
	maxResponceTime         int64
	templStat               *template.Template
	timeUp                  time.Time
	nowConnected            int32
	maxCuncurentConnection  int32
	allConnection           int32
}

// SetResponceTime - Передает в статистику максимальное значение времени ответа и команда на которую было потрачено столько времени
func (s *Statistics) SetResponceTime(responceTime time.Duration) {
	t := responceTime.Nanoseconds()
	if s.maxResponceTime < t {
		atomic.StoreInt64(&s.maxResponceTime, t)
	}
}

// NewConnection - атомарно добавляет в статистику новое соединение
func (s *Statistics) NewConnection() {
	atomic.AddInt32(&(s.allConnection), 1)
	atomic.AddInt32(&(s.nowConnected), 1)
	now := atomic.LoadInt32(&s.nowConnected)
	if now > s.maxCuncurentConnection {
		atomic.StoreInt32(&s.maxCuncurentConnection, now)
	}
}

//CloseConnection - отображет в статистеке закрытие соединения
func (s *Statistics) CloseConnection() {
	log.Info("Close connection")
	atomic.AddInt32(&s.nowConnected, -1)
}

//SetConnectionTime - Проверяет и сохраняет при необходимости максимальное время сессии
func (s *Statistics) SetConnectionTime(dt time.Duration) {
	if s.maxTimeForOneConnection < dt {
		atomic.StoreInt64((*int64)(&s.maxTimeForOneConnection), dt.Nanoseconds())
	}
}

// CreateStatistics - создает объект со статистикой
func CreateStatistics() Statistics {
	ipAddrInit()
	return Statistics{
		timeUp: time.Now(),
	}
}

func (s *Statistics) GetJsonStat() []byte {

}
