package httpGateway

import (
	log "blabu/c2cService/logWrapper"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const mapBoxToken = "pk.eyJ1IjoiaGFuYmxhIiwiYSI6ImNrMzVzYWp3ZjBseDQzbXBlMTRoZm5xemEifQ.E_EZp4QRG4DWGXQr0RpaoA" //https://api.tiles.mapbox.com/v4/mapbox.streets/%s

func getMapBoxHandler(baseDir, apiTemplateURL, accessToken string, expireSinceNow time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		_, e := checkKey(key, mapURL)
		if e != nil {
			httpError{statusCode: http.StatusForbidden, err: errors.New("Operation not permitted")}.ServeHTTP(w, r)
			return
		}
		paths := mux.Vars(r)
		p1, ok1 := paths["z"]
		p2, ok2 := paths["x"]
		filename, ok3 := paths["y"]
		if !ok1 || !ok2 || !ok3 {
			httpError{statusCode: http.StatusNotFound, err: errors.New("tile not found")}.ServeHTTP(w, r)
			return
		}
		pathToFile := fmt.Sprintf("%s/%s/%s/%s", baseDir, p1, p2, filename)
		var data []byte
		var err error
		if expireSinceNow > 0 {
			data, err = readNotExpireFile(pathToFile, expireSinceNow)
		} else {
			data, err = readFile(pathToFile)
		}
		if err == nil {
			log.Debug("Title finded in cache")
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}
		queryString := fmt.Sprintf(apiTemplateURL, p1, p2, filename, accessToken)
		log.Debugf("Query for new title %s", queryString)
		responce, err := http.Get(queryString)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(err.Error()))
			return
		}
		defer responce.Body.Close()
		data, err = ioutil.ReadAll(responce.Body)
		if responce.StatusCode != http.StatusOK || err != nil {
			log.Infof("Status not ok error %s \n", responce.Status)
			w.WriteHeader(responce.StatusCode)
			return
		}
		go writeToFile(data, pathToFile)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}
