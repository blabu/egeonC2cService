package main

import (
	log "blabu/c2cService/logWrapper"
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

// Структура, реализует интерфейс net.Conn, но для UDP
type udpConn struct {
	sessionIdentifier string
	aliveTimePoint    time.Time      /*Время жизни созданного соединения*/
	readDuration      time.Duration  /*Таймоут чтения*/
	readDeadLine      *time.Timer    /*Таймер чтения*/
	connection        net.PacketConn /*Интерфейс взаимодействия с сетью куда пишем ответ*/
	addr              net.Addr       /*Адрес куда писать ответ*/
	buffer            chan []byte    /*Канал куда listener отправляет данные в случае их получения*/
	localReadBuff     []byte         /*Буфер куда будем куда складываем прочтенные данные*/
	parent            *listener      /*Родитель создавший этот connection*/
}

func (u *udpConn) Read(b []byte) (n int, err error) {
	if u.readDeadLine == nil {
		u.readDeadLine = time.NewTimer(u.readDuration)
	} else {
		u.readDeadLine.Reset(u.readDuration)
	}
	defer u.readDeadLine.Stop()
	if len(u.localReadBuff) > 0 {
		copy(b, u.localReadBuff)
		if len(u.localReadBuff) > len(b) {
			u.localReadBuff = u.localReadBuff[len(b):]
			return len(b), nil
		}
		size := len(u.localReadBuff)
		u.localReadBuff = u.localReadBuff[:0]
		return size, nil
	}
	var ok bool
	select {
	case <-u.readDeadLine.C:
		return 0, fmt.Errorf("Read timeout")
	case u.localReadBuff, ok = <-u.buffer:
		if !ok {
			return 0, fmt.Errorf("Connection closed")
		}
		log.Tracef("Received %d bytes transmit to client", len(b))
		copy(b, u.localReadBuff)
		if len(u.localReadBuff) > len(b) {
			u.localReadBuff = u.localReadBuff[len(b):]
		} else {
			u.localReadBuff = u.localReadBuff[:0]
		}
		return len(b), nil
	}
}

func (u *udpConn) Write(b []byte) (n int, err error) {
	log.Trace("Write to session ", u.sessionIdentifier)
	b = append([]byte(u.sessionIdentifier), b...)
	u.connection.WriteTo(b, u.addr)
	log.Info("Write success")
	return 0, nil
}

func (u *udpConn) Close() error {
	close(u.buffer)
	if u.readDeadLine != nil {
		u.readDeadLine.Stop()
	}
	u.parent.deleteConnection(u.sessionIdentifier)
	return nil
}

func (u *udpConn) LocalAddr() net.Addr {
	return u.addr
}

func (u *udpConn) RemoteAddr() net.Addr {
	return u.addr
}

func (u *udpConn) SetDeadline(t time.Time) error {
	u.connection.SetDeadline(t)
	return nil
}

func (u *udpConn) SetReadDeadline(t time.Time) error {
	u.readDuration = t.Sub(time.Now())
	if u.readDuration < 0 {
		u.readDuration = 24 * time.Hour
		return fmt.Errorf("Read deadline timer not setup")
	}
	return nil
}

func (u *udpConn) SetWriteDeadline(t time.Time) error {
	return u.connection.SetWriteDeadline(t)
}

//==============================================================================================
//==============================================================================================
//==============================================================================================
//==============================================================================================
// Структура реализует интерфейс net.Listener но для udp
type listener struct {
	allUDP      map[string]*udpConn // allUDP - Мапа, которая помнит все живые UDP соединения
	allUDPmtx   sync.RWMutex        //allUDPmtx - По некоторым исследованиям https://habr.com/ru/post/338718/ and https://wrfly.kfd.me/posts/rwmutex-and-sync.map/ мьютекс эффетивнее
	conn        net.PacketConn
	receivedBuf []byte
}

// NewUDPListener - вохвращает интерфейс net.Listener для UDP соединений
func NewUDPListener(bufSize uint, udpPort string) (net.Listener, error) {
	con, err := net.ListenPacket("udp", udpPort)
	if err != nil {
		return nil, err
	}
	return &listener{
		allUDP:      make(map[string]*udpConn),
		conn:        con,
		receivedBuf: make([]byte, bufSize),
	}, nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789+-*()/")

func randStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (l *listener) deleteConnection(sessionIdentifier string) {
	l.allUDPmtx.Lock()
	defer l.allUDPmtx.Unlock()
	delete(l.allUDP, sessionIdentifier)
}

// Accept читает данные с udp порта и по его IP адресу находит соединение. Если соединение найдено передает прочтенное туда
func (l *listener) Accept() (net.Conn, error) {
	for {
		n, addr, err := l.conn.ReadFrom(l.receivedBuf)
		if err != nil {
			return nil, err
		}
		log.Tracef("Receive %s from %s", string(l.receivedBuf[:n]), addr.String())
		var session string
		sIndex := bytes.IndexByte(l.receivedBuf[:n], byte('$'))
		if sIndex < 0 || sIndex == 0 {
			session = randStringRunes(12)
			sIndex = 0
			log.Info("Generate new session key ", session)
		} else {
			session = string(l.receivedBuf[:sIndex])
			log.Info("Receive some session key ", session)
		}
		var conn *udpConn
		var ok bool
		l.allUDPmtx.RLock()
		conn, ok = l.allUDP[session]
		l.allUDPmtx.RUnlock()
		if ok {
			if conn.aliveTimePoint.After(time.Now()) {
				log.Trace("Connection already exist")
				conn.addr = addr
				conn.sessionIdentifier = session
				conn.buffer <- l.receivedBuf[sIndex:n]
				continue
			} else {
				log.Info("Old connection closed")
				conn.Close()
			}
		}
		log.Trace("Create new connection for session ", session)
		conn = &udpConn{
			sessionIdentifier: session,
			aliveTimePoint:    time.Now().Add(4 * time.Hour),
			readDuration:      time.Hour,
			connection:        l.conn,
			addr:              addr,
			buffer:            make(chan []byte, 8),
			parent:            l,
		}
		l.allUDPmtx.Lock()
		l.allUDP[session] = conn
		l.allUDPmtx.Unlock()
		conn.buffer <- l.receivedBuf[sIndex:n]
		return conn, nil
	}
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *listener) Close() error {
	for _, con := range l.allUDP { // FIXME состояние ГОНОК, но мьютекс нелязя - блокировка
		con.Close()
	}
	return l.conn.Close()
}

// Addr returns the listener's network address.
func (l *listener) Addr() net.Addr {
	return l.conn.LocalAddr()
}
