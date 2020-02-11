package server

import (
	"blabu/c2cService/parser"
	"blabu/c2cService/stat"
	"net"
	"time"
)

// StartNewSession - инициализирует все и стартует сессию
func StartNewSession(conn net.Conn, dT time.Duration, st *stat.Statistics) {
	req := make([]byte, 128)
	conn.SetReadDeadline(time.Now().Add(dT))
	if n, err := conn.Read(req); err == nil {
		if p, err := parser.InitParser(req[:n]); err == nil {
			st.NewConnection() // Регистрируем новое соединение
			start := time.Now()
			s := BidirectSession{
				Duration: dT,
				Tm:       time.NewTimer(dT),
				netReq:   req,
				logic: atomicMainLog{
					main: CreateReadWriteMainLogic(p, time.Second),
				},
			}
			s.Run(conn, p)
			s.logic.Get().Close()
			st.CloseConnection()
			st.SetConnectionTime(time.Since(start))
		}
	}
	conn.Close()
}
