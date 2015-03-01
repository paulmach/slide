package stravaheat

import (
	"github.com/paulmach/go.geo"
	"github.com/paulmach/slide/utils"
	"github.com/paulmach/slide/utils/smoothsurface"
)

// Resmooth takes the data pulled in from the tiles and applies a new smoothing
// to it based on a potentially updated `SmoothingStdDev`.
// Basically it clears and resets the LazySmoothSurface.
func (surfacer *StravaHeatSurface) Resmooth() error {
	if surfacer.smoothSurface == nil {
		return surfacer.smooth()
	}

	surfacer.smoothSurface.SetKernel(surfacer.smoothKernel())
	return nil
}

// smooth sets up the LazySmoothSurface with a kernel.
func (surfacer *StravaHeatSurface) smooth() error {
	surfacer.smoothSurface = smoothsurface.New(surfacer.surface, surfacer.smoothKernel())
	return nil
}

// smoothKernel creates the smoothing kernel that is based on `SmoothingStdDev` (meters).
// See the function utils.Kernel for more information.
func (surfacer *StravaHeatSurface) smoothKernel() []float64 {
	return utils.Kernel(
		surfacer.SmoothingStdDev,
		geo.MercatorScaleFactor(surfacer.lnglatBound.Center().Lat()),
	)
}
