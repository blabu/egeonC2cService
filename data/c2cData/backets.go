package c2cData

const (
	Names        = "nameByID"     // список имен с ключем по ID
	Clients      = "clients"      // Непосредственно сами клиенты с ключем по ID
	MaxClientID  = "maxClientID"  // Максимально выданный в системе идентификатор
	ClientLimits = "clientLimits" // Ограничение по трафику и времени работы клиентов
	Permission   = "permission"   // Ограничения по уровню доступа по ключам (нужно для Web API)
)
