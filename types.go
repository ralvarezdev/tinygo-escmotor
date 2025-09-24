package tinygo_escmotor

import (
	"time"

	"machine"

	tinygoerrors "github.com/ralvarezdev/tinygo-errors"
	tinygologger "github.com/ralvarezdev/tinygo-logger"
	tinygopwm "github.com/ralvarezdev/tinygo-pwm"
)

type (
	// DefaultHandler is the default implementation to handle ESC (Electronic Speed Controller) motor operations.
	DefaultHandler struct {
		afterSetSpeedFunc      func(speed float64)
		isMovementEnabled      func() bool
		isPolarityInverted     bool
		frequency              uint16
		minPulseWidth          uint32
		neutralPulseWidth      uint32
		maxPulseWidth          uint32
		speed                  float64
		direction              Direction
		maxForwardSpeed        float64
		maxBackwardSpeed       float64
		pulse                  uint32
		pulseStep              *uint32
		logger                 tinygologger.Logger
		lastUpdate             time.Time
		backwardToForwardDelay time.Duration
		forwardToBackwardDelay time.Duration
		lastStopTime           time.Time
		pwm                    tinygopwm.PWM
		period                 uint32
		periodDelay            time.Duration
		channel                uint8
	}
)

const (
	// Float64Precision is the precision for float64 values in log messages
	Float64Precision = 3
)

