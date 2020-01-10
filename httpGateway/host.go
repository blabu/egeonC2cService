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

const errorTextHtml = `
<!DOCTYPE html>
<html>
    <head>
        <meta charset="UTF-8">
        <title>ERROR</title>
		<style type="text/css">
		#errM {
			color : red;
		}
		</style>
	</head>
	<body>
		<h1>Sorry, something go wrong</h1>
		<h1 id="errM">
			{{.}}
		</h1>
	</body>
</html>
`

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

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	if tmp, er := template.New("errorMsg").Parse(errorTextHtml); er != nil {
		log.Error(er.Error())
	} else {
		tmp.Execute(w, err.Error())
	}
	return
}

func getStatPage(s *stat.Statistics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type statView struct {
			MaxResponceTime         int64
			TimeUp                  time.Time
			MaxTimeForOneConnection time.Duration
			WorkingTime             time.Duration
			NowConnected            int32
			MaxCuncurentConnection  int32
			AllConnection           int32
			ServerVersion           string
		}
		var templStat *template.Template
		var e error
		log.Info("Try read and parse template file")
		pathToresources, err := cf.GetConfigValue("PathToWeb")
		if err != nil {
			errorHandler(w, r, err)
			return
		}
		if templStat, e = template.ParseFiles(pathToresources + "statistics.html"); e != nil {
			errorHandler(w, r, e)
			return
		}
		view := statView{
			s.MaxResponceTime,
			s.TimeUP,
			s.MaxTimeForOneConnection,
			time.Since(s.TimeUP),
			s.NowConnected,
			s.MaxCuncurentConnection,
			s.AllConnection,
			stat.S_VERSION,
		}
		if e = templStat.Execute(w, view); e != nil {
			errorHandler(w, r, e)
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
