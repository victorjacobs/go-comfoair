package homeassistant

import (
	"encoding/json"
	"fmt"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/victorjacobs/go-comfoair/config"
)

type Client struct {
	mqtt mqtt.Client
}

func NewClient(mqtt mqtt.Client) *Client {
	return &Client{
		mqtt: mqtt,
	}
}

func (h *Client) RegisterFan() error {
	fanConfiguration, _ := json.Marshal(fanConfiguration{
		UniqueId:               "comfoair_fan",
		Name:                   "Comfoair",
		StateTopic:             fmt.Sprintf("%v/fan/state", config.TopicPrefix),
		CommandTopic:           fmt.Sprintf("%v/fan/cmd", config.TopicPrefix),
		PresetModeStateTopic:   fmt.Sprintf("%v/fan/preset/state", config.TopicPrefix),
		PresetModeCommandTopic: fmt.Sprintf("%v/fan/preset/cmd", config.TopicPrefix),
		PresetModes:            []string{"off", "low", "mid", "high"},
	})

	if t := h.mqtt.Publish(config.HomeAssistantPrefix+"/fan/comfoair/config", 0, true, fanConfiguration); t.Wait() && t.Error() != nil {
		return t.Error()
	}

	return nil
}

func (h *Client) RegisterSensor(name string, class string, unit string) (string, error) {
	uniqueId := strings.Replace(strings.ToLower(name), " ", "_", -1)

	var stateTopic string
	if class == "" {
		stateTopic = fmt.Sprintf("%v/%v", config.TopicPrefix, uniqueId)
	} else {
		stateTopic = fmt.Sprintf("%v/%v/%v", config.TopicPrefix, class, uniqueId)
	}

	sensorConfiguration, _ := json.Marshal(sensorConfiguration{
		UniqueId:          uniqueId,
		Name:              name,
		DeviceClass:       class,
		StateTopic:        stateTopic,
		UnitOfMeasurement: unit,
	})

	configTopic := fmt.Sprintf("%v/sensor/%v/config", config.HomeAssistantPrefix, uniqueId)

	if t := h.mqtt.Publish(configTopic, 0, true, sensorConfiguration); t.Wait() && t.Error() != nil {
		return "", t.Error()
	}

	return stateTopic, nil
}
