package httpGateway

import (
	cf "blabu/c2cService/configuration"
	"blabu/c2cService/data/c2cData"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"blabu/c2cService/stat"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

func errorHandler(w http.ResponseWriter, r *http.Request, statusCode int, err error) {
	const errorTextHTML = `{ "error": "{{.}}" }`
	if tmp, er := template.New("errorMsg").Parse(errorTextHTML); er != nil {
		log.Error(er.Error())
	} else {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		tmp.Execute(w, err.Error())
	}
	return
}

func checkKey(key, path string) (dto.Permission, error) {
	if rootKey, err := cf.GetConfigValue("RootKey"); err == nil {
		if rootKey == key {
			return dto.Permission{
				URL:        path,
				IsWritable: true,
			}, nil
		}
	}
	db := c2cData.GetBoltDbInstance()
	if Perm, ok := db.(c2cData.IPerm); !ok {
		return dto.Permission{}, errors.New("Can not find permission")
	} else {
		p, err := Perm.GetPermission(key)
		if err != nil {
			return dto.Permission{}, err
		}
		for _, v := range p.Perm {
			if v.URL == path {
				return v, nil
			}
		}
		return dto.Permission{}, errors.New("Operation not permitted")
	}
}

func getFileUploadHandler(filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, er := os.OpenFile(filePath, os.O_RDONLY, 0)
		if er != nil {
			log.Error(er.Error())
			errorHandler(w, r, http.StatusNotFound, errors.New("Undefine requested resource"))
			return
		}
		defer file.Close()
		w.Header().Add("Content-Type", "application/octet-stream")
		io.Copy(w, file)
	}
}

func getServerStatus(s *stat.Statistics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query()
		key := v.Get("key")
		if _, err := checkKey(key, internalStatus); err == nil {
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(s.GetJsonStat())
		} else {
			errorHandler(w, r, http.StatusMethodNotAllowed, err)
		}
	}
}

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
		errorHandler(w, r, http.StatusMethodNotAllowed, err)
	}
}

func insertClient(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if perm, err := checkKey(key, client); err == nil && perm.IsWritable {
		clType, _ := strconv.ParseUint(cf.GetConfigValueOrDefault("ClientType", "1"), 10, 16)
		name := r.URL.Query().Get("name")
		pass := r.URL.Query().Get("pass")
		storage := c2cData.GetBoltDbInstance()
		cl, err := storage.GenerateClient(c2cData.ClientType(clType), name, pass)
		if err != nil {
			errorHandler(w, r, http.StatusBadRequest, err)
		}
		if err = storage.SaveClient(cl); err != nil {
			errorHandler(w, r, http.StatusBadRequest, err)
		}
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		data, _ := json.Marshal(cl)
		w.Write(data)
	} else {
		errorHandler(w, r, http.StatusMethodNotAllowed, err)
	}
}

func getClient(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if _, err := checkKey(key, client); err != nil {
		errorHandler(w, r, http.StatusMethodNotAllowed, err)
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
			errorHandler(w, r, http.StatusBadRequest, err)
			return
		}
	} else {
		id, err = strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			errorHandler(w, r, http.StatusBadRequest, err)
			return
		}
	}
	cl, err := storage.GetClient(id)
	if err != nil {
		errorHandler(w, r, http.StatusBadRequest, err)
		return
	}
	log.Infof("client finded %v", *cl)
	data, _ := json.Marshal(cl)
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func limitsHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	perm, err := checkKey(key, limits)
	if err != nil {
		errorHandler(w, r, http.StatusMethodNotAllowed, errors.New("Operation not permitted"))
		return
	}
	storage := c2cData.GetBoltDbInstance()
	var id uint64
	idStr := r.URL.Query().Get("id")
	if len(idStr) == 0 {
		name := r.URL.Query().Get("name")
		if id, err = storage.GetClientID(name); err != nil {
			errorHandler(w, r, http.StatusBadRequest, err)
			return
		}
	}
	limit, err := storage.GetStat(id)
	if err != nil {
		errorHandler(w, r, http.StatusBadRequest, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if res, err := json.Marshal(limit); err != nil {
			errorHandler(w, r, http.StatusInternalServerError, errors.New("Can not get data from base"))
		} else {
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(res)
		}
	case http.MethodPost:
		if perm.IsWritable {
			if err := r.ParseForm(); err != nil {
				errorHandler(w, r, http.StatusBadRequest, errors.New("Can not parse form"))
			}
			balance := r.FormValue("balance")
			rate := r.FormValue("rate")
			maxRx := r.FormValue("maxRx")   // in bytes
			maxTx := r.FormValue("maxTx")   // in bytes
			period := r.FormValue("period") // in nano seconds
			if len(balance) != 0 {
				if b, e := strconv.ParseFloat(balance, 64); e == nil {
					limit.Balance = b
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
					limit.TimePeriod = time.Duration(p)
				}
			}
			storage.UpdateStat(&limit)
		} else {
			errorHandler(w, r, http.StatusForbidden, errors.New("Operation not permitted"))
		}
	}
}

