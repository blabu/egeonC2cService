package httpGateway

import (
	cf "blabu/c2cService/configuration"
	c2cData "blabu/c2cService/data/c2cdata"
	log "blabu/c2cService/logWrapper"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func getClients(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	key := v.Get("key")
	if _, err := checkKey(key, client); err == nil {
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
		log.Debug(string(res))
		res = res[:len(res)-1]
		log.Debug(string(res))
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
	idStr, ok := mux.Vars(r)["id"]
	if !ok {
		httpError{statusCode: http.StatusBadGateway, err: errors.New("Undefined client id or name")}.ServeHTTP(w, r)
		return
	}
	if len(idStr) == 0 {
		httpError{statusCode: http.StatusBadGateway, err: errors.New("empty client identifier")}.ServeHTTP(w, r)
		return
	}
	id, err = strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		if id, err = c2cData.GetBoltDbInstance().GetClientID(idStr); err != nil {
			httpError{statusCode: http.StatusNotFound, err: err}.ServeHTTP(w, r)
			return
		}
	}
	cl, err := storage.GetClient(id)
	if err != nil {
		httpError{statusCode: http.StatusNotFound, err: err}.ServeHTTP(w, r)
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
curl -i -cacert ./cert.pem --insecure -X POST https://localhost:6060/api/v1/client?key=a1s2d3f4g5h6 -d "{\"name\":\"someUser2000\", \"pass\":\"securePass2000\"}"
*/
func insertClient(w http.ResponseWriter, r *http.Request) {
	type user struct {
		Name string `json:"name"`
		Pass string `json:"pass"`
	}
	key := r.URL.Query().Get("key")
	if perm, err := checkKey(key, client); err == nil && perm.IsWritable {
		clType, _ := strconv.ParseUint(cf.GetConfigValueOrDefault("ClientType", "1"), 10, 16)
		var u user
		tempData := make([]byte, r.ContentLength)
		io.ReadFull(r.Body, tempData)
		log.Trace(string(tempData))
		if err = json.Unmarshal(tempData, &u); err != nil {
			httpError{statusCode: http.StatusBadRequest, err: errors.New("Can not parse parameters")}.ServeHTTP(w, r)
			return
		}
		storage := c2cData.GetBoltDbInstance()
		cl, err := storage.GenerateClient(c2cData.ClientType(clType), u.Name, u.Pass)
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

func getLimits(w http.ResponseWriter, r *http.Request) {
	_, limit, findLimitError := getClientLimit(r)
	if findLimitError != nil {
		httpError{statusCode: http.StatusBadRequest, err: findLimitError}.ServeHTTP(w, r)
		return
	}
	if res, err := json.Marshal(limit); err != nil {
		httpError{statusCode: http.StatusInternalServerError, err: errors.New("Can not get data from base")}.ServeHTTP(w, r)
	} else {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
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
curl -i --insecure -X PUT "https://localhost:6060/api/v1/limits?key=qwertyu&name=userName" -d "{\"balance\":1000,\"rate\":100, \"period\":3600}"
curl -i --insecure -X PUT "https://localhost:6060/api/v1/limits?key=qwertyu&name=blabu" -d "balance=-20.3"
*/
func putLimitsHandler(w http.ResponseWriter, r *http.Request) {
	type tempLimitStruct struct {
		Balance float64 `json:"balance"`
		Rate    float64 `json:"rate"`
		MaxRx   uint64  `json:"maxRx"`  // in bytes
		MaxTx   uint64  `json:"maxTx"`  // in bytes
		Period  int32   `json:"period"` // in seconds
	}
	IsWritable, limit, findLimitError := getClientLimit(r)
	if IsWritable {
		var newLimit tempLimitStruct
		tempData := make([]byte, r.ContentLength)
		io.ReadFull(r.Body, tempData)
		if err := json.Unmarshal(tempData, &newLimit); err != nil {
			httpError{statusCode: http.StatusBadRequest, err: errors.New("Can not find param in request body")}.ServeHTTP(w, r)
			return
		}
		if findLimitError != nil {
			limit.LastActivity = time.Now()
		}
		limit.Balance += newLimit.Balance
		if newLimit.Rate != 0 {
			limit.Rate = newLimit.Rate
		}
		if newLimit.MaxRx != 0 {
			limit.MaxReceivedBytes = newLimit.MaxRx
		}
		if newLimit.MaxTx != 0 {
			limit.MaxTransmittedBytes = newLimit.MaxTx
		}
		if newLimit.Period != 0 {
			limit.TimePeriod = time.Duration(newLimit.Period) * time.Second
			if findLimitError != nil {
				limit.LimitExpiration = limit.LastActivity.Add(limit.TimePeriod)
			}
		}
		c2cData.GetBoltDbInstance().UpdateStat(&limit)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		d, _ := json.Marshal(limit)
		w.Write(d)
	} else {
		httpError{statusCode: http.StatusForbidden, err: errors.New("Operation not permitted")}.ServeHTTP(w, r)
	}
}
