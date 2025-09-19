package tinygo_escmotor

import (
	"runtime"
	"time"

	"machine"

	tinygoerrors "github.com/ralvarezdev/tinygo-errors"
	tinygologger "github.com/ralvarezdev/tinygo-logger"
	tinygoservo "tinygo.org/x/drivers/servo"
)

type (
	// DefaultHandler is the default implementation to handle ESC (Electronic Speed Controller) motor operations.
	DefaultHandler struct {
		afterSetSpeedFunc      func(speed int16)
		isMovementEnabled      func() bool
		isPolarityInverted     bool
		frequency              uint16
		minPulseWidth          uint16
		neutralPulseWidth      uint16
		maxPulseWidth          uint16
		servo                  tinygoservo.Servo
		speed                  int16
		direction              Direction
		maxSpeed               uint16
		microseconds           uint16
		intervalSteps          uint16
		logger                 tinygologger.Logger
		lastUpdate             time.Time
		intervalDelay          time.Duration
		backwardToForwardDelay time.Duration
		forwardToBackwardDelay time.Duration
	}
)

var (
	// setSpeedForwardPrefix is the prefix for the log message when setting speed forward
	setSpeedForwardPrefix = []byte("Set ESC Motor speed forward to:")

	// setSpeedBackwardPrefix is the prefix for the log message when setting speed backward
	setSpeedBackwardPrefix = []byte("Set ESC Motor speed backward to:")

	// stopPrefix is the prefix for the log message when stopping the motor
	stopPrefix = []byte("Stop ESC Motor")

	// graduallySetMicrosecondsPrefix is the prefix for the log message when gradually setting microseconds
	graduallySetMicrosecondsPrefix = []byte("Gradually setting ESC Motor to:")
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
// neutralPulseWidth: Neutral pulse width for the ESC motor
// maxPulseWidth: Maximum pulse width for the ESC motor
// intervalSteps: The number of steps to change the speed of the ESC motor
// isPolarityInverted: Whether the motor polarity is inverted
// maxSpeed: The maximum speed value for the motor
// backwardToForwardDelay: Delay when changing direction from backward to forward
// forwardToBackwardDelay: Delay when changing direction from forward to backward
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
	neutralPulseWidth uint16,
	maxPulseWidth uint16,
	intervalSteps uint16,
	isPolarityInverted bool,
	maxSpeed uint16,
	backwardToForwardDelay time.Duration,
	forwardToBackwardDelay time.Duration,
	logger tinygologger.Logger,
) (*DefaultHandler, tinygoerrors.ErrorCode) {
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

	// Check if the neutral pulse width is within the valid range
	if neutralPulseWidth < minPulseWidth || neutralPulseWidth > maxPulseWidth {
		return nil, ErrorCodeESCMotorInvalidNeutralPulseWidth
	}

	// Initialize the ESC motor with the provided parameters
	handler := &DefaultHandler{
		afterSetSpeedFunc:      afterSetSpeedFunc,
		isMovementEnabled:      isMovementEnabled,
		isPolarityInverted:     isPolarityInverted,
		frequency:              frequency,
		minPulseWidth:          minPulseWidth,
		neutralPulseWidth:      neutralPulseWidth,
		maxPulseWidth:          maxPulseWidth,
		intervalSteps:          intervalSteps,
		intervalDelay:          time.Duration(1000/frequency) * time.Millisecond,
		backwardToForwardDelay: backwardToForwardDelay,
		forwardToBackwardDelay: forwardToBackwardDelay,
		maxSpeed:               maxSpeed,
		servo:                  servo,
		speed:                  0,
		microseconds:           neutralPulseWidth,
		logger:                 logger,
	}

	// Stop the motor initially
	_ = handler.Stop()

	return handler, tinygoerrors.ErrorCodeNil
}

// graduallySetMicroseconds gradually sets the microseconds to the target value
//
// Parameters:
//
// target: The target microseconds value to set
func (h *DefaultHandler) graduallySetMicroseconds(target uint16) {
	// Gradually increment or decrement the microseconds to the target value
	if h.microseconds < target {
		for us := h.microseconds; us < target; us += h.intervalSteps {
			// Log the gradual step
			if h.logger != nil {
				h.logger.AddMessageWithUint16(
					graduallySetMicrosecondsPrefix,
					us,
					true,
					true,
					false,
				)
				h.logger.Debug()
			}

			// Set the microseconds and wait for the interval delay
			h.servo.SetMicroseconds(int16(us))
			time.Sleep(h.intervalDelay)
			runtime.Gosched()
		}
	} else if h.microseconds > target {
		for us := h.microseconds; us > target; us -= h.intervalSteps {
			// Log the gradual step
			if h.logger != nil {
				h.logger.AddMessageWithUint16(
					graduallySetMicrosecondsPrefix,
					us,
					true,
					true,
					false,
				)
				h.logger.Debug()
			}

			// Set the microseconds and wait for the interval delay
			h.servo.SetMicroseconds(int16(us))
			time.Sleep(h.intervalDelay)
			runtime.Gosched()
		}
	}

	// Log the final target step
	if h.logger != nil {
		h.logger.AddMessageWithUint16(
			graduallySetMicrosecondsPrefix,
			target,
			true,
			true,
			false,
		)
		h.logger.Debug()
	}

	// Finally, set the exact microseconds
	h.servo.SetMicroseconds(int16(target))
	h.microseconds = target
	time.Sleep(h.intervalDelay)
	runtime.Gosched()
}

