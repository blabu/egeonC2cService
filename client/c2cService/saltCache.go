package c2cService

import "sync"

var saltCache map[string]int // Кеш случайных солей и кол-во раз котрое они встречались там
var saltMtx sync.RWMutex

func init() {
	saltCache = make(map[string]int)
}

//CheckSalt Функция возвращает сколько раз использовалась одна и та же соль при авторизации
func CheckSalt(salt string) int {
	saltMtx.RLock()
	cnt, ok := saltCache[salt]
	saltMtx.RUnlock()
	if !ok {
		cnt = 1
	} else {
		cnt++
	}
	saltMtx.Lock()
	saltCache[salt] = cnt
	saltMtx.Unlock()
	return cnt
}
