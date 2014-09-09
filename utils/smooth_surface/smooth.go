package smooth_surface

import (
	"sync"

	"github.com/paulmach/go.geo"
)

// Smooth applies a kernel vertically and horizontally in place on the surface.
// For most purposes the SmoothSurface object may be more efficient since it only
// smooths values that are accessed.
// TODO: There is probably a better, cleaner way to do this.
func Smooth(surface *geo.Surface, kernel []float64, goroutines ...int) {
	threads := 1
	if len(goroutines) > 0 || goroutines[0] > 1 {
		threads = goroutines[0]
	}

	size := (len(kernel) - 1) / 2
	height := surface.Height
	width := surface.Width

	cacheSurface := geo.NewSurface(surface.Bound(), surface.Width, surface.Height)

	// go from left to right and smooth vertically
	indexes := make(chan int, width)
	var wait sync.WaitGroup

	wait.Add(threads)
	for i := 0; i < threads; i++ {
		go func() {
			defer wait.Done()

			for i := range indexes {
				// first part overlapping the edge
				for j := 0; j < size; j++ {
					var sum float64

					for k := j - size; k < 0; k++ {
						sum += surface.Grid[i][0] * kernel[k-(j-size)]
					}

					for k := 0; k <= j+size; k++ {
						sum += surface.Grid[i][k] * kernel[k-(j-size)]
					}

					cacheSurface.Grid[i][j] = sum
				}

				// middle easy stuff
				for j := size; j < height-size-1; j++ {
					var sum float64

					// presetting these values does improve benchmarks
					// probably something with the compiler knowing to optimize
					length := 2*size + 1
					slice := surface.Grid[i][j-size : j+size+1]
					for k := 0; k < length; k++ {
						sum += slice[k] * kernel[k]
					}

					cacheSurface.Grid[i][j] = sum
				}

				// last part overlapping the edge
				for j := height - size - 1; j < height; j++ {
					var sum float64

					for k := j - size; k < height; k++ {
						sum += surface.Grid[i][k] * kernel[k-(j-size)]
					}

					for k := height; k <= j+size; k++ {
						sum += surface.Grid[i][height-1] * kernel[k-(j-size)]
					}

					cacheSurface.Grid[i][j] = sum
				}
			}
		}()
	}

	// seed the indexes to computer by the go routines
	for i := 0; i < width; i++ {
		indexes <- i
	}
	close(indexes)
	wait.Wait()

	// go from bottom to top and smooth horizontally
	indexes = make(chan int, height) // reopen

	wait.Add(threads)
	for i := 0; i < threads; i++ {
		go func() {
			defer wait.Done()

			for j := range indexes {
				// first part overlapping the edge
				for i := 0; i < size && i < width; i++ {
					var sum float64

					// part of the kernel that lands before our data
					for k := i - size; k < 0; k++ {
						sum += cacheSurface.Grid[0][j] * kernel[k-(i-size)]
					}

					for k := 0; k <= i+size && k < width; k++ {
						sum += cacheSurface.Grid[k][j] * kernel[k-(i-size)]
					}

					// part of the kernel that lands after our data, ie size > width
					for k := width; k <= i+size; k++ {
						sum += cacheSurface.Grid[width-1][j] * kernel[k-(i-size)]
					}

					surface.Grid[i][j] = sum
				}

				for i := size; i < width-size-1; i++ {
					var sum float64

					// presetting these values does improve benchmarks
					// probably something with the compiler knowing to optimize
					length := 2*size + 1
					slice := cacheSurface.Grid[i-size : i+size+1]
					for k := 0; k < length; k++ {
						sum += slice[k][j] * kernel[k]
					}

					surface.Grid[i][j] = sum
				}

				// last part overlapping the edge
				start := width - size - 1
				if start < 0 {
					start = 0
				}
				for i := start; i < width; i++ {
					var sum float64

					for k := i - size; k < 0; k++ {
						sum += cacheSurface.Grid[0][j] * kernel[k-(i-size)]
					}

					start := i - size
					if start < 0 {
						start = 0
					}
					for k := start; k < width; k++ {
						sum += cacheSurface.Grid[k][j] * kernel[k-(i-size)]
					}

					for k := width; k <= i+size; k++ {
						sum += cacheSurface.Grid[width-1][j] * kernel[k-(i-size)]
					}

					surface.Grid[i][j] = sum
				}
			}
		}()
	}

	// seed the indexes to computer by the go routines
	for i := 0; i < height; i++ {
		indexes <- i
	}
	close(indexes)
	wait.Wait()

	return
}
