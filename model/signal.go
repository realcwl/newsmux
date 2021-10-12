package model

import (
	"fmt"
	"io"
	"strconv"
)

type Signal struct {
	SignalType SignalType `json:"signalType"`
}

type SignalType string

const (
	SignalTypeSeedState SignalType = "SEED_STATE"
)

var AllSignalType = []SignalType{
	SignalTypeSeedState,
}

func (e SignalType) IsValid() bool {
	switch e {
	case SignalTypeSeedState:
		return true
	}
	return false
}

func (e SignalType) String() string {
	return string(e)
}

func (e *SignalType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = SignalType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid SignalType", str)
	}
	return nil
}

func (e SignalType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
