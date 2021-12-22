package model

type Signal struct {
	SignalType    SignalType `json:"signalType"`
	SignalPayload string     `json:"signalPayload"`
}
