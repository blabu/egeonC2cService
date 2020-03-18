package dto

import "time"

//ClientLimits - client base statistics
type ClientLimits struct {
	ID                  uint64        `json:"ID"`
	LastActivity        time.Time     `json:"LastActivity"`
	TransmiteBytes      uint64        `json:"Transmit"`
	ReceiveBytes        uint64        `json:"Receive"`
	MaxReceivedBytes    uint64        `json:"MaxRx"`
	MaxTransmittedBytes uint64        `json:"MaxTx"`
	LimitExpiration     time.Time     `json:"LimitExpiration"`
	TimePeriod          time.Duration `json:"Period"`
	Balance             float64       `json:"Balance"`
	Rate                float64       `json:"Rate"`
}
