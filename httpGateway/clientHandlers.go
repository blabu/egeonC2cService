package httpGateway

import (
	cf "blabu/c2cService/configuration"
	"blabu/c2cService/data/c2cData"
	log "blabu/c2cService/logWrapper"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"
)

func getClients(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	key := v.Get("key")
	if _, err := checkKey(key, clients); err == nil {
		db := c2cData.GetBoltDbInstance()
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		res := make([]byte, 0, 32)
		db.ForEach(c2cData.Clients, func(key []byte, value []byte) error {
			res = append(res, value...)
			res = append(res, ',')
			return nil
		})
		res = res[:len(res)-1]
		w.Write([]byte("["))
		w.Write(res)
		w.Write([]byte("]"))
	} else {
		httpError{statusCode: http.StatusMethodNotAllowed, err: err}.ServeHTTP(w, r)
	}
}

func getClient(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if _, err := checkKey(key, client); err != nil {
		httpError{statusCode: http.StatusMethodNotAllowed, err: err}.ServeHTTP(w, r)
		return
	}
	var id uint64
	var err error
	storage := c2cData.GetBoltDbInstance()
	idStr := r.URL.Query().Get("id")
	nameStr := r.URL.Query().Get("name")
	if len(idStr) == 0 {
		id, err = storage.GetClientID(nameStr)
		if err != nil {
			httpError{statusCode: http.StatusBadRequest, err: err}.ServeHTTP(w, r)
			return
		}
	} else {
		id, err = strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			httpError{statusCode: http.StatusBadRequest, err: err}.ServeHTTP(w, r)
			return
		}
	}
	cl, err := storage.GetClient(id)
	if err != nil {
		httpError{statusCode: http.StatusBadRequest, err: err}.ServeHTTP(w, r)
		return
	}
	log.Infof("client finded %v", *cl)
	data, _ := json.Marshal(cl)
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

/*
key parameter is required
in post for need name and pass fields
Example:
curl -i -cacert ./cert.pem --insecure -X POST https://localhost:6060/api/v1/client?key=123456 -d "name=userName&pass=securePass"
*/
func insertClient(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if perm, err := checkKey(key, client); err == nil && perm.IsWritable {
		clType, _ := strconv.ParseUint(cf.GetConfigValueOrDefault("ClientType", "1"), 10, 16)
		err = r.ParseForm()
		if err != nil {
			httpError{statusCode: http.StatusBadRequest, err: errors.New("Can not parse parameters")}.ServeHTTP(w, r)
			return
		}
		name := r.FormValue("name")
		pass := r.FormValue("pass")
		storage := c2cData.GetBoltDbInstance()
		cl, err := storage.GenerateClient(c2cData.ClientType(clType), name, pass)
		if err != nil {
			httpError{statusCode: http.StatusBadRequest, err: err}.ServeHTTP(w, r)
		}
		if err = storage.SaveClient(cl); err != nil {
			httpError{statusCode: http.StatusBadRequest, err: err}.ServeHTTP(w, r)
		}
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		data, _ := json.Marshal(cl)
		w.Write(data)
	} else {
		httpError{statusCode: http.StatusMethodNotAllowed, err: err}.ServeHTTP(w, r)
	}
}

/*
Set limits for client
request param:
1. key is required
2. name or id
POST form parameters
balance
rate
maxRx
maxTx
period in seconds
Example
curl -i --insecure -X POST "https://localhost:6060/api/v1/limits?key=qwertyu&name=userName" -d "balance=1000&rate=100&period=3600"
curl -i --insecure -X POST "https://localhost:6060/api/v1/limits?key=qwertyu&name=blabu" -d "balance=-20.3"
*/
func limitsHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	perm, err := checkKey(key, limits)
	if err != nil {
		httpError{statusCode: http.StatusMethodNotAllowed, err: errors.New("Operation not permitted")}.ServeHTTP(w, r)
		return
	}
	storage := c2cData.GetBoltDbInstance()
	var id uint64
	idStr := r.URL.Query().Get("id")
	if len(idStr) == 0 {
		name := r.URL.Query().Get("name")
		if id, err = storage.GetClientID(name); err != nil {
			httpError{statusCode: http.StatusBadRequest, err: err}.ServeHTTP(w, r)
			return
		}
	}
	limit, findLimitError := storage.GetStat(id)
	switch r.Method {
	case http.MethodGet:
		if findLimitError != nil {
			httpError{statusCode: http.StatusBadRequest, err: findLimitError}.ServeHTTP(w, r)
			return
		}
		if res, err := json.Marshal(limit); err != nil {
			httpError{statusCode: http.StatusInternalServerError, err: errors.New("Can not get data from base")}.ServeHTTP(w, r)
		} else {
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write(res)
		}
	case http.MethodPost:
		if perm.IsWritable {
			if err := r.ParseForm(); err != nil {
				httpError{statusCode: http.StatusBadRequest, err: errors.New("Can not parse form")}.ServeHTTP(w, r)
			}
			balance := r.FormValue("balance")
			rate := r.FormValue("rate")
			maxRx := r.FormValue("maxRx")   // in bytes
			maxTx := r.FormValue("maxTx")   // in bytes
			period := r.FormValue("period") // in seconds
			if findLimitError != nil {
				limit.ID = id
				limit.LastActivity = time.Now()
			}
			if len(balance) != 0 {
				if b, e := strconv.ParseFloat(balance, 64); e == nil {
					limit.Balance += b
				}
			}
			if len(rate) != 0 {
				if r, e := strconv.ParseFloat(rate, 64); e == nil {
					limit.Rate = r
				}
			}
			if len(maxRx) != 0 {
				if m, e := strconv.ParseUint(maxRx, 10, 64); e == nil {
					limit.MaxReceivedBytes = m
				}
			}
			if len(maxTx) != 0 {
				if m, e := strconv.ParseUint(maxTx, 10, 64); e == nil {
					limit.MaxTransmittedBytes = m
				}
			}
			if len(period) != 0 {
				if p, e := strconv.ParseInt(period, 10, 64); e == nil {
					limit.TimePeriod = time.Duration(p) * time.Second
					if findLimitError != nil {
						limit.LimitExpiration = limit.LastActivity.Add(limit.TimePeriod)
					}
				}
			}
			storage.UpdateStat(&limit)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			d, _ := json.Marshal(limit)
			w.Write(d)
		} else {
			httpError{statusCode: http.StatusForbidden, err: errors.New("Operation not permitted")}.ServeHTTP(w, r)
		}
	}
}
