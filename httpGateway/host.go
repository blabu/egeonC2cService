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
	"time"

	"github.com/gorilla/mux"
)

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

/*
Required parameters of post query is key
in post body required:
token - key
url - path:isWritable or path
Query example
curl -i -cacert ./cert.pem --insecure -H "application/x-www-form-urlencoded" -X POST "https://localhost:6060/api/v1/perm?key=123456" -d "token=qwertyu&name=someName&url=info:true&url=limits:true&url=perm:true"
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
	name := r.FormValue("name")
	if urls, ok := r.Form["url"]; ok && len(newToken) > 5 && len(name) > 0 {
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
			Name: name,
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

/*
HTTP Gateway is http server.
can send some request to another peer over http protocol
can upload server by wget command
RunGateway start handle base http url
*/
func RunGateway(address, confPath string, s *stat.Statistics) error {
	r := mux.NewRouter()
	r.Methods(http.MethodGet).Path(uploadBinPath).HandlerFunc(getFileUploadHandler(os.Args[0]))
	r.Methods(http.MethodGet).Path(uploadConfPath).HandlerFunc(getFileUploadHandler(confPath))
	r.Methods(http.MethodGet).Path(internalStatus).HandlerFunc(getServerStatus(s))
	r.Methods(http.MethodGet).Path(client + "/{id}").HandlerFunc(getClient)
	r.Methods(http.MethodPost).Path(client).HandlerFunc(insertClient)
	r.Methods(http.MethodGet).Path(client).HandlerFunc(getClients)
	r.Methods(http.MethodGet).Path(checkKeyURL).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := getClientPermission(r.URL.Query().Get("key"))
		if err != nil {
			httpError{http.StatusForbidden, err}.ServeHTTP(w, r)
		} else {
			if d, e := json.Marshal(p); e == nil {
				w.Header().Add("Access-Control-Allow-Origin", "*")
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(d)
			} else {
				httpError{http.StatusInternalServerError, errors.New("")}.ServeHTTP(w, r)
			}
		}
	})
	r.Methods(http.MethodGet).Path(limits).HandlerFunc(getLimits)
	r.Methods(http.MethodPut, http.MethodPost).Path(limits).HandlerFunc(putLimitsHandler)
	r.Methods(http.MethodPost).Path(perm).HandlerFunc(addPerm)
	r.Use(mux.CORSMethodMiddleware(r))
	maxRequestSize, err := strconv.ParseInt(cf.GetConfigValueOrDefault("MaxPacketSize", "128"), 10, 32)
	if err != nil {
		maxRequestSize = 128 * 1024 // 128 KB
	} else {
		maxRequestSize *= 1024 // n KB
	}
	r.Use(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.ContentLength < 0 || r.ContentLength > maxRequestSize {
					httpError{statusCode: http.StatusRequestEntityTooLarge, err: fmt.Errorf("Request more than %d", maxRequestSize)}.ServeHTTP(w, r)
					return
				}
				next.ServeHTTP(w, r)
			})
		})
	if pathToWeb, err := cf.GetConfigValue("PathToWeb"); err == nil {
		var expire time.Duration
		if expireDuration, err := strconv.ParseInt(cf.GetConfigValueOrDefault("ExpireMapCache", "0"), 10, 32); err != nil {
			expire = 0
		} else {
			expire = time.Duration(expireDuration) * time.Second
		}
		r.HandleFunc(mapURL+"/{z}/{x}/{y}", getMapBoxHandler(pathToWeb+"maps",
			"https://api.tiles.mapbox.com/v4/mapbox.streets/%s/%s/%s.png?access_token=%s",
			mapBoxToken, expire)).Methods(http.MethodGet, http.MethodOptions)

		r.Methods(http.MethodGet).PathPrefix("/").Handler(http.StripPrefix("", http.FileServer(http.Dir(pathToWeb+"build/"))))
	}
	r.MethodNotAllowedHandler = httpError{err: errors.New("Method not allowed. Sorry"), statusCode: http.StatusMethodNotAllowed}
	r.NotFoundHandler = httpError{err: errors.New("Method not exist. Sorry"), statusCode: http.StatusNotExtended}
	gateway := http.Server{
		Handler:     r,
		Addr:        address,
		ReadTimeout: 60 * time.Second,
	}
	cert, err := cf.GetConfigValue("CertificatePath")
	key, e := cf.GetConfigValue("PrivateKeyPath")
	if err == nil && e == nil {
		log.Info("Start https gateway on ", address)
		return gateway.ListenAndServeTLS(cert, key)
	}
	log.Info("Start http gateway on ", address)
	return gateway.ListenAndServe()
}
