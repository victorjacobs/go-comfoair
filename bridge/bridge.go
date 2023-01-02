package bridge

import (
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/victorjacobs/go-comfoair/comfoair"
	"github.com/victorjacobs/go-comfoair/config"
	"github.com/victorjacobs/go-comfoair/homeassistant"
)

type Bridge struct {
	cfg            *config.Configuration
	comfoairClient *comfoair.Client
	lastPreset     string
}

func New(cfg *config.Configuration) (*Bridge, error) {
	log.Printf("Connecting to %v", cfg.SerialPort)

	comfoairClient, err := comfoair.NewClient(cfg.SerialPort)
	if err != nil {
		return nil, err
	}

	deviceInfo, err := comfoairClient.GetDeviceInfo()
	if err != nil {
		return nil, err
	}
	log.Printf("Connected to %v (v%v.%v)", deviceInfo.DeviceName, deviceInfo.MajorVersion, deviceInfo.MinorVersion)

	operatingTime, err := comfoairClient.GetOperatingTime()
	if err != nil {
		return nil, err
	}
	log.Printf("Operatingtime: %+v", operatingTime)

	return &Bridge{
		cfg:            cfg,
		comfoairClient: comfoairClient,
		lastPreset:     "",
	}, nil
}

func (b *Bridge) RegisterFan(mqttClient mqtt.Client) error {
	homeAssistantClient := homeassistant.NewClient(mqttClient)

	return homeAssistantClient.RegisterFan()
}

func (b *Bridge) SubscribeToFanCommands(mqttClient mqtt.Client) {
	if t := mqttClient.Subscribe(fmt.Sprintf("%v/fan/cmd", config.TopicPrefix), 0, func(client mqtt.Client, msg mqtt.Message) {
		command := string(msg.Payload())

		if command == "OFF" {
			b.comfoairClient.ToggleFan(false)
		} else {
			b.comfoairClient.ToggleFan(true)
		}
	}); t.Wait() && t.Error() != nil {
		log.Printf("MQTT receive error: %v", t.Error())
	}

	if t := mqttClient.Subscribe(fmt.Sprintf("%v/fan/preset/cmd", config.TopicPrefix), 0, func(client mqtt.Client, msg mqtt.Message) {
		preset := string(msg.Payload())

		if err := b.comfoairClient.SetFanPreset(preset); err != nil {
			log.Printf("Error setting fan speed: %v", err)
		}
	}); t.Wait() && t.Error() != nil {
		log.Printf("MQTT receive error: %v", t.Error())
	}
}

func (b *Bridge) RegisterSensors(mqttClient mqtt.Client) error {
	homeAssistantClient := homeassistant.NewClient(mqttClient)

	for _, sensorConfig := range sensorDefinitions {
		if stateTopic, err := homeAssistantClient.RegisterSensor(sensorConfig.name, sensorConfig.class, sensorConfig.unit); err != nil {
			return err
		} else {
			log.Printf("Registered sensor %v", sensorConfig.name)
			sensorConfig.stateTopic = stateTopic
		}
	}

	return nil
}

func (b *Bridge) PollSensors(mqttClient mqtt.Client) {
	status, err := b.comfoairClient.GetStatus()
	if err != nil {
		log.Panicf("Failed to get status: %v", err)
	}

	for _, sensorConfig := range sensorDefinitions {
		value := fmt.Sprintf("%v", sensorConfig.get(status))

		if t := mqttClient.Publish(sensorConfig.stateTopic, 0, true, value); t.Wait() && t.Error() != nil {
			log.Printf("MQTT publishing failed: %v", t.Error())
			continue
		}
	}
}

func (b *Bridge) PollFanState(mqttClient mqtt.Client) {
	fanStatus, err := b.comfoairClient.GetFanStatus()

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

	if b.lastPreset == "" || b.lastPreset != fanStatus.Preset {
		if t := mqttClient.Publish(config.TopicPrefix+"/fan/state", 0, true, stateMessage); t.Wait() && t.Error() != nil {
			log.Printf("MQTT publishing failed: %v", t.Error())
			return
		}

		if t := mqttClient.Publish(config.TopicPrefix+"/fan/preset/state", 0, true, fanStatus.Preset); t.Wait() && t.Error() != nil {
			log.Printf("MQTT publishing failed: %v", t.Error())
			return
		}

		b.lastPreset = fanStatus.Preset
	}
}

func (b *Bridge) GetOperatingTime() (*comfoair.OperatingTime, error) {
	return b.comfoairClient.GetOperatingTime()
}
