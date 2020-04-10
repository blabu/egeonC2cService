package httpGateway

import (
	log "blabu/c2cService/logWrapper"
	"net/http"
	"text/template"
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
