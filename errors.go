package tinygo_escmotor

import (
	tinygoerrors "github.com/ralvarezdev/tinygo-errors"
)

const (
	// ErrorCodeESCMotorStartNumber is the starting number for ESC motor-related error codes.
	ErrorCodeESCMotorStartNumber uint16 = 5210
)

const (
	ErrorCodeESCMotorFailedToConfigurePWM tinygoerrors.ErrorCode = tinygoerrors.ErrorCode(iota + ErrorCodeESCMotorStartNumber)
	ErrorCodeESCMotorZeroFrequency
	ErrorCodeESCMotorSpeedOutOfRange
	ErrorCodeESCMotorNilHandler
	ErrorCodeESCMotorInvalidNeutralPulseWidth
	ErrorCodeESCMotorInvalidMinPulseWidth
	ErrorCodeESCMotorInvalidMaxPulseWidth
	ErrorCodeESCMotorUnknownDirection
	ErrorCodeESCMotorInvalidMaxForwardSpeed
	ErrorCodeESCMotorInvalidMaxBackwardSpeed
	ErrorCodeESCMotorFailedToGetPWMChannel
)
