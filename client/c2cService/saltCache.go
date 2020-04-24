package c2cService

import (
	"strconv"
	"strings"
	"sync"
)

var saltCache map[string]int // Кеш случайных солей и кол-во раз котрое они встречались там
var saltMtx sync.RWMutex

func init() {
	saltCache = make(map[string]int)
}

// CheckSaltByUserName Функция возвращает сколько раз использовалась одна и та же соль
// при авторизации конкретного пользователя
func CheckSaltByUserName(name, salt string) int {
	var s strings.Builder
	s.WriteString(name)
	s.WriteString(salt)
	saltMtx.RLock()
	cnt, ok := saltCache[s.String()]
	saltMtx.RUnlock()
	if !ok {
		cnt = 1
	} else {
		cnt++
	}
	saltMtx.Lock()
	saltCache[s.String()] = cnt
	saltMtx.Unlock()
	return cnt
}

// CheckSaltByID Функция возвращает сколько раз использовалась одна и та же соль
// при авторизации конкретного пользователя
func CheckSaltByID(ID uint64, salt string) int {
	var s strings.Builder
	s.WriteString(strconv.FormatUint(ID, 16))
	s.WriteString(salt)
	saltMtx.RLock()
	cnt, ok := saltCache[s.String()]
	saltMtx.RUnlock()
	if !ok {
		cnt = 1
	} else {
		cnt++
	}
	saltMtx.Lock()
	saltCache[s.String()] = cnt
	saltMtx.Unlock()
	return cnt
}
