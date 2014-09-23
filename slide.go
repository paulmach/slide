package slide

import (
	"errors"
	"math"
	"runtime"
	"time"

	"github.com/paulmach/go.geo"
	geo_reducers "github.com/paulmach/go.geo/reducers"
	slide_reducers "github.com/paulmach/slide/reducers"
)

// Optimization Parameter defaults
const (
	DefaultMinLoops         = 100
	DefaultMaxLoops         = 4000
	DefaultThresholdEpsilon = 0.0005

	DefaultResampleInterval = 5.0 // meters
)

// Slide is the struct that holds all the information to perform a slide.
type Slide struct {
	Geometry   []*geo.Path
	Surfacer   Surfacer
	GeoReducer geo.GeoReducer

	MinLoops   int // will run at least this many refinement steps
	MaxLoops   int // limit on refinement steps
	Goroutines int // concurrency during refinement

	// ThresholdEpsilon is the stop condition used for improvement.
	// See the internal "score" function for more details.
	ThresholdEpsilon float64

	// meters to resample the geometries into before sliding,
	// can impact performance.
	ResampleInterval float64

	// weights for the different components of the cost function.
	GradientScale float64
	DistanceScale float64
	AngleScale    float64
	MomentumScale float64

	// set to the default internal values of gradientContribution, distanceContribution and angleContribution
	// but if you want to get fancy, you can override them.
	GradientContributionFunc func(surfacer Surfacer, point *geo.Point, scale float64) *geo.Point
	DistanceContributionFunc func(path *geo.Path, index int, scale float64) *geo.Point
	AngleContributionFunc    func(path *geo.Path, index int, scale float64) *geo.Point

	// Reduce the correction for paths that are in the valley of the surface.
	// The reduction is based on the original surface value.
	// This option can be helpful when sliding to good data, such as rasterized vector geometry.
	DepthBasedReduction bool

	// NumberIntermediateGeometries is the steps of the refinement processes to save.
	// This is for debugging or animation.
	NumberIntermediateGeometries int

	latLngBound *geo.Bound
}

// Result is the structure containing the results of the sliding process.
// Geometries will be paths in lat/lng (EPSG:4326).
type Result struct {
	CorrectedGeometry    []*geo.Path
	IntermediateGeometry [][]*geo.Path
	LoopsCompleted       int
	LastLoopError        float64
	LastLoopScore        float64
	Runtime              time.Duration
}

// New creates a new Slide structure with the default parameters.
func New(geometry []*geo.Path, surfacer Surfacer) *Slide {
	suggested := surfacer.SuggestedOptions()
	return &Slide{
		Geometry:   geometry,
		Surfacer:   surfacer,
		GeoReducer: slide_reducers.NewTrim(geo_reducers.NewDouglasPeucker(1.0)),

		MinLoops:         DefaultMinLoops,
		MaxLoops:         DefaultMaxLoops,
		ThresholdEpsilon: DefaultThresholdEpsilon,
		Goroutines:       runtime.NumCPU(),

		ResampleInterval: DefaultResampleInterval,

		GradientScale: suggested.GradientScale,
		DistanceScale: suggested.DistanceScale,
		AngleScale:    suggested.AngleScale,
		MomentumScale: suggested.MomentumScale,

		GradientContributionFunc: gradientContribution,
		DistanceContributionFunc: distanceContribution,
		AngleContributionFunc:    angleContribution,

		DepthBasedReduction: suggested.DepthBasedReduction,
	}
}

// Do performs the slide algorithm which includes the following:
// - transform geometries into EPSG:3857 and resample
// - iterate and refine path
// - transform the result back into EPSG:4326
func (s *Slide) Do() (*Result, error) {

	if len(s.Geometry) == 0 {
		return nil, errors.New("slide: please provide at least one path")
	}

	if len(s.Geometry) > 1 {
		return nil, errors.New("slide: currently the sliding of only one path is supported")
	}

	if s.Geometry[0] == nil {
		return nil, errors.New("slide: geometry[0] is nil")
	}

	if s.Geometry[0].Length() < 2 {
		return nil, errors.New("slide: path less than 2 points")
	}

	start := time.Now()

	if s.Goroutines < 1 {
		s.Goroutines = 1
	}

	s.latLngBound = s.Geometry[0].Bound()
	for i := 1; i < len(s.Geometry); i++ {
		s.latLngBound.Union(s.Geometry[i].Bound())
	}

	scaleFactor := geo.MercatorScaleFactor(s.latLngBound.Center().Lat())

	for i := range s.Geometry {
		// The slider works in EPSG:3857
		s.Geometry[i].Transform(geo.Mercator.Project)

		// resamples the path so that there is a data point
		// at least every options.PathResampleInterval meters.
		// This makes sure the path initially satisfies the equidistant constraint.
		distance := s.Geometry[0].Distance()
		count := int(math.Ceil(distance / (s.ResampleInterval * scaleFactor)))
		s.Geometry[i].Resample(count + 3)
	}

	// slide the single path
	result, err := s.refine()
	if err != nil {
		return nil, err
	}

	// convert everything back into the lat/lng space and simplify.
	// TODO: find a better reducer.
	for i, p := range result.CorrectedGeometry {
		p.Transform(geo.Mercator.Inverse)
		if s.GeoReducer != nil {
			result.CorrectedGeometry[i] = s.GeoReducer.GeoReduce(p)
		} else {
			result.CorrectedGeometry[i] = p
		}
	}

	for i := range result.IntermediateGeometry {
		for j, p := range result.IntermediateGeometry[i] {
			p.Transform(geo.Mercator.Inverse)
			if s.GeoReducer != nil {
				result.IntermediateGeometry[i][j] = s.GeoReducer.GeoReduce(p)
			} else {
				result.IntermediateGeometry[i][j] = p
			}
		}
	}

	result.Runtime = time.Since(start)
	return result, nil
}