// SetSpeed sets the ESC motor speed.
//
// Parameters:
//
// speed: Speed value between -half of the maximum pulse (full backward) and half of the maximum pulse (full forward).
// direction: Direction of the motor.
//
// Returns:
//
// An error if the speed could not be set, otherwise nil.
func (h *DefaultHandler) SetSpeed(
	speed uint16,
	direction Direction,
) tinygoerrors.ErrorCode {
	// Check if the is polarity inverted
	if h.isPolarityInverted {
		direction = direction.InvertedDirection()
	}

	// Check if the speed is within the valid range
	if speed > h.maxSpeed {
		return ErrorCodeESCMotorSpeedOutOfRange
	}

	// Calculate the microseconds based on the speed and direction
	var microseconds uint16
	switch direction {
	case DirectionStop:
		speed = 0
		microseconds = h.neutralPulseWidth
	case DirectionForward:
		microseconds = h.neutralPulseWidth + speed
		h.speed = int16(speed)
	case DirectionBackward:
		microseconds = h.neutralPulseWidth - speed
		h.speed = -int16(speed)
	default:
		return ErrorCodeESCMotorUnknownDirection
	}

	// Ensure the microseconds is within the valid range
	if microseconds < h.minPulseWidth {
		return ErrorCodeESCMotorSpeedBelowMinPulseWidth
	} else if microseconds > h.maxPulseWidth {
		return ErrorCodeESCMotorSpeedAboveMaxPulseWidth
	}

	// Set the servo microseconds if movement is enabled
	if h.isMovementEnabled != nil && !h.isMovementEnabled() {
		microseconds = h.neutralPulseWidth
	} else if h.microseconds != microseconds {
		// Check if it has to sleep the remaining time to match the interval delay
		if !h.lastUpdate.IsZero() {
			elapsed := time.Since(h.lastUpdate)

			// Sleep the remaining time to match the interval delay
			if elapsed < h.intervalDelay {
				time.Sleep(h.intervalDelay - elapsed)
			}
		}

		// Check if the direction has changed
		if (h.direction != direction) && (h.direction != DirectionStop) {
			// Set to neutral pulse width first
			h.graduallySetMicroseconds(h.neutralPulseWidth)

			// Sleep the appropriate delay based on the direction change
			if direction == DirectionForward {
				time.Sleep(h.backwardToForwardDelay)
			} else if direction == DirectionBackward {
				time.Sleep(h.forwardToBackwardDelay)
			}
		}

		// Continue with the gradual change until reaching the target microseconds
		h.graduallySetMicroseconds(microseconds)

		// Update the current direction
		h.direction = direction

		// Set the last update time
		h.lastUpdate = time.Now()
	}

	// Log the speed change
	if h.logger != nil {
		switch direction {
		case DirectionStop:
			h.logger.AddMessage(
				stopPrefix,
				true,
			)
			h.logger.Debug()
		case DirectionForward:
			h.logger.AddMessageWithUint16(
				setSpeedForwardPrefix,
				speed,
				true,
				true,
				false,
			)
			h.logger.Debug()
		case DirectionBackward:
			h.logger.AddMessageWithUint16(
				setSpeedBackwardPrefix,
				speed,
				true,
				true,
				false,
			)
			h.logger.Debug()
		}
	}

	// Call the after set speed function if provided
	if h.afterSetSpeedFunc != nil {
		h.afterSetSpeedFunc(h.speed)
	}
	return tinygoerrors.ErrorCodeNil
}

// GetSpeed returns the current speed of the ESC motor.
//
// Returns:
//
// The current speed of the ESC motor as an int16 value.
func (h *DefaultHandler) GetSpeed() int16 {
	return h.speed
}

// Stop sets the ESC motor speed to 0 (stop).
//
// Returns:
//
// An error if the speed could not be set to 0, otherwise nil.
func (h *DefaultHandler) Stop() tinygoerrors.ErrorCode {
	return h.SetSpeed(0, DirectionStop)
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
func (h *DefaultHandler) SetSpeedForward(speed uint16) tinygoerrors.ErrorCode {
	return h.SetSpeed(speed, DirectionForward)
}

// SafeSetSpeedForward sets the ESC motor speed forward safely.
//
// Parameters:
//
// speed: Speed value between 0 (stop) and half of the maximum pulse (full forward).
//
// Returns:
//
// An error if the speed could not be set, otherwise nil.
func (h *DefaultHandler) SafeSetSpeedForward(speed uint16) tinygoerrors.ErrorCode {
	if speed > h.maxSpeed {
		speed = h.maxSpeed
	}
	return h.SetSpeed(speed, DirectionForward)
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
func (h *DefaultHandler) SetSpeedBackward(speed uint16) tinygoerrors.ErrorCode {
	return h.SetSpeed(speed, DirectionBackward)
}

// SafeSetSpeedBackward sets the ESC motor speed backward safely.
//
// Parameters:
//
// speed: Speed value between 0 (stop) and half of the maximum pulse (full backward).
//
// Returns:
//
// An error if the speed could not be set, otherwise nil.
func (h *DefaultHandler) SafeSetSpeedBackward(speed uint16) tinygoerrors.ErrorCode {
	if speed > h.maxSpeed {
		speed = h.maxSpeed
	}
	return h.SetSpeed(speed, DirectionBackward)
}
