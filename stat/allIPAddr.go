package stat

import (
	cf "blabu/c2cService/configuration"
	"strconv"
	"time"

	log "blabu/c2cService/logWrapper"
)

// OldIPAddrTime - время последней активности после которого можно сбросить счетчик подключений
var OldIPAddrTime time.Duration

type AllIP struct {
	IP               string    `json:"IP"`
	Count            uint32    `json:"Count"`
	TimeLastActivity time.Time `json:"LastTime"`
}

func ipAddrInit() {
	v, err := cf.GetConfigValue("OldIPAddrTimeout")
	if err != nil {
		OldIPAddrTime = 12 * time.Hour
	} else {
		minutes, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			OldIPAddrTime = 12 * time.Hour
		} else {
			OldIPAddrTime = time.Duration(minutes) * time.Minute
		}
	}
}

// AddIPAddres - добавляет к кол-ву подключений от указанного Ip адреса "1" и возвращает полученное значение
func (s *Statistics) AddIPAddres(addr string) uint32 {
	log.Info(addr)
	s.rwM.Lock()
	defer s.rwM.Unlock()
	res := s.IPAddresses[addr]
	if time.Since(res.TimeLastActivity) > OldIPAddrTime {
		res.Count = 1
	} else {
		res.Count++
	}
	res.IP = addr
	res.TimeLastActivity = time.Now()
	s.IPAddresses[addr] = res
	return res.Count
}
