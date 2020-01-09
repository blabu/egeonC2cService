package c2cService

import (
	"strconv"
)

var saltCache map[string]int // Кеш случайных солей и кол-во раз котрое они встречались там
// var saltMtx sync.RWMutex

func init() {
	saltCache = make(map[string]int)
}

// CheckSaltByUserName Функция возвращает сколько раз использовалась одна и та же соль
// при авторизации конкретного пользователя
func CheckSaltByUserName(name, salt string) int {
	s := name + salt
	cnt, ok := saltCache[s]
	if !ok {
		cnt = 1
	} else {
		cnt++
	}
	saltCache[s] = cnt
	return cnt
}

// CheckSaltByID Функция возвращает сколько раз использовалась одна и та же соль
// при авторизации конкретного пользователя
func CheckSaltByID(ID uint64, salt string) int {
	s := strconv.FormatUint(ID, 16) + salt
	cnt, ok := saltCache[s]
	if !ok {
		cnt = 1
	} else {
		cnt++
	}
	saltCache[s] = cnt
	return cnt
}
