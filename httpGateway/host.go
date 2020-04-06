package httpGateway

import (
	cf "blabu/c2cService/configuration"
	"blabu/c2cService/data/c2cData"
	"blabu/c2cService/dto"
	log "blabu/c2cService/logWrapper"
	"blabu/c2cService/stat"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

type httpError struct {
	statusCode int
	err        error
}

func (h httpError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const errorTextHTML = `{ "error": "{{.}}" }`
	if tmp, er := template.New("errorMsg").Parse(errorTextHTML); er != nil {
		log.Error(er.Error())
	} else {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(h.statusCode)
		tmp.Execute(w, h.err.Error())
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
		url := strings.TrimPrefix(path, apiLevel)
		for _, v := range p.Perm {
			if v.URL == url {
				return v, nil
			}
			log.Tracef("Url %s not equal requested %s", v.URL, url)
		}
		return dto.Permission{}, errors.New("Operation not permitted")
	}
}

func getFileUploadHandler(filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, er := os.OpenFile(filePath, os.O_RDONLY, 0)
		if er != nil {
			log.Error(er.Error())
			httpError{statusCode: http.StatusNotFound, err: errors.New("Undefine requested resource")}.ServeHTTP(w, r)
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
			httpError{statusCode: http.StatusMethodNotAllowed, err: err}.ServeHTTP(w, r)
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
		httpError{statusCode: http.StatusMethodNotAllowed, err: err}.ServeHTTP(w, r)
	}
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
		w.WriteHeader(http.StatusOK)
		data, _ := json.Marshal(cl)
		w.Write(data)
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
			w.WriteHeader(http.StatusOK)
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

/*
Required parameters of post query is key
in post body required:
token - key
url - path:isWritable or path
Query example
curl -i -cacert ./cert.pem --insecure -H "application/x-www-form-urlencoded" -X POST "https://localhost:6060/api/v1/perm?key=123456" -d "token=qwertyu&url=info:true&url=limits:true&url=perm:true"
*/
func addPerm(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	p, err := checkKey(key, perm)
	if err != nil {
		httpError{statusCode: http.StatusForbidden, err: err}.ServeHTTP(w, r)
		return
	}
	if !p.IsWritable {
		httpError{statusCode: http.StatusNotImplemented, err: errors.New("Method not implemented")}.ServeHTTP(w, r)
		return
	}
	err = r.ParseForm()
	if err != nil {
		httpError{statusCode: http.StatusBadRequest, err: errors.New("Bad request")}.ServeHTTP(w, r)
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
		log.Infof("Create new tocken %s with permision %v", cl.Key, cl.Perm)
		storage, ok := c2cData.GetBoltDbInstance().(c2cData.IPerm)
		if !ok {
			httpError{statusCode: http.StatusInternalServerError, err: errors.New("Database not supported permissions")}.ServeHTTP(w, r)
		}
		if err = storage.UpdatePermission(cl); err != nil {
			httpError{statusCode: http.StatusInternalServerError, err: errors.New("Can not save permission")}.ServeHTTP(w, r)
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		b, _ := json.Marshal(cl)
		w.Write(b)
		return
	}
	httpError{statusCode: http.StatusBadRequest, err: errors.New("Incorrect body in post request")}.ServeHTTP(w, r)
}

func optionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Allow", http.MethodGet)
	w.Header().Add("Allow", http.MethodPost)
	w.Header().Add("Allow", http.MethodOptions)
	w.Header().Add("Access-Control-Allow-Methods", http.MethodGet)
	w.Header().Add("Access-Control-Allow-Methods", http.MethodPost)
	w.Header().Add("Access-Control-Allow-Origin", "*")
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
	r.Methods(http.MethodGet).Path(checkKeyURL).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := checkKey(r.URL.Query().Get("key"), r.URL.Query().Get("path"))
		if err != nil {
			httpError{http.StatusForbidden, err}.ServeHTTP(w, r)
		} else {
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"url":"%s","isWrite":%v}`, p.URL, p.IsWritable)))
		}
	})
	r.Path(limits).HandlerFunc(limitsHandler)
	r.Methods(http.MethodPost).Path(perm).HandlerFunc(addPerm)
	r.MethodNotAllowedHandler = httpError{err: errors.New("Method not allowed. Sorry"), statusCode: http.StatusMethodNotAllowed}
	r.NotFoundHandler = httpError{err: errors.New("Method not exist. Sorry"), statusCode: http.StatusNotExtended}
	if pathToWeb, err := cf.GetConfigValue("PathToWeb"); err == nil {
		log.Info("Path to static resource ", http.Dir(pathToWeb))
		r.Methods(http.MethodGet).PathPrefix("/").Handler(http.StripPrefix("", http.FileServer(http.Dir(pathToWeb))))
	}
	gateway := http.Server{
		Handler:     r,
		Addr:        address,
		ReadTimeout: 60 * time.Second,
	}
	cert, err := cf.GetConfigValue("CertificatePath")
	key, e := cf.GetConfigValue("PrivateKeyPath")
	if err == nil && e == nil {
		log.Info("Start https server")
		return gateway.ListenAndServeTLS(cert, key)
	} else {
		return gateway.ListenAndServe()
	}
}
