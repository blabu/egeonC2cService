package trafficClient

import (
	cf "blabu/c2cService/configuration"
	"blabu/c2cService/data/c2cData"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"errors"
	"strconv"
	"time"
)

func updateLimits(stat *dto.ClientStat) error {
	stat.LastActivity = time.Now()
	if stat.Balance < 0.0 {
		return errors.New("Not enough balance")
	}
	if (stat.LimitExpiration.Before(time.Now())) ||
		(stat.MaxReceivedBytes > 0 && stat.MaxTransmittedBytes > 0 &&
			stat.ReceiveBytes > stat.MaxReceivedBytes && stat.TransmiteBytes > stat.MaxTransmittedBytes) {
		stat.Balance -= stat.Rate
		stat.LimitExpiration = time.Now().Add(stat.TimePeriod)
		stat.ReceiveBytes = 0
		stat.TransmiteBytes = 0
	}
	return nil
}

func initStat(from string, storage c2cData.DB) (dto.ClientStat, error) {
	var e error
	var stat dto.ClientStat
	if stat.ID, e = strconv.ParseUint(from, 16, 64); e != nil {
		if stat.ID, e = storage.GetClientID(from); e != nil {
			log.Warning("Undefine client when try init it ", from)
			stat.ID = 0
			return stat, e
		}
	}
	log.Trace("Try find client stat by ID ", stat.ID)
	s, err := storage.GetStat(stat.ID)
	if err != nil {
		s.ID = stat.ID
		s.Rate, _ = strconv.ParseFloat(cf.GetConfigValueOrDefault("DefaultPacketPrice", "0.0"), 64)
	}
	return s, err
}
