package main

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/victorjacobs/go-comfoair/comfoair"
	"github.com/victorjacobs/go-comfoair/config"
)

const topicPrefix = "test"

func main() {
	var cfg *config.Configuration
	var err error
	if cfg, err = config.LoadConfiguration("comfoair.json"); err != nil {
		log.Fatalf("Error loading configuration: %v", err)
		return
	}

	// Connect serial
	log.Printf("Connecting to %v", cfg.SerialPort)

	comfoairClient, err := comfoair.NewComfoairClient(cfg.SerialPort)
	if err != nil {
		log.Fatal(err)
		return
	}

	deviceInfo, err := comfoairClient.GetDeviceInfo()
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Printf("Connected to %v (v%v.%v)", deviceInfo.DeviceName, deviceInfo.MajorVersion, deviceInfo.MinorVersion)

	// Connect mqtt
	mqttOpts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%v:1883", cfg.Mqtt.IpAddress)).SetUsername(cfg.Mqtt.Username).SetPassword(cfg.Mqtt.Password)
	mqttClient := mqtt.NewClient(mqttOpts)
	if t := mqttClient.Connect(); t.Wait() && t.Error() != nil {
		log.Printf("MQTT connection error: %v", t.Error())
		return
	}

	// Send configuration

	// Publish sensors
	for {
		time.Sleep(5 * time.Second)

		temp, err := comfoairClient.GetTemperatureStatus()

		if err != nil {
			log.Printf("Retrieving temperature failed: %v", err)
			break
		}

		// TODO retain == true
		if t := mqttClient.Publish(topicPrefix+"/outside", 0, false, fmt.Sprintf("%v", temp.Outside)); t.Wait() && t.Error() != nil {
			log.Printf("MQTT publishing failed: %v", err)
			break
		}
	}

	// fanStatus, _ := comfoairClient.GetFanStatus()
	// log.Printf("%+v", fanStatus)

	// comfoairClient.SetFanSpeed(2)

	// time.Sleep(2 * time.Second)

	// fanStatus, _ = comfoairClient.GetFanStatus()
	// log.Printf("%+v", fanStatus)
}
