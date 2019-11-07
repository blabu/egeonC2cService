/*
Package configuration - содержит основные средства для чтения конфигурации. После чего все данные конфигурации будут доступны
в key-value store
*/
package configuration

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	log "blabu/c2cService/logWrapper"
)

const (
	comandPrint = "PrintThis"
	comandFile  = "ReadFile"
	runTask     = "Run"
)

//key-value store for config parameters
var configStore map[string]string

func init() {
	configStore = make(map[string]string, 128)
}

//ReadConfigFile - Read configuration file and fill key-value store.
func ReadConfigFile(filename string) error {
	buff := make([]byte, 0, 1024)
	file, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	n, err := readFile(file, &buff)
	log.Tracef("Conf file readed size %d", n)
	if err != nil {
		return err
	}
	parseBuff(buff)
	return nil
}

//GetConfigValue - return value for key string from internal storage
func GetConfigValue(key string) (string, error) {
	log.Trace("Request for configuration value by key: ", key)
	v, ok := configStore[key]
	if !ok {
		log.Info("Not find data for key ", key)
		return "", fmt.Errorf("Not find data for key %s", key)
	}
	return v, nil
}

//GetConfigValueOrDefault - return value for key string from internal storage if not find return default value
func GetConfigValueOrDefault(key string, defaultVal string) string {
	if val, err := GetConfigValue(key); err == nil {
		return val
	}
	return defaultVal
}

// ShowAllConfigStore - показать key-value store
func ShowAllConfigStore(w io.Writer) {
	i := 0
	for k, v := range configStore {
		i++
		str := fmt.Sprintf("%d.Key:%v, Value:%v \n", i, k, v)
		w.Write([]byte(str))
	}
}

func keyAnalysis(key, value string) {
	switch key {
	// Обработка скриптовых команд
	case comandPrint:
		fmt.Println(value)
	case comandFile:
		err := ReadConfigFile(value)
		if err != nil {
			log.Info(err)
		}
	case runTask:
		prepare := strings.Split(value, ";")
		var cmd *exec.Cmd
		if len(prepare) > 1 {
			cmd = exec.Command(prepare[0], prepare[1:]...)
			log.Infof("Try run command %s with arguments %v", prepare[0], prepare[1:])
		} else {
			cmd = exec.Command(prepare[0])
			log.Infof("Try run command %s", prepare[0])
		}
		go cmd.Run()
	default:
		configStore[key] = value
	}
}

func readFile(file *os.File, buff *[]byte) (int, error) {
	//Read all file
	bufer := bufio.NewReader(file)
	temp := make([]byte, 128)
	for {
		i, err := bufer.Read(temp)
		if err == io.EOF {
			log.Trace("End of file")
			break
		} else if err != nil {
			return len(*buff), err
		}
		*buff = append(*buff, temp[:i]...)
	}
	return len(*buff), nil
}

func parseBuff(buff []byte) {
	var isComment = false
	var isValue = false
	var key bytes.Buffer
	var value bytes.Buffer
	for n := 0; n < len(buff)-1; n++ {
		if buff[n] == ' ' || buff[n] == 0 || buff[n] == '\t' {
			continue
		}
		if buff[n] == '\n' || buff[n] == '\r' { // Конец строки
			isValue = false
			isComment = false
			if key.Len() != 0 && value.Len() != 0 {
				keyAnalysis(key.String(), value.String())
				key.Reset()
				value.Reset()
			}
			continue
		}
		if isComment { // Если коментарий пропускаем
			continue
		}
		if buff[n] == '/' && buff[n+1] == '/' { // Начало коментария
			isComment = true
			continue
		}
		if buff[n] == '#' {
			isComment = true
			continue
		}
		if buff[n] == '=' {
			isValue = true
			continue
		}
		if isValue {
			value.WriteByte(buff[n])
		} else {
			key.WriteByte(buff[n])
		}
	}
}
