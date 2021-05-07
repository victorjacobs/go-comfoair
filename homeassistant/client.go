package homeassistant

import (
	"fmt"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/victorjacobs/go-comfoair/config"
)

type HomeAssistantClient struct {
	mqtt mqtt.Client
}

func NewHomeAssistantClient(mqtt mqtt.Client) *HomeAssistantClient {
	return &HomeAssistantClient{
		mqtt: mqtt,
	}
}

func (h *HomeAssistantClient) RegisterFan() error {
	fanConfiguration := fmt.Sprintf(`{
		"unique_id": "comfoair_fan",
		"name": "Comfoair",
		"state_topic": "%v/fan/state",
		"command_topic": "%v/fan/cmd",
		"preset_mode_state_topic": "%v/fan/preset/state",
		"preset_mode_command_topic": "%v/fan/preset/cmd",
		"preset_modes": ["off", "low", "mid", "high"]
	}`, config.TopicPrefix, config.TopicPrefix, config.TopicPrefix, config.TopicPrefix)

	if t := h.mqtt.Publish(config.HomeAssistantPrefix+"/fan/comfoair/config", 0, config.RetainMessages, fanConfiguration); t.Wait() && t.Error() != nil {
		return t.Error()
	}

	return nil
}

func (h *HomeAssistantClient) RegisterSensor(name string, deviceClass string, unitOfMeasurement string) (string, error) {
	uniqueId := strings.Replace(strings.ToLower(name), " ", "_", -1)
	stateTopic := fmt.Sprintf("%v/%v/%v", config.TopicPrefix, deviceClass, uniqueId)

	sensorConfiguration := fmt.Sprintf(`{
		"unique_id": "%v",
		"name": "%v",
		"device_class": "%v",
		"state_topic": "%v",
		"unit_of_measurement": "%v"
	}`, uniqueId, name, deviceClass, stateTopic, unitOfMeasurement)

	configTopic := fmt.Sprintf("%v/sensor/%v/config", config.HomeAssistantPrefix, uniqueId)

	if t := h.mqtt.Publish(configTopic, 0, config.RetainMessages, sensorConfiguration); t.Wait() && t.Error() != nil {
		return "", t.Error()
	}

	return stateTopic, nil
}
