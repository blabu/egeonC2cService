package c2cService

/*
Набор поддерживаемых команд
протого моста между клиентами
Обработка всех команд происходит в Write методе
*/
const (
	errorCOMMAND         uint16 = 1
	pingCOMMAND          uint16 = 2
	registerCOMMAND      uint16 = 3
	generateCOMMAND      uint16 = 4
	initByIDCOMMAND      uint16 = 5
	initByNameCOMMAND    uint16 = 6
	connectByIDCOMMAND   uint16 = 7
	connectByNameCOMMAND uint16 = 8
	dataCOMMAND          uint16 = 9
	destroyConCOMMAND    uint16 = 10
	propertiesCOMMAND    uint16 = 11
)
