package comfoair

type DeviceInfo struct {
	MajorVersion int
	MinorVersion int
	DeviceName   string
}

type FanStatus struct {
	Preset       string
	Supply       int
	Exhaust      int
	SupplySpeed  int
	ExhaustSpeed int
}

type ValveStatus struct {
	Bypass                 int
	PreHeating             bool
	BypassMotorCurrent     int
	PreheatingMotorCurrent int
}

type TemperatureStatus struct {
	Comfort float32
	Outside float32
	Supply  float32
	Exhaust float32
	Return  float32
}

type OperatingTime struct {
	LowHours    int
	MediumHours int
	HighHours   int
	FilterHours int
}

type Status struct {
	Temperature *TemperatureStatus
	Fan         *FanStatus
	Valve       *ValveStatus
}
