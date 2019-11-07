package stat

import (
	cf "blabu/c2cService/configuration"
	"strconv"
	"sync"
	"time"

	log "blabu/c2cService/logWrapper"
)

// OldIPAddrTime - время последней активности после которого можно сбросить счетчик подключений
var OldIPAddrTime time.Duration

type allIP struct {
	count            uint32
	timeLastActivity time.Time
}

// ipAddreses - мапа всех адресов и колличества входов за последние OldIPAddrTime времени
var ipAddreses map[string]allIP

func init() {
	ipAddreses = make(map[string]allIP, 1)
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
func AddIPAddres(addr string) uint32 {
	log.Info(addr)
	rwM := sync.RWMutex{}
	rwM.Lock()
	defer rwM.Unlock()
	res := ipAddreses[addr]
	if time.Since(res.timeLastActivity) > OldIPAddrTime {
		res.count = 1

	} else {
		res.count++
	}
	res.timeLastActivity = time.Now()
	ipAddreses[addr] = res
	return res.count
}
