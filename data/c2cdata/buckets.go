package c2cdata

const (
	Names        = "nameByID"     // список имен с ключем по ID
	Clients      = "clients"      // Непосредственно сами клиенты с ключем по ID
	UnsededMsg   = "unsended"     // Не отправленные сообщения для каждого пользователя
	MaxClientID  = "maxClientID"  // Максимально выданный в системе идентификатор
	ClientLimits = "clientLimits" // Ограничение по трафику и времени работы клиентов
	Permission   = "permission"   // Ограничения по уровню доступа по ключам (нужно для Web API)
)
