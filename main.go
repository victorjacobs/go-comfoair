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
	name       string
	class      string
	unit       string
	get        func(temp *comfoair.Status) interface{}
	stateTopic string
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

	operatingTime, err := comfoairClient.GetOperatingTime()
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Printf("Operatingtime: %+v", operatingTime)

	// Connect mqtt
	mqttOpts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%v:1883", cfg.Mqtt.IpAddress)).SetUsername(cfg.Mqtt.Username).SetPassword(cfg.Mqtt.Password)
	mqttClient := mqtt.NewClient(mqttOpts)
	if t := mqttClient.Connect(); t.Wait() && t.Error() != nil {
		log.Printf("MQTT connection error: %v", t.Error())
		return
	}

	homeAssistantClient := homeassistant.NewHomeAssistantClient(mqttClient)

	// TODO do availability

	// Fan control
	homeAssistantClient.RegisterFan()

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

	var lastPreset string

	// Poll fan speed
	go loopSafely(func() {
		fanStatus, err := comfoairClient.GetFanStatus()

		if err != nil {
			log.Panicf("Retrieving fan status failed: %v", err)
		}

		// Update state
		var stateMessage string
		if fanStatus.Preset == "off" {
			stateMessage = "OFF"
		} else {
			stateMessage = "ON"
		}

		if lastPreset == "" || lastPreset != fanStatus.Preset {
			if t := mqttClient.Publish(config.TopicPrefix+"/fan/state", 0, true, stateMessage); t.Wait() && t.Error() != nil {
				log.Printf("MQTT publishing failed: %v", t.Error())
				return
			}

			if t := mqttClient.Publish(config.TopicPrefix+"/fan/preset/state", 0, true, fanStatus.Preset); t.Wait() && t.Error() != nil {
				log.Printf("MQTT publishing failed: %v", t.Error())
				return
			}

			lastPreset = fanStatus.Preset
		}

		time.Sleep(1 * time.Second)
	})

	// Sensors
	sensors := [...]*sensorConfiguration{
		{
			name:  "Comfoair Outside Temperature",
			class: "temperature",
			unit:  "°C",
			get:   func(status *comfoair.Status) interface{} { return status.Temperature.Outside },
		},
		{
			name:  "Comfoair Exhaust Temperature",
			class: "temperature",
			unit:  "°C",
			get:   func(status *comfoair.Status) interface{} { return status.Temperature.Exhaust },
		},
		{
			name:  "Comfoair Return Temperature",
			class: "temperature",
			unit:  "°C",
			get:   func(status *comfoair.Status) interface{} { return status.Temperature.Return },
		},
		{
			name:  "Comfoair Supply Temperature",
			class: "temperature",
			unit:  "°C",
			get:   func(status *comfoair.Status) interface{} { return status.Temperature.Supply },
		},
		{
			name:  "Comfoair Comfort Temperature",
			class: "temperature",
			unit:  "°C",
			get:   func(status *comfoair.Status) interface{} { return status.Temperature.Comfort },
		},
		{
			name: "Comfoair Supply Fan Speed",
			unit: "rpm",
			get:  func(status *comfoair.Status) interface{} { return status.Fan.SupplySpeed },
		},
		{
			name: "Comfoair Exhaust Fan Speed",
			unit: "rpm",
			get:  func(status *comfoair.Status) interface{} { return status.Fan.ExhaustSpeed },
		},
		{
			name: "Comfoair Bypass",
			unit: "%",
			get:  func(status *comfoair.Status) interface{} { return status.Valve.Bypass },
		},
	}

	// Register sensors
	for _, sensorConfig := range sensors {
		if stateTopic, err := homeAssistantClient.RegisterSensor(sensorConfig.name, sensorConfig.class, sensorConfig.unit); err != nil {
			log.Fatalf("Failed to register sensor: %v", err)
		} else {
			log.Printf("Registered sensor %v", sensorConfig.name)
			sensorConfig.stateTopic = stateTopic
		}
	}

	// Poll sensor values
	go loopSafely(func() {
		status, err := comfoairClient.GetStatus()
		if err != nil {
			log.Panicf("Failed to get status: %v", err)
		}

		for _, sensorConfig := range sensors {
			value := fmt.Sprintf("%v", sensorConfig.get(status))

			if t := mqttClient.Publish(sensorConfig.stateTopic, 0, true, value); t.Wait() && t.Error() != nil {
				log.Printf("MQTT publishing failed: %v", t.Error())
				continue
			}
		}

		time.Sleep(time.Minute)
	})

	select {}
}

func loopSafely(f func()) {
	defer func() {
		if v := recover(); v != nil {
			log.Printf("Panic: %v, restarting", v)
			time.Sleep(time.Second)
			go loopSafely(f)
		}
	}()

	for {
		f()
	}
}
