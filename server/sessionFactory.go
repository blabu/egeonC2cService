package server

import (
	"net"
	"time"

	"github.com/blabu/egeonC2cService/parser"
)

const minHeaderSize = 128

// StartNewSession - инициализирует все и стартует сессию
func StartNewSession(conn net.Conn, dT time.Duration) {
	req := make([]byte, minHeaderSize)
	conn.SetReadDeadline(time.Now().Add(dT))
	if n, err := conn.Read(req); err == nil {
		if p, err := parser.InitParser(req[:n]); err == nil && p != nil {
			s := BidirectSession{
				Duration: dT,
				Tm:       time.NewTimer(dT),
				netReq:   req,
				logic:    CreateReadWriteMainLogic(p, dT),
			}
			s.Run(conn, p)
			s.logic.Close()
		}
	}
	conn.Close()
}
