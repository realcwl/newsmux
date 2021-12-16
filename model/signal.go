package model

type Signal struct {
	SignalType SignalType `json:"signalType"`
	Payload    string     `json:"payload"`
}
