package main

import wazihub "github.com/j-forster/Wazihub-API"

type Acceleration struct {
	X, Y, Z float32
}

type Force [6]int32

type Avg struct {
	Min, Mean, Max    Acceleration
	Force, Force2     Force
	AValues           int32
	PValues, P2Values int32
	Packets, Dropps   int64
}

type ValuesPerSecond struct {
	Values, Dropps int64
}

var device = &wazihub.Device{
	Id:     wazihub.CurrentDeviceId(),
	Name:   "MEKOB-P",
	Domain: "MEKOB-P",
	Sensors: []*wazihub.Sensor{
		&wazihub.Sensor{
			Id:            "acc0",
			Name:          "Acceleration 1",
			SensingDevice: "AccelerationSensor",
			Unit:          "m/s",
		},
		&wazihub.Sensor{
			Id:            ".acc0",
			Name:          "Condensed Acceleration 1",
			SensingDevice: "AccelerationSensor",
			Unit:          "m/s",
		},
		&wazihub.Sensor{
			Id:            "acc-vps",
			Name:          "Acceleration Values/Sec",
			SensingDevice: "Counter",
			Unit:          "Values/Sec",
		},
		&wazihub.Sensor{
			Id:            "event",
			Name:          "Device Events",
			SensingDevice: "String",
			Unit:          "",
		},
		&wazihub.Sensor{
			Id:            "position",
			Name:          "Position",
			SensingDevice: "Zigpos",
			Unit:          "",
		},
		&wazihub.Sensor{
			Id:            "gps",
			Name:          "GPS Position",
			SensingDevice: "GPS",
			Unit:          "",
		},
	},
	Actuators: []*wazihub.Actuator{
		&wazihub.Actuator{
			Id:   "event",
			Name: "Device Events",
		},
	},
}

type EventError struct {
	Error string
}

type EventStart struct {
	NumDevices int
}
