package tile_overlay

import (
	"image/color"

	"github.com/paulmach/go.geo"
	"github.com/paulmach/slide"
	"github.com/paulmach/slide/surfacers"
	"github.com/paulmach/slide/utils"
	"github.com/paulmach/slide/utils/smooth_surface"
)

// defaults for newly created ImageOverlaySurface.
// See ImageOverlaySurface for more information about these parameters.
const (
	DefaultMaxSurfaceTileDim  = 5
	DefaultDownloadGoroutines = 4
	DefaultDownloadRetries    = 2
)

const (
	suggestedGradientScale = 0.5
	suggestedDistanceScale = 0.2
	suggestedAngleScale    = 0.1
	suggestedMomentumScale = 0.7
)

// A TileOverlaySurface reprents a builder and data for a Slide Surface
// based on overlay tile image data. For example, the "New & Misaligned TIGER Roads" overlay
// is a tileset for the iD editor where TIGER roads are yellow. This object can build a
// slidable surface using the yellow roads as target data.
type TileOverlaySurface struct {
	Surface       *geo.Surface
	SmoothSurface *smooth_surface.LazySmoothSurface

	// required options
	SourceURLTemplate string // should be of the form http://host.com/{z}/{x}/{y}.png
	targetColor       color.Color

	// The function used to compute the covert the image color to a [0,1] value.
	// image is the color from the image. targetColor is the same as just above and the color targeted.
	// Initially set to the ColorValue function below. See that function for more details.
	ColorValueFunc func(image, targetColor color.Color) float64

	// SmoothingStdDev is used to do the smoothing of the surface.
	// They are in meters and are scaled to match the mercator projection of the final surface
	// to be able to use the same values for any geo-location.
	SmoothingStdDev float64 // the standard deviation of the Gaussian in meters

	// set from defaults, so optional
	MaxSurfaceTileDim  int // represents the max height and width of the surface in tiles, to cap memory usage
	DownloadGoroutines int // how many goroutines to use when downloading remote tiles
	DownloadRetries    int // number of times to retry when fetching a tile, to absorb network errors

	lnglatBound        *geo.Bound
	xTileMin, yTileMin uint64
	xTileMax, yTileMax uint64
	level              uint64
}

// New creates a new TileOverlaySurface with the given options,
// plus the others set to the defaults.
func New(
	lnglatBound *geo.Bound,
	sourceURLTemplate string,
	smoothingStdDev float64,
	targetColor color.Color,
) *TileOverlaySurface {
	return &TileOverlaySurface{
		SourceURLTemplate: sourceURLTemplate,
		SmoothingStdDev:   smoothingStdDev,

		MaxSurfaceTileDim:  DefaultMaxSurfaceTileDim,
		DownloadGoroutines: DefaultDownloadGoroutines,
		DownloadRetries:    DefaultDownloadRetries,

		lnglatBound: lnglatBound.Clone(),
		targetColor: targetColor,

		ColorValueFunc: ColorValue,
	}
}

// ColorValue takes an image color and the targetColor to compute a value within the [0,1] for the surface.
// Currently it does a simple ratio of image/base. However, this does not for for black.
// TODO: improve this!
func ColorValue(image, targetColor color.Color) float64 {
	ri, bi, gi, _ := targetColor.RGBA()
	r, b, g, _ := image.RGBA()
	ratio := float64(r) / float64(ri)

	if ratio*float64(gi) == float64(g) && ratio*float64(bi) == float64(b) {
		return ratio
	}

	return 0
}

// Build goes through the whole process of building the surface:
//  - figures out the proper zoom and tiles to download
//  - downloads those tiles
//  - smooths the surface, per the options
func (surfacer *TileOverlaySurface) Build() error {
	if surfacer.lnglatBound.Empty() {
		return surfacers.ErrBoundEmpty
	}

	if surfacer.SmoothingStdDev < 0.0 {
		return surfacers.ErrStdDevNegative
	}

	err := surfacer.initialize()
	if err != nil {
		return err
	}

	err = surfacer.downloadTiles()
	if err != nil {
		return err
	}

	return surfacer.smooth()
}

// initialize figures out the proper size of the surface and initializes it.
// The next step should be to download the tiles and place them in the surface, see downloadTiles()
func (surfacer *TileOverlaySurface) initialize() error {
	// padding is 5% of average height and width.
	// in lat/lng space, but that shouldn't matter.
	padding := (surfacer.lnglatBound.Width() + surfacer.lnglatBound.Height()) / 2.0 * 0.05
	surfacer.lnglatBound.Pad(padding)

	xTileMin, xTileMax, yTileMin, yTileMax, level := utils.TileRanges(
		surfacer.lnglatBound,
		uint64(surfacer.MaxSurfaceTileDim))

	shift := geo.ScalarMercator.Level - level

	// build latlng and mercator bounds for the tile ranges we just found
	lng, lat := geo.ScalarMercator.Inverse(xTileMin<<shift, yTileMin<<shift)
	sw := geo.NewPoint(lng, lat)

	lng, lat = geo.ScalarMercator.Inverse((xTileMax+1)<<shift, (yTileMax+1)<<shift)
	ne := geo.NewPoint(lng, lat)

	surfacer.lnglatBound = geo.NewBoundFromPoints(sw, ne)
	mercatorBound := geo.NewBoundFromPoints(sw.Transform(geo.Mercator.Project), ne.Transform(geo.Mercator.Project))

	if mercatorBound.Empty() {
		// since surfacer.lnglatBound.Empty() passed in New() this is a weird check.
		// It may happen for non-zero latlng bounds that are two small to represent a full mercator tile.
		return surfacers.ErrBoundEmpty
	}

	surfacer.Surface = geo.NewSurface(mercatorBound, int((xTileMax-xTileMin+1)*256), int((yTileMax-yTileMin+1)*256))

	// these values will be used by the downloader
	surfacer.xTileMin = xTileMin
	surfacer.xTileMax = xTileMax
	surfacer.yTileMin = yTileMin
	surfacer.yTileMax = yTileMax
	surfacer.level = level

	return nil
}

// GradientAt provides a pass through to surfacer.SmoothSurface.GradientAt()
func (surfacer *TileOverlaySurface) GradientAt(point *geo.Point) *geo.Point {
	return surfacer.SmoothSurface.GradientAt(point)
}

// ValueAt provides a pass through to surfacer.Surface.ValueAt()
func (surfacer *TileOverlaySurface) ValueAt(point *geo.Point) float64 {
	return surfacer.Surface.ValueAt(point)
}

// SuggestedOptions returns the defaults slide should use for some parameters.
func (surfacer *TileOverlaySurface) SuggestedOptions() *slide.SuggestedOptions {
	return &slide.SuggestedOptions{
		GradientScale:       suggestedGradientScale,
		DistanceScale:       suggestedDistanceScale,
		AngleScale:          suggestedAngleScale,
		MomentumScale:       suggestedMomentumScale,
		DepthBasedReduction: true,
	}
}
