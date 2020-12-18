package server

import (
	"net"
	"time"

	"github.com/blabu/egeonC2cService/configuration"
	"github.com/blabu/c2cLib/parser"
)

// StartNewSession - инициализирует все и стартует сессию
func StartNewSession(conn net.Conn, dT time.Duration) {
	p := parser.CreateEmptyParser(uint64(configuration.Config.MaxPacketSize) * 1024)
	conn.SetReadDeadline(time.Now().Add(dT))
	if buf, err := p.ReadPacketHeader(conn); err == nil {
		s := BidirectSession{
			Duration: dT,
			Tm:       time.NewTimer(dT),
			logic:    CreatePanicCoverLogic(CreateReadWriteMainLogic(p, dT)),
			netReq:   buf,
		}
		s.Run(conn, p)
		s.Tm.Stop()
		s.logic.Close()
	}
	conn.Close()
}
