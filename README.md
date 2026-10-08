# tinygo-escmotor

ESC (Electronic Speed Controller) motor handler for [TinyGo](https://tinygo.org/). It drives a hobby-style ESC through a PWM output, translating a speed and a direction into pulse widths, with direction-change delays and optional gradual pulse stepping.

**Note:** This repository is archived and read-only.

## Installation

```bash
go get github.com/ralvarezdev/tinygo-escmotor
```

Depends on `tinygo-errors`, `tinygo-logger` and `tinygo-pwm` (same author). It imports TinyGo's `machine` package, so it must be built for a TinyGo target.

## Usage

```go
type Handler interface {
    GetSpeed() float64
    Stop() tinygoerrors.ErrorCode
    SetSpeed(speed float64, direction Direction) tinygoerrors.ErrorCode
    SetSpeedForward(speed float64) tinygoerrors.ErrorCode
    SetSpeedBackward(speed float64) tinygoerrors.ErrorCode
}

func NewDefaultHandler(
    pwm tinygopwm.PWM,
    pin machine.Pin,
    afterSetSpeedFunc func(speed float64),
    isMovementEnabled func() bool,
    frequency uint16,
    minPulseWidth, neutralPulseWidth, maxPulseWidth uint32,
    isPolarityInverted bool,
    maxForwardSpeed, maxBackwardSpeed float64,
    pulseStep *uint32,
    backwardToForwardDelay, forwardToBackwardDelay time.Duration,
    logger tinygologger.Logger,
) (*DefaultHandler, tinygoerrors.ErrorCode)
```

The constructor sets the PWM period from `frequency`, resolves the PWM channel for the pin, validates the pulse widths (min < neutral < max, all below the period) and max speeds (above 0 and at most 1), then stops the motor.

`Direction` values: `DirectionNil`, `DirectionForward`, `DirectionBackward`, `DirectionStop`; `InvertedDirection()` returns the opposite. Errors are `tinygoerrors.ErrorCode` values starting at 5210 (`ErrorCodeESCMotorStartNumber`); see `errors.go`.

## Project structure

```
types.go        DefaultHandler
interfaces.go   Handler interface
enums.go        Direction
errors.go       Error codes
```

## License

GNU General Public License v3.0. See [LICENSE](LICENSE).
