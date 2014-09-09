package image

import (
	"image"
	"image/color"

	"github.com/paulmach/go.geo"
	"github.com/paulmach/slide"
	"github.com/paulmach/slide/surfacers"
	"github.com/paulmach/slide/utils/smooth_surface"
)

const (
	suggestedGradientScale = 0.5
	suggestedDistanceScale = 0.01
	suggestedAngleScale    = 0.01
	suggestedMomentumScale = 0.0
)

// A ImageSurface reprents a builder and data for a Slide Surface
// based on an image, such as a map scan. It works best with monochromatic images.
type ImageSurface struct {
	Surface       *geo.Surface
	SmoothSurface *smooth_surface.LazySmoothSurface

	// SmoothingStdDev is used to do the smoothing of the surface.
	// They are in meters and are scaled to match the mercator projection of the final surface
	// so the value can be used for any location.
	SmoothingStdDev float64 // the standard deviation of the Gaussian in meters

	lnglatBound *geo.Bound
	image       image.Image
	targetColor color.Color

	// The function used to compute the covert the image color to a [0,1] value.
	// image is the color from the image. lineColor is the same as just above and the color targeted.
	// Initially set to the ColorValue function below. See that function for more details.
	ColorValueFunc func(image, targetColor color.Color) float64
}

// New creates a new ImageSurface with the given options,
// plus the others set to the defaults.
func New(
	lnglatBound *geo.Bound,
	img image.Image,
	targetColor color.Color,
	smoothingStdDev float64,
) *ImageSurface {
	return &ImageSurface{
		SmoothingStdDev: smoothingStdDev,

		image:       img,
		lnglatBound: lnglatBound.Clone(),
		targetColor: targetColor,

		ColorValueFunc: ColorValue,
	}
}

// Build does the converting of the provided image into the surface.
func (surfacer *ImageSurface) Build() error {

	if surfacer.lnglatBound.Empty() {
		return surfacers.ErrBoundEmpty
	}

	if surfacer.SmoothingStdDev < 0.0 {
		return surfacers.ErrStdDevNegative
	}

	// copy the image into the surface
	surfacer.Surface = geo.NewSurface(
		geo.NewBoundFromPoints(
			surfacer.lnglatBound.SouthWest().Clone().Transform(geo.Mercator.Project),
			surfacer.lnglatBound.NorthEast().Clone().Transform(geo.Mercator.Project),
		),
		surfacer.image.Bounds().Max.X-surfacer.image.Bounds().Min.X+1,
		surfacer.image.Bounds().Max.Y-surfacer.image.Bounds().Min.Y+1,
	)

	for i := surfacer.image.Bounds().Min.Y; i <= surfacer.image.Bounds().Max.Y; i++ {
		offset := surfacer.image.Bounds().Max.Y - surfacer.image.Bounds().Min.Y - i
		for j := surfacer.image.Bounds().Min.X; j <= surfacer.image.Bounds().Max.X; j++ {
			// while this does a matrix transpose, I did not get any speedup when trying to mess with this.
			surfacer.Surface.Grid[j][offset] = surfacer.ColorValueFunc(surfacer.image.At(j, i), surfacer.targetColor)
		}
	}

	err := surfacer.smooth()
	if err != nil {
		return err
	}

	return nil
}

// GradientAt provides a pass through to surfacer.SmoothSurface.GradientAt()
func (surfacer *ImageSurface) GradientAt(point *geo.Point) *geo.Point {
	return surfacer.SmoothSurface.GradientAt(point)
}

// ValueAt provides a pass through to surfacer.Surface.ValueAt()
func (surfacer *ImageSurface) ValueAt(point *geo.Point) float64 {
	return surfacer.Surface.ValueAt(point)
}

// SuggestedOptions returns the defaults the surfacer should use for some parameters.
func (surfacer *ImageSurface) SuggestedOptions() *slide.SuggestedOptions {
	return &slide.SuggestedOptions{
		GradientScale: suggestedGradientScale,
		DistanceScale: suggestedDistanceScale,
		AngleScale:    suggestedAngleScale,
		MomentumScale: suggestedMomentumScale,
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
