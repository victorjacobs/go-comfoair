package main

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/victorjacobs/go-comfoair/comfoair"
	"github.com/victorjacobs/go-comfoair/config"
	"github.com/victorjacobs/go-comfoair/homeassistant"
)

// TODO maybe move this
type sensorConfiguration struct {
	name              string
	sensorClass       string
	unitOfMeasurement string
	getter            func() string
}

func main() {
	var cfg *config.Configuration
	var err error
	if cfg, err = config.LoadConfiguration("comfoair.json"); err != nil {
		log.Fatalf("Error loading configuration: %v", err)
		return
	}

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

	homeAssistant := homeassistant.NewHomeAssistantClient(mqttClient)

	// TODO do availability
	// TODO set retain on all MQTT

	// Fan control
	homeAssistant.RegisterFan()

	// TODO maybe move to homeassistant client?
	if t := mqttClient.Subscribe(fmt.Sprintf("%v/fan/cmd", config.TopicPrefix), 0, func(client mqtt.Client, msg mqtt.Message) {
		command := string(msg.Payload())
		if command == "OFF" {
			comfoairClient.ToggleFan(false)
		} else {
			comfoairClient.ToggleFan(true)
		}
	}); t.Wait() && t.Error() != nil {
		log.Printf("MQTT receive error: %v", t.Error())
	}

	if t := mqttClient.Subscribe(fmt.Sprintf("%v/fan/preset/cmd", config.TopicPrefix), 0, func(client mqtt.Client, msg mqtt.Message) {
		preset := string(msg.Payload())

		if err = comfoairClient.SetFanPreset(preset); err != nil {
			log.Printf("Error setting fan speed: %v", err)
		}
	}); t.Wait() && t.Error() != nil {
		log.Printf("MQTT receive error: %v", t.Error())
	}

	// Poll fan speed
	go loopSafely(func() {
		time.Sleep(1 * time.Second)

		fanStatus, err := comfoairClient.GetFanStatus()

		if err != nil {
			log.Printf("Retrieving fan status failed: %v", err)
			return
		}

		// Update state
		var stateMessage string
		if fanStatus.Preset == "off" {
			stateMessage = "OFF"
		} else {
			stateMessage = "ON"
		}

		if t := mqttClient.Publish(config.TopicPrefix+"/fan/state", 0, config.RetainMessages, stateMessage); t.Wait() && t.Error() != nil {
			log.Printf("MQTT publishing failed: %v", t.Error())
			return
		}

		// Update preset
		if t := mqttClient.Publish(config.TopicPrefix+"/fan/preset/state", 0, config.RetainMessages, fanStatus.Preset); t.Wait() && t.Error() != nil {
			log.Printf("MQTT publishing failed: %v", t.Error())
			return
		}
	})

	sensors := [...]sensorConfiguration{
		{
			name:              "Comfoair Outside Temperature",
			sensorClass:       "temperature",
			unitOfMeasurement: "°C",
			getter:            func() string { return "test" },
		},
	}

	for _, sensorConfig := range sensors {
		// TODO errors
		stateTopic, _ := homeAssistant.RegisterSensor(sensorConfig.name, sensorConfig.sensorClass, sensorConfig.unitOfMeasurement)

		// TODO how to bind stateTopic
		go loopSafely(func() {
			time.Sleep(10 * time.Second)

			log.Printf("Publishing %v to %v", sensorConfig.getter(), stateTopic)
		})
	}

	select {}
}

func loopSafely(f func()) {
	defer func() {
		if v := recover(); v != nil {
			log.Printf("Panic: %v, restarting", v)
			go loopSafely(f)
		}
	}()

	for {
		f()
	}
}
