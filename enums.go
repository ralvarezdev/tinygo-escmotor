package tinygo_escmotor

type (
	// Direction is an enum to represent the different motor directions for the vehicle.
	Direction uint8
)

const (
	DirectionNil Direction = iota
	DirectionForward
	DirectionBackward
	DirectionStop
)

// InvertedDirection returns the inverted direction.
func (d Direction) InvertedDirection() Direction {
	switch d {
	case DirectionStop:
		return DirectionStop
	case DirectionForward:
		return DirectionBackward
	case DirectionBackward:
		return DirectionForward
	default:
		return DirectionNil
	}
}
