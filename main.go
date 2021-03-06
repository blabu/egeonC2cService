package main

import (
	"crypto/tls"
	"errors"
	"flag"
	lg "log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	cf "github.com/blabu/egeonC2cService/configuration"
	c2cData "github.com/blabu/egeonC2cService/data/c2cdata"
	log "github.com/blabu/egeonC2cService/logWrapper"
	"github.com/blabu/egeonC2cService/server"
	"go.uber.org/atomic"
)

var confPath = flag.String("conf", "./config.conf", "Set path to config file")

var sigTerm chan os.Signal

func init() {
	flag.Parse()
	log.Infof("Try read configuration file %s\n", *confPath)
	if err := cf.ReadConfig(*confPath); err != nil {
		log.Fatal("Undefined Configuration file. " + err.Error())
	}
	sigTerm = make(chan os.Signal)
}

func initLogger() {
	logFilePath := cf.Config.LogPath
	var minutes = uint32(cf.Config.SaveDuration) * 60
	if minutes == 0 {
		minutes = uint32(60) * 24 // Раз в сутки по умолчанию
	}
	go log.GetLogger().ChangeFile(logFilePath, time.Duration(minutes)*time.Minute)
	log.SetFlags(lg.Ldate | lg.Ltime | lg.Lshortfile)
}

func getTCPListener() net.Listener {
	port := cf.Config.ServerTCPPort
	if len(port) == 0 {
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
	if portTLS := cf.Config.ServerTLSPort; len(portTLS) == 0 {
		return nil, errors.New("Undefine tls port for server")
	} else if certPath := cf.Config.CertificatePath; len(certPath) == 0 {
		return nil, errors.New("Undefine certificate path")
	} else if privateKeyPath := cf.Config.PrivateKeyPath; len(privateKeyPath) == 0 {
		return nil, errors.New("Undefine private key path")
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
	log.Info("Create new connection from ", Con.RemoteAddr().String())
	go server.StartNewSession(Con, timeout*time.Second)
}

func main() {
	// Подписываемся на оповещение, когда операционка захочет нас прибить
	signal.Notify(sigTerm, os.Interrupt, os.Kill, syscall.SIGQUIT)
	initLogger()
	timeout := time.Duration(cf.Config.SessionTimeOut) * time.Second
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
