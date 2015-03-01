package tileoverlay

import (
	"github.com/paulmach/go.geo"
	"github.com/paulmach/slide/utils"
	"github.com/paulmach/slide/utils/smoothsurface"
)

// Resmooth takes the data pulled in from the tiles and applies a new smoothing
// to it based on a potentially updated `SmoothingStdDev`.
// Basically it clears and resets the LazySmoothSurface.
func (surfacer *TileOverlaySurface) Resmooth() error {
	if surfacer.SmoothSurface == nil {
		return surfacer.smooth()
	}

	surfacer.SmoothSurface.SetKernel(surfacer.smoothKernel())
	return nil
}

// smooth sets up the LazySmoothSurface with a kernel.
func (surfacer *TileOverlaySurface) smooth() error {
	surfacer.SmoothSurface = smoothsurface.New(surfacer.Surface, surfacer.smoothKernel())
	return nil
}

// smoothKernel creates the smoothing kernel that is based `SmoothingStdDev` (meters).
// See the function utils.Kernel for more information.
func (surfacer *TileOverlaySurface) smoothKernel() []float64 {
	return utils.Kernel(
		surfacer.SmoothingStdDev,
		geo.MercatorScaleFactor(surfacer.lnglatBound.Center().Lat()),
	)
}