/*
Required parameters of post query is key
in post body required:
token - key
url - path:isWritable or path
*/
func addPerm(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if p, err := checkKey(key, perm); err != nil {
		errorHandler(w, r, http.StatusForbidden, err)
	} else {
		if p.IsWritable {
			err := r.ParseForm()
			if err != nil {
				errorHandler(w, r, http.StatusBadRequest, errors.New("Bad request"))
				return
			}
			newToken := r.FormValue("token")
			if urls, ok := r.Form["url"]; ok && len(newToken) > 5 {
				var allPerm []dto.Permission
				for _, v := range urls {
					pair := strings.Split(v, ":")
					if len(pair) == 1 {
						allPerm = append(allPerm, dto.Permission{URL: pair[0], IsWritable: false})
					} else if len(pair) == 2 {
						flag, _ := strconv.ParseBool(pair[1])
						allPerm = append(allPerm, dto.Permission{URL: pair[0], IsWritable: flag})
					}
				}
				cl := dto.ClientPermission{
					Key:  newToken,
					Perm: allPerm,
				}
				storage, ok := c2cData.GetBoltDbInstance().(c2cData.IPerm)
				if !ok {
					errorHandler(w, r, http.StatusInternalServerError, errors.New("Database not supported permissions"))
				}
				if err = storage.UpdatePermission(cl); err != nil {
					errorHandler(w, r, http.StatusInternalServerError, errors.New("Can not save permission"))
				}
			}
		} else {
			errorHandler(w, r, http.StatusNotImplemented, errors.New("Method not implemented"))
		}
	}
}

func optionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Allow", http.MethodGet)
	w.Header().Add("Allow", http.MethodPost)
	w.Header().Add("Allow", http.MethodOptions)
	w.Header().Add("Access-Control-Allow-Methods", http.MethodGet)
	w.Header().Add("Access-Control-Allow-Methods", http.MethodPost)
	w.WriteHeader(http.StatusOK)
}

/*
HTTP Gateway is http server.
can send some request to another peer over http protocol
can upload server by wget command
RunGateway start handle base http url
*/
func RunGateway(address, confPath string, s *stat.Statistics) error {
	log.Info("Start http gateway on ", address)
	r := mux.NewRouter()
	r.Methods(http.MethodOptions).HandlerFunc(optionsHandler)
	r.Methods(http.MethodGet).Path(uploadBinPath).HandlerFunc(getFileUploadHandler(os.Args[0]))
	r.Methods(http.MethodGet).Path(uploadConfPath).HandlerFunc(getFileUploadHandler(confPath))
	r.Methods(http.MethodGet).Path(internalStatus).HandlerFunc(getServerStatus(s))
	r.Methods(http.MethodGet).Path(clients).HandlerFunc(getClients)
	r.Methods(http.MethodPost).Path(client).HandlerFunc(insertClient)
	r.Methods(http.MethodGet).Path(client).HandlerFunc(getClient)
	r.Path(limits).HandlerFunc(limitsHandler)
	r.Methods(http.MethodPost).Path(perm).HandlerFunc(addPerm)
	gateway := http.Server{
		Handler:     r,
		Addr:        address,
		ReadTimeout: 60 * time.Second,
	}
	return gateway.ListenAndServe()
}
