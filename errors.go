//go:build tinygo && (rp2040 || rp2350)

package tinygo_escmotor

import (
	tinygotypes "github.com/ralvarezdev/tinygo-types"
)

const (
	// ErrorCodeESCMotorStartNumber is the starting number for ESC motor-related error codes.
	ErrorCodeESCMotorStartNumber uint16 = 5210
)

const (
	ErrorCodeESCMotorFailedToConfigurePWM tinygotypes.ErrorCode = tinygotypes.ErrorCode(iota + ErrorCodeESCMotorStartNumber)
	ErrorCodeESCMotorFailedToInitializeServo
	ErrorCodeESCMotorSpeedOutOfRange
	ErrorCodeESCMotorSpeedBelowMinPulseWidth
	ErrorCodeESCMotorSpeedAboveMaxPulseWidth
)
