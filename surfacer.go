package slide

import (
	"github.com/paulmach/go.geo"
)

// A Surfacer defines what a surface needs to do to be used for sliding.
// It should define a surface in the mercator projected space (EPSG:3857).
// Better values should be positive with a maximum of 1 meter, scaled up
// to be consistent with the EPSG:3857 scaling factor for that latitude.
type Surfacer interface {
	// GradientAt and ValueAt should accept points in the EPSG:3857 (mercator) space.
	GradientAt(point *geo.Point) *geo.Point // typically derived from a smoothed surface
	ValueAt(point *geo.Point) float64       // typically the original surface value (pre smooth)

	// SuggestedOptions allows the surfacer to tell slide what the defaults should be.
	SuggestedOptions() *SuggestedOptions
}

// SuggestedOptions is returned by surfacers to allows them to tell
// the slide algorithm what the default options should be.
type SuggestedOptions struct {
	// weights for the different components of the cost function
	GradientScale float64
	DistanceScale float64
	AngleScale    float64
	MomentumScale float64

	// reduce the correction based on surface depth
	DepthBasedReduction bool
}