var (
	// setPeriodPrefix is the prefix for the log message when setting the PWM period
	setPeriodPrefix = []byte("Set ESC Motor PWM period to:")

	// setSpeedForwardPrefix is the prefix for the log message when setting speed forward
	setSpeedForwardPrefix = []byte("Set ESC Motor speed forward to:")

	// setSpeedBackwardPrefix is the prefix for the log message when setting speed backward
	setSpeedBackwardPrefix = []byte("Set ESC Motor speed backward to:")

	// stopPrefix is the prefix for the log message when stopping the motor
	stopPrefix = []byte("Stop ESC Motor")

	// setPulseWidthPrefix is the prefix for the log message when gradually setting the pulse width
	setPulseWidthPrefix = []byte("Set ESC Motor pulse width to:")
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
// isPolarityInverted: Whether the motor polarity is inverted
// maxForwardSpeed: The maximum forward percentage speed value for the motor
// maxBackwardSpeed: The maximum backward percentage speed value for the motor
// pulseStep: Step value for gradually changing the pulse width
// backwardToForwardDelay: Delay when changing direction from backward to forward
// forwardToBackwardDelay: Delay when changing direction from forward to backward
// logger: The logger to log messages
//
// Returns:
//
// An instance of DefaultHandler and an error if any occurred during initialization
func NewDefaultHandler(
	pwm tinygopwm.PWM,
	pin machine.Pin,
	afterSetSpeedFunc func(speed float64),
	isMovementEnabled func() bool,
	frequency uint16,
	minPulseWidth uint32,
	neutralPulseWidth uint32,
	maxPulseWidth uint32,
	isPolarityInverted bool,
	maxForwardSpeed float64,
	maxBackwardSpeed float64,
	pulseStep *uint32,
	backwardToForwardDelay time.Duration,
	forwardToBackwardDelay time.Duration,
	logger tinygologger.Logger,
) (*DefaultHandler, tinygoerrors.ErrorCode) {
	// Check if the frequency is zero
	if frequency == 0 {
		return nil, ErrorCodeESCMotorZeroFrequency
	}

	// Configure the PWM
	period := 1e9 / float64(frequency)
	if err := pwm.Configure(
		machine.PWMConfig{
			Period: uint64(period),
		},
	); err != nil {
		return nil, ErrorCodeESCMotorFailedToConfigurePWM
	}

	// Log the configured period
	if logger != nil {
		logger.AddMessageWithUint32(
			setPeriodPrefix,
			uint32(period),
			true,
			true,
			false,
		)
		logger.Debug()
	}

	// Get the channel from the pin
	channel, err := pwm.Channel(pin)
	if err != nil {
		return nil, ErrorCodeESCMotorFailedToGetPWMChannel
	}

	// Check if the neutral pulse width is within the valid range
	if neutralPulseWidth < minPulseWidth || neutralPulseWidth > maxPulseWidth {
		return nil, ErrorCodeESCMotorInvalidNeutralPulseWidth
	}

	// Check if the min pulse width is valid
	if minPulseWidth == 0 || minPulseWidth >= neutralPulseWidth || minPulseWidth >= uint32(period) {
		return nil, ErrorCodeESCMotorInvalidMinPulseWidth
	}

	// Check if the max pulse width is valid
	if maxPulseWidth == 0 || maxPulseWidth <= neutralPulseWidth || maxPulseWidth >= uint32(period) {
		return nil, ErrorCodeESCMotorInvalidMaxPulseWidth
	}

	// Check if the max forward speed is valid
	if maxForwardSpeed <= 0 || maxForwardSpeed > 1 {
		return nil, ErrorCodeESCMotorInvalidMaxForwardSpeed
	}

	// Check if the max backward speed is valid
	if maxBackwardSpeed <= 0 || maxBackwardSpeed > 1 {
		return nil, ErrorCodeESCMotorInvalidMaxBackwardSpeed
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
		pulseStep:              pulseStep,
		backwardToForwardDelay: backwardToForwardDelay,
		forwardToBackwardDelay: forwardToBackwardDelay,
		maxForwardSpeed:        maxForwardSpeed,
		maxBackwardSpeed:       maxBackwardSpeed,
		speed:                  0,
		pulse:                  neutralPulseWidth,
		logger:                 logger,
		pwm:                    pwm,
		channel:                channel,
		period:                 uint32(period),
		periodDelay:            time.Duration(period),
	}

	// Stop the motor initially
	_ = handler.Stop()

	return handler, tinygoerrors.ErrorCodeNil
}

// graduallySetPulseWidth gradually sets the pulse width to the pulse value
//
// Parameters:
//
// pulse: The pulse pulse width value to set
func (h *DefaultHandler) graduallySetPulseWidth(pulse uint32) {
	// Gradually increment or decrement the pulse to the target value
	if h.pulseStep != nil {
		if h.pulse < pulse {
			for i := h.pulse; i < pulse; i += *h.pulseStep {
				// Log the gradual step
				if h.logger != nil {
					h.logger.AddMessageWithUint32(
						setPulseWidthPrefix,
						i,
						true,
						true,
						false,
					)
					h.logger.Debug()
				}
				tinygopwm.SetDuty(h.pwm, h.channel, i, h.period)
				time.Sleep(h.periodDelay)

				// Update the stop time if it is set to neutral
				if i == h.neutralPulseWidth {
					h.lastStopTime = time.Now()
				}
			}
		} else if h.pulse > pulse {
			for i := h.pulse; i > pulse; i -= *h.pulseStep {
				// Log the gradual step
				if h.logger != nil {
					h.logger.AddMessageWithUint32(
						setPulseWidthPrefix,
						i,
						true,
						true,
						false,
					)
					h.logger.Debug()
				}
				tinygopwm.SetDuty(h.pwm, h.channel, i, h.period)
				time.Sleep(h.periodDelay)

				// Update the stop time if it is set to neutral
				if i == h.neutralPulseWidth {
					h.lastStopTime = time.Now()
				}
			}
		}
	}

	// Log the final pulse
	if h.logger != nil {
		h.logger.AddMessageWithUint32(
			setPulseWidthPrefix,
			pulse,
			true,
			true,
			false,
		)
		h.logger.Debug()
	}

	// Finally, set the exact pulse width
	tinygopwm.SetDuty(h.pwm, h.channel, pulse, h.period)
	h.pulse = pulse

	// Update the stop time if it is set to neutral
	if pulse == h.neutralPulseWidth {
		h.lastStopTime = time.Now()
	}
}

// SetSpeed sets the ESC motor speed.
//
// Parameters:
//
// speed: Speed value between 0 (stop) and maxSpeed (full speed).
// direction: Direction of the motor.
//
// Returns:
//
// An error if the speed could not be set, otherwise nil.
func (h *DefaultHandler) SetSpeed(
	speed float64,
	direction Direction,
) tinygoerrors.ErrorCode {
	// Check if the is polarity inverted
	if h.isPolarityInverted {
		direction = direction.InvertedDirection()
	}

	// Check if the speed is within the valid range
	if speed < 0 || speed > 1 {
		return ErrorCodeESCMotorSpeedOutOfRange
	}

	// Calculate the pulse width based on the speed and direction
	var pulse uint32
	switch direction {
	case DirectionStop:
		speed = 0
		pulse = h.neutralPulseWidth
	case DirectionForward:
		pulse = h.neutralPulseWidth + uint32(float64(h.maxPulseWidth-h.neutralPulseWidth)*speed)
		h.speed = speed
	case DirectionBackward:
		pulse = h.neutralPulseWidth - uint32(float64(h.neutralPulseWidth-h.minPulseWidth)*speed)
		h.speed = -speed
	default:
		return ErrorCodeESCMotorUnknownDirection
	}

	// Set the pulse width if movement is enabled
	if h.isMovementEnabled != nil && !h.isMovementEnabled() {
		pulse = h.neutralPulseWidth
	} else if h.pulse != pulse {
		// Check if it has to sleep the remaining time to match the interval delay
		if !h.lastUpdate.IsZero() {
			elapsed := time.Since(h.lastUpdate)

			// Sleep the remaining time to match the period delay
			if elapsed < h.periodDelay {
				time.Sleep(h.periodDelay - elapsed)
			}
		}

		// Check if the direction has changed
		if (h.direction != direction) && (h.direction != DirectionStop) {
			// Set to neutral pulse width first
			h.graduallySetPulseWidth(h.neutralPulseWidth)
		}

		// Sleep the appropriate delay based on the direction change
		if h.direction != DirectionForward && direction == DirectionForward {
			if !h.lastStopTime.IsZero() {
				time.Sleep(h.backwardToForwardDelay - time.Since(h.lastStopTime))
			} else {
				time.Sleep(h.backwardToForwardDelay)
			}
		} else if h.direction != DirectionBackward && direction == DirectionBackward {
			if !h.lastStopTime.IsZero() {
				time.Sleep(h.forwardToBackwardDelay - time.Since(h.lastStopTime))
			} else {
				time.Sleep(h.forwardToBackwardDelay)
			}
		}

		// Continue with the gradual change until reaching the pulse width
		h.graduallySetPulseWidth(pulse)

		// Update the current direction
		h.direction = direction
		if direction != DirectionStop {
			// Reset the last stop time if not stopping
			h.lastStopTime = time.Time{}
		}

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
			h.logger.AddMessageWithFloat64(
				setSpeedForwardPrefix,
				speed,
				Float64Precision,
				true,
				true,
			)
			h.logger.Debug()
		case DirectionBackward:
			h.logger.AddMessageWithFloat64(
				setSpeedBackwardPrefix,
				speed,
				Float64Precision,
				true,
				true,
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
func (h *DefaultHandler) GetSpeed() float64 {
	if h.direction == DirectionBackward {
		return -h.speed
	} else if h.direction == DirectionForward {
		return h.speed
	}
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
// speed: Speed value between 0 (stop) and maxForwardSpeed (full forward).
//
// Returns:
//
// An error if the speed could not be set, otherwise nil.
func (h *DefaultHandler) SetSpeedForward(speed float64) tinygoerrors.ErrorCode {
	// Check if the speed is within the valid range
	if speed < 0 {
		speed = 0
	}
	if speed > h.maxForwardSpeed {
		speed = h.maxForwardSpeed
	}
	return h.SetSpeed(speed, DirectionForward)
}

// SetSpeedBackward sets the ESC motor speed backward.
//
// Parameters:
//
// speed: Speed value between 0 (stop) and maxBackwardSpeed (full backward).
//
// Returns:
//
// An error if the speed could not be set, otherwise nil.
func (h *DefaultHandler) SetSpeedBackward(speed float64) tinygoerrors.ErrorCode {
	// Check if the speed is within the valid range
	if speed < 0 {
		speed = 0
	}
	if speed > h.maxBackwardSpeed {
		speed = h.maxBackwardSpeed
	}
	return h.SetSpeed(speed, DirectionBackward)
}
