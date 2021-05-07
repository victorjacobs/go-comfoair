package main

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/victorjacobs/go-comfoair/comfoair"
	"github.com/victorjacobs/go-comfoair/config"
)

const topicPrefix = "comfoair"
const homeAssistantPrefix = "homeassistant"
const retainMessages = false // TODO remove this when done

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

	// TODO do availability
	// TODO set retain on all MQTT

	// Fan control
	homeAssistantFanConfiguration := fmt.Sprintf(`{
		"unique_id": "comfoair_fan",
		"name": "Comfoair",
		"state_topic": "%v/fan/state",
		"command_topic": "%v/fan/cmd",
		"preset_mode_state_topic": "%v/fan/preset/state",
		"preset_mode_command_topic": "%v/fan/preset/cmd",
		"preset_modes": ["off", "low", "mid", "high"]
	}`, topicPrefix, topicPrefix, topicPrefix, topicPrefix)

	if t := mqttClient.Publish(homeAssistantPrefix+"/fan/comfoair/config", 0, retainMessages, homeAssistantFanConfiguration); t.Wait() && t.Error() != nil {
		log.Printf("MQTT publishing failed: %v", err)
		return
	}

	// Since we check connectivity to the controller earlier, just publish that it's turned on
	if t := mqttClient.Publish(topicPrefix+"/fan/state", 0, retainMessages, "ON"); t.Wait() && t.Error() != nil {
		log.Printf("MQTT publishing failed: %v", err)
		return
	}

	if t := mqttClient.Subscribe(fmt.Sprintf("%v/fan/cmd", topicPrefix), 0, func(client mqtt.Client, msg mqtt.Message) {
		command := string(msg.Payload())
		if command == "OFF" {
			comfoairClient.ToggleFan(false)
		} else {
			comfoairClient.ToggleFan(true)
		}
	}); t.Wait() && t.Error() != nil {
		log.Printf("MQTT receive error: %v", t.Error())
	}

	if t := mqttClient.Subscribe(fmt.Sprintf("%v/fan/preset/cmd", topicPrefix), 0, func(client mqtt.Client, msg mqtt.Message) {
		preset := string(msg.Payload())
		comfoairClient.SetFanPreset(preset)
	}); t.Wait() && t.Error() != nil {
		log.Printf("MQTT receive error: %v", t.Error())
	}

	// Publish sensors
	for {
		time.Sleep(5 * time.Second)

		fanStatus, err := comfoairClient.GetFanStatus()

		if err != nil {
			log.Printf("Retrieving fan status failed: %v", err)
			break
		}

		log.Printf("Fans: %+v", fanStatus)

		// Update state
		var stateMessage string
		if fanStatus.Preset == "off" {
			stateMessage = "OFF"
		} else {
			stateMessage = "ON"
		}

		if t := mqttClient.Publish(topicPrefix+"/fan/state", 0, retainMessages, stateMessage); t.Wait() && t.Error() != nil {
			log.Printf("MQTT publishing failed: %v", err)
			return
		}

		// Update preset
		if t := mqttClient.Publish(topicPrefix+"/fan/preset/state", 0, retainMessages, fanStatus.Preset); t.Wait() && t.Error() != nil {
			log.Printf("MQTT publishing failed: %v", err)
			return
		}
	}
}
