package main

import (
	"crypto/tls"
	"flag"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	cf "github.com/blabu/egeonC2cService/configuration"
	c2cData "github.com/blabu/egeonC2cService/data/c2cdata"
	"github.com/blabu/egeonC2cService/server"

	"go.uber.org/atomic"

	lg "log"

	log "github.com/blabu/egeonC2cService/logWrapper"
)

var confPath = flag.String("conf", "./config.conf", "Set path to config file")

var sigTerm chan os.Signal

func init() {
	flag.Parse()
	log.Infof("Try read configuration file %s\n", *confPath)
	if err := cf.ReadConfigFile(*confPath); err != nil {
		log.Fatal("Undefined Configuration file")
	}
	sigTerm = make(chan os.Signal)
}

func getMaxConnectionValue() uint32 {
	maxConnectStr, err := cf.GetConfigValue("MaxConnectionFromIP")
	if err != nil {
		return 0
	}
	v, err := (strconv.ParseInt(maxConnectStr, 10, 16))
	if err == nil {
		return uint32(v)
	}
	return 0
}

func initLogger() {
	logFilePath, err := cf.GetConfigValue("LogPath")
	if err == nil {
		var minutes uint32
		saveDuration, err := cf.GetConfigValue("SaveDuration")
		if err != nil {
			minutes = 60 * 24 // Раз в сутки по умолчанию
		} else {
			if res, err := strconv.ParseUint(saveDuration, 10, 32); err == nil {
				minutes = uint32(res)
			} else {
				minutes = 60 * 24 // Раз в сутки по умолчанию
			}
		}
		go log.GetLogger().ChangeFile(logFilePath, time.Duration(minutes)*time.Minute)
	}
	log.SetFlags(lg.Ldate | lg.Ltime | lg.Lshortfile)
}

func getSessionTimeout() time.Duration {
	var timeout time.Duration // Таймоут одной сессии
	timeoutStr, err := cf.GetConfigValue("SessionTimeOut")
	if err != nil {
		timeout = 120
	} else {
		t, err := strconv.ParseUint(timeoutStr, 10, 16)
		if err != nil {
			timeout = 120
		} else {
			timeout = time.Duration(t)
		}
	}
	return timeout
}

func getTCPListener() net.Listener {
	port, err := cf.GetConfigValue("ServerTcpPort")
	if err != nil {
		log.Fatal("Undefined ServerTcpPort parameter")
		return nil
	}
	listen, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Can not run listener at port %s %v", port, err)
		return nil
	}
	log.Info("Start TCP server at ", port)
	return listen
}

func getTLSListener() (net.Listener, error) {
	if portTLS, err := cf.GetConfigValue("ServerTlsPort"); err != nil {
		return nil, err
	} else if certPath, err := cf.GetConfigValue("CertificatePath"); err != nil {
		return nil, err
	} else if privateKeyPath, err := cf.GetConfigValue("PrivateKeyPath"); err != nil {
		return nil, err
	} else if certificate, err := tls.LoadX509KeyPair(certPath, privateKeyPath); err != nil {
		return nil, err
	} else if localSrv, err := net.Listen("tcp", portTLS); err != nil {
		return nil, err
	} else {
		conf := &tls.Config{Certificates: []tls.Certificate{certificate}}
		server := tls.NewListener(localSrv, conf)
		log.Info("Start TLS server at ", portTLS)
		return server, nil
	}
}

func startServer(listen net.Listener, timeout time.Duration) {
	Con, err := listen.Accept() // Ждущая функция (Висим ждем соединения)
	if err != nil {
		if nerr, ok := err.(net.Error); ok && nerr.Temporary() { //check type of error is network error
			log.Warningf("Temporary Accept() failure - %s", err)
		} else {
			log.Infof("Can not accept connection, %v", err)
		}
		runtime.Gosched()
		return
	}
	count := st.AddIPAddres(strings.Split(Con.RemoteAddr().String(), ":")[0])
	maxConnectionForOneIP := getMaxConnectionValue()
	if maxConnectionForOneIP != 0 && count > maxConnectionForOneIP { // Ограничение максимального кол-ва конектов с одного IP адреса
		Con.Close()
		return
	}
	log.Info("Create new connection from ", Con.RemoteAddr().String())
	go server.StartNewSession(Con, timeout*time.Second)
}

func main() {
	// Подписываемся на оповещение, когда операционка захочет нас прибить
	signal.Notify(sigTerm, os.Interrupt, os.Kill, syscall.SIGQUIT)
	initLogger()
	cf.ShowAllConfigStore(os.Stderr)
	timeout := getSessionTimeout()
	defer c2cData.InitC2cDB().Close()
	isStoped := atomic.NewBool(false)
	tlsListener, err := getTLSListener()
	if err != nil {
		log.Error(err.Error())
	} else {
		go func() {
			for !isStoped.Load() {
				startServer(tlsListener, timeout)
			}
			log.Info("Finish tls service")
		}()
	}
	tcpListener := getTCPListener()
	go func() {
		for !isStoped.Load() {
			startServer(tcpListener, timeout)
		}
		log.Info("Finish tcp service")
	}()
	<-sigTerm
	isStoped.Store(true)
	log.Info("Operation system kill server")
	tcpListener.Close()
	if tlsListener != nil {
		log.Info("Try close tls connection")
		tlsListener.Close()
	}
}
