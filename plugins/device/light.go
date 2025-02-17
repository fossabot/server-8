package device

import (
	"reflect"

	"go-home.io/x/server/plugins/common"
)

// ILight defines lights plugin interface.
type ILight interface {
	IDevice
	Load() (*LightState, error)
	On() error
	Off() error
	Toggle() error
	Update() (*LightState, error)
	SetBrightness(GradualBrightness) error
	SetScene(common.String) error
	SetColor(common.Color) error
	SetTransitionTime(common.Int) error
}

// LightState returns information about known light.
type LightState struct {
	TransitionTime    int          `json:"transition_time"`
	BrightnessPercent uint8        `json:"brightness"`
	On                bool         `json:"on"`
	Color             common.Color `json:"color"`
	Scenes            []string     `json:"scenes"`
}

// GradualBrightness defines request for gradual brightness increase.
type GradualBrightness struct {
	common.Percent
	TransitionSeconds uint16 `json:"transition_seconds" validate:"isdefault|gt=0,lt=15"`
}

// TypeLight is a syntax sugar around ILight type.
var TypeLight = reflect.TypeOf((*ILight)(nil)).Elem()
