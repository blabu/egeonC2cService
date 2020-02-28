package logWrapper

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/logger"
)

//LogFileType - обертка над логером с функцией изменеия файла сохранения
type LogFileType struct {
	file       *os.File
	logWrapper *logger.Logger
	mtx        sync.RWMutex
}

var log LogFileType

func init() {
	log.logWrapper = logger.Init("telemetryAPI", true, false, os.Stdout)
}

//GetLogger - вернет дефолтный логгер
func GetLogger() *LogFileType {
	return &log
}

// newFile - Регистрирует новый файл (держим на него ссылку пока не вызовем функцию Close)
func (l *LogFileType) newFile(f *os.File) {
	if l.file != nil {
		l.logWrapper.Infof("Close old file %s", l.file.Name())
		err := l.file.Close()
		if err != nil {
			l.logWrapper.Infof("Error when try close file %s", err.Error())
		}
	} else {
		logger.Warning("Old file is nil")
	}
	l.file = f
}

//closeFile - Закрывает файл уничтожает ссылку
func (l *LogFileType) closeFile() {
	logger.Info("Try close file wrapper")
	if l.file != nil {
		l.logWrapper.Infof("Close old file %s", l.file.Name())
		l.file.Close()
	} else {
		if l.logWrapper == nil {
			logger.Warning("Error! File is nil")
		} else {
			l.logWrapper.Warning("Error! File is nil")
		}
	}
}

// ChangeFile - Запускает периодическое изменение имени файла куда сохраняются логи
func (l *LogFileType) ChangeFile(addrPath string, dT time.Duration) {
	defer l.closeFile()
	for {
		logFilePath := addrPath + "/log " + strings.Split(time.Now().Format("2006-01-02 15_04_05"), " ")[0] + ".txt"
		logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			if l.logWrapper == nil {
				logger.Errorf("Error when try open a file for loging %s, %s", logFilePath, err.Error())
			} else {
				l.logWrapper.Errorf("Error when try open a file for loging %s, %s", logFilePath, err.Error())
			}
			return
		}
		l.newFile(logFile)
		func(l *LogFileType, dT time.Duration) {
			l.mtx.Lock()
			l.logWrapper = logger.Init("telemetryAPI", true, false, l.file)
			l.mtx.Unlock()
			defer l.logWrapper.Close()
			time.Sleep(dT)
		}(l, dT)
	}
}

/*
Реализация стандартных функций логера
*/

func SetFlags(flags int) {
	logger.SetFlags(flags)
}

func Debug(v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.InfoDepth(1, v...)
}

func Debugf(format string, v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.InfoDepth(1, fmt.Sprintf(format, v...))
}

func Trace(v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.InfoDepth(1, v...)
}

func Tracef(format string, v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.InfoDepth(1, fmt.Sprintf(format, v...))
}

// Info uses the default logger and logs with the Info severity.
// Arguments are handled in the manner of fmt.Print.
func Info(v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.InfoDepth(1, v...)
}

// Infof uses the default logger and logs with the Info severity.
// Arguments are handled in the manner of fmt.Printf.
func Infof(format string, v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.InfoDepth(1, fmt.Sprintf(format, v...))
}

// Warning uses the default logger and logs with the Warning severity.
// Arguments are handled in the manner of fmt.Print.
func Warning(v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.WarningDepth(1, v...)
}

// Warningf uses the default logger and logs with the Warning severity.
// Arguments are handled in the manner of fmt.Printf.
func Warningf(format string, v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.WarningDepth(1, fmt.Sprintf(format, v...))
}

// Error uses the default logger and logs with the Error severity.
// Arguments are handled in the manner of fmt.Print.
func Error(v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.ErrorDepth(1, v...)
}

// Errorf uses the default logger and logs with the Error severity.
// Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.ErrorDepth(1, fmt.Sprintf(format, v...))
}

// Fatal uses the default logger, logs with the Fatal severity,
func Fatal(v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.FatalDepth(1, v...)
}

// Fatalf uses the default logger, logs with the Fatal severity,
// and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Printf.
func Fatalf(format string, v ...interface{}) {
	log.mtx.RLock()
	defer log.mtx.RUnlock()
	log.logWrapper.FatalDepth(1, fmt.Sprintf(format, v...))
}
