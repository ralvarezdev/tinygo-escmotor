//go:build tinygo && (rp2040 || rp2350)

package tinygo_escmotor

import (
	"time"

	"machine"

	tinygoservo "tinygo.org/x/drivers/servo"
	tinygotypes "github.com/ralvarezdev/tinygo-types"
	tinygologger "github.com/ralvarezdev/tinygo-logger"
)

type (
	// DefaultHandler is the default implementation to handle ESC (Electronic Speed Controller) motor operations.
	DefaultHandler struct {
		afterSetSpeedFunc  func(speed int16)
		isMovementEnabled   func() bool
		isPolarityInverted  bool
		frequency           uint16
		minPulseWidth       uint16
		halfPulseWidth      uint16
		maxPulseWidth       uint16
		rangePulseWidth     uint16
		servo              tinygoservo.Servo
		speed              int16
		maxSpeed           uint16
		microseconds       uint16
		changeInterval    uint16
		changeIntervalDelay time.Duration
		logger             tinygologger.Logger
	}
)

var (
	// setSpeedForwardPrefix is the prefix for the log message when setting speed forward
	setSpeedForwardPrefix = []byte("Set ESC Motor speed forward to:")

	// setSpeedBackwardPrefix is the prefix for the log message when setting speed backward
	setSpeedBackwardPrefix = []byte("Set ESC Motor speed backward to:")
)

// NewDefaultHandler creates a new instance of DefaultHandler
//
// Parameters:
//
// pwm: The PWM interface to control the ESC motor
// pin: The pin connected to the ESC motor
// afterSetSpeedFunc: Function to call after setting the speed
// isMovementEnabled: Function to check if movement is enabled
// frequency: Frequency for the PWM signal
// minPulseWidth: Minimum pulse width for the ESC motor
// maxPulseWidth: Maximum pulse width for the ESC motor
// changeInterval: The interval to change the speed of the ESC motor
// changeIntervalDelay: The interval delay to change the speed of the ESC motor
// isPolarityInverted: Whether the motor polarity is inverted
// maxSpeed: The maximum speed value for the motor
// logger: The logger to log messages
//
// Returns:
//
// An instance of DefaultHandler and an error if any occurred during initialization
func NewDefaultHandler(
	pwm tinygoservo.PWM,
	pin machine.Pin,
	afterSetSpeedFunc func(speed int16),
	isMovementEnabled func() bool,
	frequency uint16,
	minPulseWidth uint16,
	maxPulseWidth uint16,
	changeInterval uint16,
	changeIntervalDelay time.Duration,
	isPolarityInverted bool,
	maxSpeed uint16,
	logger tinygologger.Logger,
) (*DefaultHandler, tinygotypes.ErrorCode) {
	// Configure the PWM
	if err := pwm.Configure(
		machine.PWMConfig{
			Period: uint64(time.Second / time.Duration(frequency)),
		},
	); err != nil {
		return nil, ErrorCodeESCMotorFailedToConfigurePWM
	}

	// Create a new instance of the servo
	servo, err := tinygoservo.New(pwm, pin)
	if err != nil {
		return nil, ErrorCodeESCMotorFailedToInitializeServo
	}

	// Calculate the half pulse and range pulse
	halfPulseWidth := (maxPulseWidth + minPulseWidth) / 2
	rangePulseWidth := maxPulseWidth - minPulseWidth

	// Initialize the ESC motor with the provided parameters
	handler := &DefaultHandler{
		afterSetSpeedFunc:  afterSetSpeedFunc,
		isMovementEnabled:   isMovementEnabled,
		isPolarityInverted:  isPolarityInverted,
		frequency:           frequency,
		minPulseWidth:       minPulseWidth,
		halfPulseWidth:      halfPulseWidth,
		maxPulseWidth:       maxPulseWidth,
		changeInterval:      changeInterval,
		changeIntervalDelay: changeIntervalDelay,
		rangePulseWidth:     rangePulseWidth,
		maxSpeed:           maxSpeed,
		servo:              servo,
		speed:              0,
		microseconds:       halfPulseWidth,
		logger:             logger,
	}

	// Stop the motor initially
	_ = handler.Stop()

	return handler, tinygotypes.ErrorCodeNil
}

