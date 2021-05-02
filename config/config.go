package config

import (
	"encoding/json"
	"os"
)

type Configuration struct {
	SerialPort string `json:"serial_port"`
	Mqtt       Mqtt   `json:"mqtt"`
}

type Mqtt struct {
	IpAddress string `json:"ip_address"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

func LoadConfiguration(filename string) (*Configuration, error) {
	var file *os.File
	var err error
	if file, err = os.Open(filename); err != nil {
		return nil, err
	}

	defer file.Close()
	decoder := json.NewDecoder(file)
	configuration := &Configuration{}
	if err := decoder.Decode(configuration); err != nil {
		return nil, err
	}

	return configuration, nil
}
