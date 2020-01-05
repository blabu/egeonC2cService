package main

import (
	cf "blabu/c2cService/configuration"
	"blabu/c2cService/data/c2cData"
	http "blabu/c2cService/httpGateway"
	"blabu/c2cService/server"
	"blabu/c2cService/stat"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"flag"

	"go.uber.org/atomic"

	log "blabu/c2cService/logWrapper"
	lg "log"
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
		saveDuration, err := cf.GetConfigValue("saveDuration")
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

func startTCPMainServer() net.Listener {
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
	log.Infof("Start listening at port %s", port)
	return listen
}

func startUDPServer(portStr string, timeout time.Duration, st *stat.Statistics) {
	listen, err := NewUDPListener(4096, portStr)
	if err != nil {
		log.Warning(err.Error())
		return
	}
	for {
		Con, err := listen.Accept()
		if err != nil {
			log.Error(err.Error())
			return
		}
		go server.NewBidirectConnector(timeout*time.Second).ManageSession(Con, st)
	}
}

func main() {
	// Подписываемся на оповещение, когда операционка захочет нас прибить
	signal.Notify(sigTerm, os.Interrupt, os.Kill, syscall.SIGQUIT)
	initLogger()
	cf.ShowAllConfigStore(os.Stderr)
	maxConnectionForOneIP := getMaxConnectionValue()
	timeout := getSessionTimeout()
	defer c2cData.InitC2cDB().Close()
	st := stat.CreateStatistics()
	listen := startTCPMainServer()
	isStoped := atomic.NewBool(false)
	go func() {
		<-sigTerm
		isStoped.Store(true)
		log.Info("Operation system kill server")
		listen.Close()
	}()
	if portStr, err := cf.GetConfigValue("ServerUdpPort"); err == nil {
		log.Info("Start UDP server on ", portStr)
		go startUDPServer(portStr, timeout, &st)
	}
	go http.RunGateway(cf.GetConfigValueOrDefault("GateWayAddr", "localhost:8080"), *confPath, &st)
	for !isStoped.Load() {
		Con, err := listen.Accept() // Ждущая функция (Висим ждем соединения)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() { //check type of error is network error
				log.Warningf("Temporary Accept() failure - %s", err)
			} else {
				log.Infof("Can not accept connection, %v", err)
			}
			runtime.Gosched()
			continue
		}
		count := stat.AddIPAddres(strings.Split(Con.RemoteAddr().String(), ":")[0])
		if maxConnectionForOneIP != 0 && count > maxConnectionForOneIP { // Ограничение максимального кол-ва конектов с одного IP адреса
			Con.Close()
			continue
		}
		log.Info("Create new connection")
		go server.NewBidirectConnector(timeout*time.Second).ManageSession(Con, &st)
	}
}
