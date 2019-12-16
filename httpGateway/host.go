package httpGateway

import (
	log "blabu/c2cService/logWrapper"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

/*
HTTP Gateway is http server.
can send some request to another peer over http protocol
can upload server by wget command
*/

const uploadBinPath = "/bin"
const uploadConfPath = "/bin/conf"

func getFileUpladHandler(filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, er := os.OpenFile(filePath, os.O_RDONLY, 0)
		if er != nil {
			log.Error(er.Error())
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		w.Header().Add("Content-Type", "application/octet-stream")
		io.Copy(w, file)
	}
}

// RunGateway start handle base http url
func RunGateway(address, confPath string) error {
	log.Info("Start http gateway on ", address)
	r := mux.NewRouter()
	r.HandleFunc(uploadBinPath, getFileUpladHandler(os.Args[0]))
	r.HandleFunc(uploadConfPath, getFileUpladHandler(confPath))
	gateway := http.Server{
		Handler:     r,
		Addr:        address,
		ReadTimeout: 60 * time.Second,
	}
	return gateway.ListenAndServe()
}
