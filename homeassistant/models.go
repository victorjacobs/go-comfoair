package homeassistant

type sensorConfiguration struct {
	UniqueId          string `json:"unique_id"`
	Name              string `json:"name"`
	DeviceClass       string `json:"device_class,omitempty"`
	StateTopic        string `json:"state_topic"`
	UnitOfMeasurement string `json:"unit_of_measurement"`
}

type fanConfiguration struct {
	UniqueId               string   `json:"unique_id"`
	Name                   string   `json:"name"`
	StateTopic             string   `json:"state_topic"`
	CommandTopic           string   `json:"command_topic"`
	PresetModeStateTopic   string   `json:"preset_mode_state_topic"`
	PresetModeCommandTopic string   `json:"preset_mode_command_topic"`
	PresetModes            []string `json:"preset_modes"`
}
