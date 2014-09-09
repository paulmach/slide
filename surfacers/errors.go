package surfacers

import (
	"errors"
)

var (
	// ErrBoundEmpty is returned when trying to build a surface
	// for an empty latlng boundary.
	ErrBoundEmpty = errors.New("surface area bound is empty")

	// ErrStdDevNegative is returned building a surface for a negative Std Dev.
	ErrStdDevNegative = errors.New("standard deviation negative")
)
