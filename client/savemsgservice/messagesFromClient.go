package savemsgservice

import (
	"context"
	"fmt"
	"strconv"

	"github.com/blabu/egeonC2cService/client"
	"github.com/blabu/egeonC2cService/data"
	"github.com/blabu/c2cLib/dto"

	log "github.com/blabu/egeonC2cService/logWrapper"
)

type saveMsgClient struct {
	db     data.DB
	client client.ReadWriteCloser
}

func NewDecorator(db data.DB, client client.ReadWriteCloser) client.ReadWriteCloser {
	return &saveMsgClient{
		db:     db,
		client: client,
	}
}

// SaveMsgFilter - return true if message need save
func (s *saveMsgClient) SaveMsgFilter(msg *dto.Message) bool {
	return s.client.GetID() != 0 && len(msg.Content) > 0 &&
		(msg.Command == dto.SaveDataCOMMAND || msg.Command == dto.PropertiesCOMMAND)
}

// Write - пытаемся отправить сообщение клиенту.
// Если не получается, но клиент существует,
// а сообщение не пустое и важное (узнаем по команде), сохранием его в базе
func (s *saveMsgClient) Write(msg *dto.Message) error {
	err := s.client.Write(msg)
	if err != nil && s.SaveMsgFilter(msg) { // Если
		toID, er := strconv.ParseUint(msg.To, 10, 64)
		if er != nil {
			toID, err = s.db.GetClientID(msg.To)
		}
		if toID != 0 {
			id, e := s.db.Add(toID, dto.UnSendedMsg{
				Proto:   msg.Proto,
				Command: msg.Command,
				From:    msg.From,
				Content: msg.Content,
			})
			if e == nil {
				msg.ID = id
			}
			//TODO error handling
			log.Infof("Message %d from %s to %x is saved", msg.ID, msg.From, toID)
			return nil
		}
		return fmt.Errorf("Undefine client ID in %s", msg.To)
	}
	return err
}

func (s *saveMsgClient) Read(ctx context.Context, handler dto.ClientReadHandler) {
	s.client.Read(ctx,
		func(msg dto.Message, err error) error {
			clientError := handler(msg, err)
			if clientError == nil {
				userID := s.client.GetID()
				if userID > 0 {
					for m, e := s.db.GetNext(userID); e == nil; m, e = s.db.GetNext(userID) {
						log.Tracef("Try send to %x from %s unordered message %d", userID, m.From, m.ID)
						err := handler(dto.Message{
							ID:      m.ID,
							From:    m.From,
							To:      strconv.FormatUint(userID, 16),
							Command: m.Command,
							Proto:   m.Proto,
							Content: m.Content,
							Jmp:     1,
						}, nil)
						if err != nil {
							return err
						}
						s.db.IsSended(userID, m.ID)
					}
				}
			}
			return clientError
		})
}

func (s *saveMsgClient) Close() error {
	return s.client.Close()
}

func (s *saveMsgClient) GetID() uint64 {
	return s.client.GetID()
}