// SetSpeed sets the ESC motor speed.
//
// Parameters:
//
// speed: Speed value between -half of the maximum pulse (full backward) and half of the maximum pulse (full forward).
// isForward: Direction of the motor, true for forward, false for backward.
//
// Returns:
//
// An error if the speed could not be set, otherwise nil.
func (e *DefaultHandler) SetSpeed(speed uint16, isForward bool) tinygotypes.ErrorCode {
	// Check if the is polarity inverted
	if e.isPolarityInverted {
		isForward = !isForward
	}

	// Check if the speed is within the valid range
	if speed > e.maxSpeed {
		return ErrorCodeESCMotorSpeedOutOfRange
	}

	// Calculate the microseconds based on the speed and direction
	var microseconds uint16
	if isForward {
		microseconds = e.halfPulseWidth + speed
		e.speed = int16(speed)
	} else {
		microseconds = e.halfPulseWidth - speed
		e.speed = -int16(speed)
	}

	// Ensure the microseconds is within the valid range
	if microseconds < e.minPulseWidth {
		return ErrorCodeESCMotorSpeedBelowMinPulseWidth
	} else if microseconds > e.maxPulseWidth {
		return ErrorCodeESCMotorSpeedAboveMaxPulseWidth
	}

	// Set the servo microseconds if movement is enabled
	if e.isMovementEnabled != nil && !e.isMovementEnabled() {
		microseconds = e.halfPulseWidth
	} else {
		// Gradually change the speed to avoid sudden jumps
		if e.microseconds > microseconds {
			for us := e.microseconds; us > microseconds; us -= e.changeInterval {
				e.servo.SetMicroseconds(int16(us))
				time.Sleep(e.changeIntervalDelay)
			}
		} else if e.microseconds < microseconds {
			for us := e.microseconds; us < microseconds; us += e.changeInterval {
				e.servo.SetMicroseconds(int16(us))
				time.Sleep(e.changeIntervalDelay)
			}
		}

		// Finally, set the exact microseconds
		if e.microseconds != microseconds {
			e.servo.SetMicroseconds(int16(microseconds))

			// Update the current microseconds
			e.microseconds = microseconds
		}
	}

	// Log the speed change
	if e.logger != nil {
		if isForward {
			e.logger.AddMessageWithUint16(setSpeedForwardPrefix, speed, true, true, false)
		} else {
			e.logger.AddMessageWithUint16(setSpeedBackwardPrefix, speed, true, true, false)
		}
		e.logger.Debug()
	}

	// Call the after set speed function if provided
	if e.afterSetSpeedFunc != nil {
		e.afterSetSpeedFunc(e.speed)
	}
	return tinygotypes.ErrorCodeNil
}

// GetSpeed returns the current speed of the ESC motor.
//
// Returns:
//
// The current speed of the ESC motor as an int16 value.
func (e *DefaultHandler) GetSpeed() int16 {
	return e.speed
}

// Stop sets the ESC motor speed to 0 (stop).
//
// Returns:
//
// An error if the speed could not be set to 0, otherwise nil.
func (e *DefaultHandler) Stop() tinygotypes.ErrorCode {
	return e.SetSpeed(0, true)
}

// SetSpeedForward sets the ESC motor speed forward.
//
// Parameters:
//
// speed: Speed value between 0 (stop) and half of the maximum pulse (full forward).
//
// Returns:
//
// An error if the speed could not be set, otherwise nil.
func (e *DefaultHandler) SetSpeedForward(speed uint16) tinygotypes.ErrorCode {
	return e.SetSpeed(speed, true)
}

// SetSpeedBackward sets the ESC motor speed backward.
//
// Parameters:
//
// speed: Speed value between 0 (stop) and half of the maximum pulse (full backward).
//
// Returns:
//
// An error if the speed could not be set, otherwise nil.
func (e *DefaultHandler) SetSpeedBackward(speed uint16) tinygotypes.ErrorCode {
	return e.SetSpeed(speed, false)
}