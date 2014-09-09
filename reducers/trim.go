package reducers

import (
	"github.com/paulmach/go.geo"
)

// Defaults for the Trim reducer
const (
	TrimDefaultResampleInterval = 2.0  // meters
	TrimDefaultEndpointRadius   = 15.0 // meters
)

// Trim is a "pass-through" reducer that removes all the points within `EndPointRadius` of the endpoints.
// Then it calls the Parent reducer.
// It works, semi-inefficiently, by resampling the whole path and
// then removing points until it finds one outside the radius.
type Trim struct {
	Parent           geo.GeoReducer
	ResampleInterval float64
	EndPointRadius   float64
}

// NewTrim creates a new Trim reducer that will wrap the given reducer.
func NewTrim(reducer geo.GeoReducer) *Trim {
	return &Trim{
		Parent:           reducer,
		ResampleInterval: TrimDefaultResampleInterval,
		EndPointRadius:   TrimDefaultEndpointRadius,
	}
}

// GeoReduce will first trim any points within `EndPointRadius` of the endpoints.
// Then it will run the path through the Parent reducer.
func (t *Trim) GeoReduce(path *geo.Path) *geo.Path {
	parts := int(path.GeoDistance() / t.ResampleInterval)
	path = path.Clone().Resample(parts)

	for path.Length() > 2 && path.GetAt(0).GeoDistanceFrom(path.GetAt(1)) < t.EndPointRadius {
		path.RemoveAt(1)
	}

	for path.Length() > 2 && path.GetAt(path.Length()-1).GeoDistanceFrom(path.GetAt(path.Length()-2)) < t.EndPointRadius {
		path.RemoveAt(path.Length() - 2)
	}

	return t.Parent.GeoReduce(path)
}
