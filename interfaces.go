package tinygo_escmotor

import (
	tinygoerrors "github.com/ralvarezdev/tinygo-errors"
)

type (
	// Handler is the interface to handle ESC (Electronic Speed Controller) motor operations
	Handler interface {
		GetSpeed() int16
		SetSpeed(speed uint16, isForward bool) tinygoerrors.ErrorCode
		Stop() tinygoerrors.ErrorCode
		SetSpeedForward(speed uint16) tinygoerrors.ErrorCode
		SetSpeedBackward(speed uint16) tinygoerrors.ErrorCode
	}
)
