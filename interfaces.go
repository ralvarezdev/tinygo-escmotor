package tinygo_escmotor

import (
	tinygoerrors "github.com/ralvarezdev/tinygo-errors"
)

type (
	// Handler is the interface to handle ESC (Electronic Speed Controller) motor operations
	Handler interface {
		GetSpeed() float64
		Stop() tinygoerrors.ErrorCode
		SetSpeed(speed float64, direction Direction) tinygoerrors.ErrorCode
		SetSpeedForward(speed float64) tinygoerrors.ErrorCode
		SetSpeedBackward(speed float64) tinygoerrors.ErrorCode
	}
)
