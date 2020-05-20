package savemsgservice

import (
	"blabu/c2cService/client"
	"blabu/c2cService/data/c2cdata"
	"blabu/c2cService/dto"
	"strconv"
	"time"

	log "blabu/c2cService/logWrapper"
)

type saveMsgClient struct {
	db     c2cdata.DB
	client client.ReadWriteCloser
}

func NewDecorator(db c2cdata.DB, client client.ReadWriteCloser) client.ReadWriteCloser {
	return &saveMsgClient{
		db:     db,
		client: client,
	}
}

// SaveMsgFilter - return true if message need save
func (s *saveMsgClient) SaveMsgFilter(msg *dto.Message) bool {
	return s.client.GetID() != 0 && len(msg.Content) > 0 &&
		(msg.Command == client.DataCOMMAND || msg.Command == client.PropertiesCOMMAND)
}

func (s *saveMsgClient) Write(msg *dto.Message) error {
	err := s.client.Write(msg)
	if err != nil && s.SaveMsgFilter(msg) {
		toID, er := strconv.ParseUint(msg.To, 10, 64)
		if er != nil {
			toID, err = s.db.GetClientID(msg.To)
		}
		id, e := s.db.Add(toID, dto.UnSendedMsg{
			Proto:   msg.Proto,
			Command: msg.Command,
			From:    msg.From,
			Content: msg.Content,
		})
		if e == nil {
			msg.ID = id
		}
		//TODO return err!!!!
	}
	return err
}

func (s *saveMsgClient) Read(dt time.Duration, handler func(msg dto.Message, err error)) {
	s.client.Read(dt,
		func(msg dto.Message, err error) {
			handler(msg, err)
			userID := s.client.GetID()
			for m, e := s.db.GetNext(userID); e == nil && userID != 0; m, e = s.db.GetNext(userID) {
				log.Tracef("Try send to %x from %s unordered message %d", userID, m.From, m.ID)
				handler(dto.Message{
					ID:      m.ID,
					From:    m.From,
					To:      strconv.FormatUint(userID, 16),
					Command: m.Command,
					Proto:   m.Proto,
					Content: m.Content,
					Jmp:     1,
				}, nil)
				s.db.IsSended(userID, m.ID)
				userID = s.client.GetID()
			}

		})
}

func (s *saveMsgClient) Close() error {
	return s.client.Close()
}

func (s *saveMsgClient) GetID() uint64 {
	return s.client.GetID()
}
