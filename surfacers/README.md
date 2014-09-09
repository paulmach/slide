Slide Surfacers
===============

Slide is based on the concept to sliding lines into the valleys of a surface.
This surface can be based on any data. To wire it up you'll need to implement the
`Surfacer` interface:

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

Tile Overlay Surfacer
---------------------

This surfacer is provided as a basic example. It downloads [Mapbox TIGER tile layer](https://www.mapbox.com/blog/openstreetmap-tiger/) 
tiles making yellow areas "deep". Vector data can now be "slided" to updated TIGER geometry. [Checkout the demo](http://paulmach.github.io/slide).
