package image

import (
	"github.com/paulmach/go.geo"
	"github.com/paulmach/slide/utils"
	"github.com/paulmach/slide/utils/smooth_surface"
)

// Resmooth takes the data pulled in from the tiles and applies a new smoothing
// to it based on a potentially updated `SmoothingStdDev`.
// Basically it clears and resets the LazySmoothSurface.
func (surfacer *ImageSurface) Resmooth() error {
	if surfacer.SmoothSurface == nil {
		return surfacer.smooth()
	}

	surfacer.SmoothSurface.SetKernel(surfacer.smoothKernel())
	return nil
}

// smooth sets ups the LazySmoothSurface with a kernel.
func (surfacer *ImageSurface) smooth() error {
	surfacer.SmoothSurface = smooth_surface.New(surfacer.Surface, surfacer.smoothKernel())
	return nil
}

// smoothKernel creates the smoothing kernel that is based `SmoothingStdDev` (meters).
// See the function utils.Kernel for more information.
func (surfacer *ImageSurface) smoothKernel() []float64 {
	return utils.Kernel(
		surfacer.SmoothingStdDev,
		geo.MercatorScaleFactor(surfacer.lnglatBound.Center().Lat()),
	)
}
