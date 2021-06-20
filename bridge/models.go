package bridge

import "github.com/victorjacobs/go-comfoair/comfoair"

type sensorConfiguration struct {
	name       string
	class      string
	unit       string
	get        func(temp *comfoair.Status) interface{}
	stateTopic string
}
