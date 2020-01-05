package httpGateway

import (
	cf "blabu/c2cService/configuration"
	log "blabu/c2cService/logWrapper"
	"blabu/c2cService/stat"
	"io"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

/*
HTTP Gateway is http server.
can send some request to another peer over http protocol
can upload server by wget command
*/

const uploadBinPath = "/upload/bin"
const uploadConfPath = "/upload/conf"
const showStat = "/stat"

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

func getStatPage(s *stat.Statistics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type statView struct {
			ServerVersion           string
			TimeUp                  time.Time
			NowConnected            int32
			MaxCuncurentConnection  int32
			AllConnection           int32
			MaxTimeForOneConnection time.Duration
			MaxResponceTime         int64
			WorkingTime             time.Duration
		}
		var templStat *template.Template
		var e error
		log.Info("Try read and parse template file")
		pathToresources, err := cf.GetConfigValue("PathToWeb")
		if err != nil {
			pathToresources = "./httpResource/"
		}
		if templStat, e = template.ParseFiles(pathToresources + "statistics.html"); e != nil {
			log.Error(e.Error())
			w.Write([]byte("<h1>Sorry, Can not parse template html file</h1>"))
		}
		view := statView{
			stat.S_VERSION,
			s.TimeUP,
			s.NowConnected,
			s.MaxCuncurentConnection,
			s.AllConnection,
			s.MaxTimeForOneConnection,
			s.MaxResponceTime,
			time.Since(s.TimeUP),
		}
		if e = templStat.Execute(w, view); e != nil {
			log.Info(e.Error())
		}
	}
}

// RunGateway start handle base http url
func RunGateway(address, confPath string, s *stat.Statistics) error {
	log.Info("Start http gateway on ", address)
	r := mux.NewRouter()
	r.HandleFunc(uploadBinPath, getFileUpladHandler(os.Args[0]))
	r.HandleFunc(uploadConfPath, getFileUpladHandler(confPath))
	r.HandleFunc(showStat, getStatPage(s))
	gateway := http.Server{
		Handler:     r,
		Addr:        address,
		ReadTimeout: 60 * time.Second,
	}
	return gateway.ListenAndServe()
}
