package stat

import (
	cf "blabu/c2cService/configuration"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	log "blabu/c2cService/logWrapper"
)

// OldIPAddrTime - время последней активности после которого можно сбросить счетчик подключений
var OldIPAddrTime time.Duration

type AllIP struct {
	IP               string    `json:IP`
	count            uint32    `json:Count`
	timeLastActivity time.Time `json:LastTime`
}

// ipAddreses - мапа всех адресов и колличества входов за последние OldIPAddrTime времени
var ipAddreses map[string]AllIP
var rwM sync.RWMutex

func init() {
	ipAddreses = make(map[string]AllIP, 1)
	rwM = sync.RWMutex{}
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
	rwM.Lock()
	defer rwM.Unlock()
	res := ipAddreses[addr]
	if time.Since(res.timeLastActivity) > OldIPAddrTime {
		res.count = 1
	} else {
		res.count++
	}
	res.IP = addr
	res.timeLastActivity = time.Now()
	ipAddreses[addr] = res
	return res.count
}

// GetAllIPMap return json for visualisate all IP addresses
func GetAllIPMap() []byte {
	res := make([]byte, 0, 256)
	rwM.RLock()
	defer rwM.RUnlock()
	res = append(res, '[')
	for _, value := range ipAddreses {
		if r, er := json.Marshal(value); er == nil {
			res = append(res, r...)
			res = append(res, ',')
		}
	}
	if res[len(res)-1] == ',' {
		res[len(res)-1] = ']'
	} else {
		res = append(res, ']')
	}
	return res
}
