//go:build tinygo && (rp2040 || rp2350)

package tinygo_escmotor

import (
	tinygotypes "github.com/ralvarezdev/tinygo-types"
)

type (
	// Handler is the interface to handle ESC (Electronic Speed Controller) motor operations
	Handler interface {
		GetSpeed() int16
		SetSpeed(speed uint16, isForward bool) tinygotypes.ErrorCode
		Stop() tinygotypes.ErrorCode
		SetSpeedForward(speed uint16) tinygotypes.ErrorCode
		SetSpeedBackward(speed uint16) tinygotypes.ErrorCode
	}
)
