package bridge

import "github.com/victorjacobs/go-comfoair/comfoair"

var sensorDefinitions = [...]*sensorConfiguration{
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
