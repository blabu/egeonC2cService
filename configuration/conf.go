/*
Package configuration - содержит основные средства для чтения конфигурации. После чего все данные конфигурации будут доступны
в key-value store
*/
package configuration

import (
	"gopkg.in/yaml.v2"

	"io/ioutil"
	"os"
)

// Config - глобальная структура описывающая конфигурационный файл
type ConfigFile struct {
	ServerTCPPort      string `yaml:"ServerTCPPort"`      // TCP адресс для получения данных
	ServerTLSPort      string `yaml:"ServerTLSPort"`      // TLS адресс для получения данных. Для него также обязательным является абсолютный путь до сертификата и приватного ключа
	CertificatePath    string `yaml:"CertificatePath"`    //Путь к сертификату для TLS сессии
	PrivateKeyPath     string `yaml:"PrivateKeyPath"`     // Путь к приватному ключу для сертиификата для TLS сессии
	MaxQueuePacketSize uint32 `yaml:"MaxQueuePacketSize"` // Максимальная длина очереди сообщений к одному клиенту
	SessionTimeOut     uint32 `yaml:"SessionTimeOut"`     // Таймоут сессии, Если от клиента в течении этого времени в секундах не приходят запросы, Клиент отключается
	MaxPacketSize      uint16 `yaml:"MaxPacketSize"`      // Максимальный размер принимаемого сообщения в Kb за один раз (один пакет)
	C2cStore           string `yaml:"C2cStore"`           // Путь к базе данных клиентов, при отсутствии будет создана новая
	LogPath            string `yaml:"LogPath"`            // Путь куда сохранять логи
	ClientType         uint16 `yaml:"ClientType"`         // Тип клиента должен быть больше 0
	SaveDuration       uint16 `yaml:"SaveDuration"`       // Промежуток времени для сохранения логов
}

//Config - глобальная структура со всеми конфигурациями сервера
var Config ConfigFile

func ReadConfig(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, &Config)
	return err
}
