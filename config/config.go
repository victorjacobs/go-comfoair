package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// TODO remove these when done
const HomeAssistantPrefix = "homeassistant"
const TopicPrefix = "comfoair"

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

func (m *Mqtt) ClientOptions() *mqtt.ClientOptions {
	return mqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://%v:1883", m.IpAddress)).
		SetUsername(m.Username).
		SetPassword(m.Password).
		SetAutoReconnect(true).
		SetConnectionLostHandler(func(client mqtt.Client, err error) {
			log.Printf("MQTT connection lost: %v", err)
		}).
		SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
			log.Printf("MQTT reconnecting")
		})
}
