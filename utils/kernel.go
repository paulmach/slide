package utils

import (
	"math"
)

// Kernel creates a good kernel for smoothing surfaces for the purpose of sliding.
// Creates a Gaussian kernel with one difference, points within on STD of the center follow f(x) = (-a  / sd) * x + (a + 1/(e^0.5)).
// This creates a shape where we have a line from (sd, 1/e^0.5) to (0, 1/e^0.5 + addition), ie. a sharp point at the center of the Gaussian.
// a (the addition) is defined as sd*1.5 and can be tuned to change the sharpness of the kernel.
// These parameters are scaled to match the scaling caused by the mercator projection assumed by slide.
func Kernel(stdDev, mercatorScale float64) []float64 {
	if stdDev == 0 {
		return []float64{1.0}
	}

	// addition is the extra depth of v within one std of center of kernel.
	addition := stdDev * 1.5

	sd := stdDev * mercatorScale
	depth := math.Sqrt(mercatorScale) / (addition + 1.0/math.SqrtE)

	size := int(math.Ceil(sd * 3.5)) // we go 3.5 sds out, everything beyond that is zero
	kernel := make([]float64, 2*size+1)

	// create the kernel
	for i := 0; i <= size; i++ {
		var x float64

		if float64(i) < sd {
			// something linear
			x = -addition/sd*float64(i) + (addition + 1.0/math.SqrtE)

		} else {
			x = float64(i) / sd
			x = math.Exp(-x * x)
		}

		kernel[size-i] = x * depth
		kernel[size+i] = x * depth
	}

	return kernel
}
