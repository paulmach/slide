package smooth_surface

import (
	"math"

	"github.com/paulmach/go.geo"
)

// LazySmoothSurface provides ValueAt and GradientAt function based on
// a surface vertically and horizontally smoothed using the given kernel.
// The values are smoothed on request and cached.
// Note that is takes 3x the memory, but using arrays vs maps with locking was 2x faster.
// Cache hit rate for a 100 iteration sample was 50% for the grid value and 99+% for the
// Intermediate vertical smoothing values.
type LazySmoothSurface struct {
	Surface      *geo.Surface
	kernel       []float64
	gridCache    []float64
	midGridCache []float64
}

// New creates a new smooth service using the kernel.
func New(surface *geo.Surface, kernel []float64) *LazySmoothSurface {
	if len(kernel)%2 == 0 {
		panic("length of kernel must be odd")
	}

	gridCache := make([]float64, surface.Width*surface.Height)
	midGridCache := make([]float64, surface.Width*surface.Height)

	// using NaN for "not found" was the same performance as math.MaxFloat64
	nan := math.NaN()
	for i := range gridCache {
		gridCache[i] = nan
		midGridCache[i] = nan
	}

	return &LazySmoothSurface{
		Surface:      surface,
		kernel:       kernel,
		gridCache:    gridCache,
		midGridCache: midGridCache,
	}
}

// SetKernel allows the updating of the kernel used. The cache is cleared.
func (s *LazySmoothSurface) SetKernel(kernel []float64) {
	s.kernel = kernel

	nan := math.NaN()
	for i := range s.gridCache {
		s.gridCache[i] = nan
		s.midGridCache[i] = nan
	}
}

// GradientAt is the same as for a normal surface but with the smoothing kernel applied.
func (s *LazySmoothSurface) GradientAt(point *geo.Point) *geo.Point {
	if !s.Surface.Bound().Contains(point) {
		return geo.NewPoint(0, 0)
	}

	xi, yi, deltaX, deltaY := s.gridCoordinate(point)

	xi1 := xi + 1
	if limit := s.Surface.Width - 1; xi1 > limit {
		xi = limit - 1
		xi1 = limit
		deltaX = 1.0
	}

	yi1 := yi + 1
	if limit := s.Surface.Height - 1; yi1 > limit {
		yi = limit - 1
		yi1 = limit
		deltaY = 1.0
	}

	u1 := s.SmoothedGrid(xi, yi)*(1-deltaX) + s.SmoothedGrid(xi1, yi)*deltaX
	u2 := s.SmoothedGrid(xi, yi1)*(1-deltaX) + s.SmoothedGrid(xi1, yi1)*deltaX

	w1 := (1 - deltaY) * (s.SmoothedGrid(xi1, yi) - s.SmoothedGrid(xi, yi))
	w2 := deltaY * (s.SmoothedGrid(xi1, yi1) - s.SmoothedGrid(xi, yi1))

	return geo.NewPoint((w1+w2)/s.gridBoxWidth(), (u2-u1)/s.gridBoxHeight())

}

// ValueAt is the same as for a normal surface but with the smoothing kernel applied.
func (s *LazySmoothSurface) ValueAt(point *geo.Point) float64 {
	if !s.Surface.Bound().Contains(point) {
		return 0
	}

	// find height and width
	xi, yi, w, h := s.gridCoordinate(point)

	xi1 := xi + 1
	if limit := s.Surface.Width - 1; xi1 > limit {
		xi1 = limit
	}

	yi1 := yi + 1
	if limit := s.Surface.Height - 1; yi1 > limit {
		yi1 = limit
	}

	w1 := s.SmoothedGrid(xi, yi)*(1-w) + s.SmoothedGrid(xi1, yi)*w
	w2 := s.SmoothedGrid(xi, yi1)*(1-w) + s.SmoothedGrid(xi1, yi1)*w

	return w1*(1-h) + w2*h
}

// SmoothedGrid provides the same stuff as surface.Grid[x][y] but smoothed
// vertically and horizontally by the given kernel. Computed values are cached.
func (s *LazySmoothSurface) SmoothedGrid(x, y int) float64 {
	key := y*s.Surface.Width + x
	if v := s.gridCache[key]; !math.IsNaN(v) {
		return v
	}

	size := (len(s.kernel) - 1) / 2

	// compute the vertical value.
	sum := 0.0
	for j := x - size; j <= x+size; j++ {
		k := j
		if j < 0 {
			k = 0
		}

		if j >= s.Surface.Width {
			k = s.Surface.Width - 1
		}

		sum += s.kernel[j-(x-size)] * s.verticalValue(k, y)
	}

	s.gridCache[key] = sum
	return sum
}

// verticalValue returns the location value after vertical smoothing
// using the provided kernel.
func (s *LazySmoothSurface) verticalValue(x, y int) float64 {
	key := y*s.Surface.Width + x
	if v := s.midGridCache[key]; !math.IsNaN(v) {
		return v
	}

	size := (len(s.kernel) - 1) / 2
	sum := 0.0

	// compute the vertical value.
	for j := y - size; j <= y+size; j++ {
		k := j
		if j < 0 {
			k = 0
		}

		if j >= s.Surface.Height {
			k = s.Surface.Height - 1
		}

		sum += s.kernel[j-(y-size)] * s.Surface.Grid[x][k]
	}

	s.midGridCache[key] = sum
	return sum
}

// The next three functions are duplicates from go.geo/surface.go, unfortunately.

// gridBoxWidth returns the width of a grid element in the units of s.Bound.
func (s LazySmoothSurface) gridBoxWidth() float64 {
	return s.Surface.Bound().Width() / float64(s.Surface.Width-1)
}

// gridBoxHeight returns the height of a grid element in the units of s.Bound.
func (s LazySmoothSurface) gridBoxHeight() float64 {
	return s.Surface.Bound().Height() / float64(s.Surface.Height-1)
}

func (s LazySmoothSurface) gridCoordinate(point *geo.Point) (x, y int, deltaX, deltaY float64) {
	w := (point[0] - s.Surface.Bound().SouthWest()[0]) / s.gridBoxWidth()
	h := (point[1] - s.Surface.Bound().SouthWest()[1]) / s.gridBoxHeight()

	x = int(math.Floor(w))
	y = int(math.Floor(h))

	deltaX = w - math.Floor(w)
	deltaY = h - math.Floor(h)

	return
}
