package connectorC2c

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"

	"github.com/blabu/egeonC2cService/dto"
	"github.com/blabu/egeonC2cService/parser"
)

const proto = 1

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type C2cConnection struct {
	chunkSize   uint64
	user        string
	pass        string
	conn        net.Conn
	receiveBuff []byte
	p           parser.Parser
}

type IC2cConnection interface {
	Read() (from string, command uint16, data []byte)
	Write(to string, command uint16, data []byte) error
	Connect(name string) error
	Register() error
	Init() error
	Ping() error
	Close() error
}

func NewC2cConnection(user, pass string, conn net.Conn, maxSize uint64) (IC2cConnection, error) {
	p := parser.CreateEmptyParser(maxSize)
	res := &C2cConnection{
		chunkSize:   maxSize,
		user:        user,
		pass:        pass,
		conn:        conn,
		p:           p,
		receiveBuff: make([]byte, p.GetMinimumDataSize(), maxSize),
	}
	return res, nil
}

func (c *C2cConnection) Write(to string, command uint16, data []byte) error {
	buf, err := c.p.FormMessage(dto.Message{
		Command: command,
		Proto:   proto,
		Jmp:     3,
		From:    c.user,
		To:      to,
		Content: data,
	})
	if err != nil {
		return err
	}
	_, err = c.conn.Write(buf)
	return err
}

func (c *C2cConnection) Read() (from string, command uint16, data []byte) {
	reader := bufio.NewReader(c.conn)
	n, err := reader.Read(c.receiveBuff)
	if err != nil {
		return "", 0, nil
	}
	c.receiveBuff = c.receiveBuff[:n]
	restSize, err := c.p.IsFullReceiveMsg(c.receiveBuff)
	if err != nil {
		return "", 0, nil
	}
	if restSize != 0 {
		resp := make([]byte, restSize)
		c.conn.SetReadDeadline(time.Now().Add(time.Duration(restSize) * 10 * time.Millisecond))
		_, err := io.ReadFull(reader, resp)
		if err != nil {
			return "", 0, nil
		}
		c.receiveBuff = append(c.receiveBuff, resp...)
	}
	m, err := c.p.ParseMessage(c.receiveBuff)
	if err != nil {
		return "", 0, nil
	}
	c.receiveBuff = c.receiveBuff[:c.p.GetMinimumDataSize()]
	return m.From, m.Command, m.Content
}

func (c *C2cConnection) Register() error {
	sign := sha256.Sum256([]byte(c.user + c.pass))
	signature := base64.StdEncoding.EncodeToString(sign[:])
	if err := c.Write("0", dto.RegisterCOMMAND, []byte(signature)); err != nil {
		return err
	}
	_, cmd, data := c.Read()
	if data == nil || cmd != dto.RegisterCOMMAND {
		return errors.New("Can not register. Error while read")
	}
	return nil
}

func (c *C2cConnection) Init() error {
	temp := sha256.Sum256([]byte(c.user + c.pass))
	credentials := base64.StdEncoding.EncodeToString(temp[:])
	salt := randStringRunes(32)
	resSign := sha256.Sum256([]byte(c.user + salt + credentials))
	signature := base64.StdEncoding.EncodeToString(resSign[:])
	if err := c.Write("0", dto.InitByNameCOMMAND, []byte(salt+";"+signature)); err != nil {
		return err
	}
	_, cmd, data := c.Read()
	if data == nil || cmd != dto.InitByNameCOMMAND {
		return errors.New("Can not init. Errors while read")
	}
	if bytes.Index(data, []byte("INIT OK")) < 0 {
		return fmt.Errorf("Bad init %s", data)
	}
	return nil
}

func (c *C2cConnection) Connect(name string) error {
	if err := c.Write(name, dto.ConnectByNameCOMMAND, nil); err != nil {
		return err
	}
	_, cmd, data := c.Read()
	if data == nil || cmd != dto.ConnectByNameCOMMAND {
		return errors.New("Can not connect. Errors while read")
	}
	if bytes.Index(data, []byte("CONNECT OK")) < 0 {
		return fmt.Errorf("Bad connection error")
	}
	return nil
}

func (c *C2cConnection) Ping() error {
	err := c.Write("0", dto.PingCOMMAND, nil)
	return err
}

func (c *C2cConnection) Close() error {
	return c.conn.Close()
}
