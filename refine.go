package slide

import (
	"math"
	"sync"

	"github.com/paulmach/go.geo"
)

const (
	scoreSmoothingFactor = 0.2 // (0, 1.0), higher is more smoothing
)

type workerPayload struct {
	Path        *geo.Path
	NewPath     *geo.Path
	Index       int
	Corrections []geo.Point
	WG          *sync.WaitGroup
}

// refine does the iterative refinement.
func (s *Slide) refine() (*Result, error) {
	var (
		loop         int
		delta        float64
		currentScore float64
		pathScore    float64
	)

	// currently only one line is supported. TODO: improve.
	path := s.Geometry[0]

	// start the workers
	var workersWG sync.WaitGroup
	payloads := make(chan workerPayload, 100)

	workersWG.Add(s.Goroutines)
	for i := 0; i < s.Goroutines; i++ {
		go s.refineWorker(payloads, &workersWG)
	}

	intermediateGeometries := make([][]*geo.Path, 0, s.NumberIntermediateGeometries)
	previousCorrections := make([]geo.Point, path.Length()) // used for momentum

	for loop = 0; loop < s.MaxLoops; loop++ {
		newPath := path.Clone()

		var wait sync.WaitGroup
		wait.Add(path.Length() - 2)

		for j := 1; j < path.Length()-1; j++ {
			payloads <- workerPayload{
				Path:        path,
				Corrections: previousCorrections,
				NewPath:     newPath,
				Index:       j,
				WG:          &wait,
			}
		}
		wait.Wait()

		path = newPath // new becomes current
		if loop < s.NumberIntermediateGeometries {
			intermediateGeometries = append(intermediateGeometries, []*geo.Path{path})
		}

		// check how we did
		// First, compute the score taking the average surface value.
		// Then exponentially smooth those values and keep looping until they don't change very much.
		pathScore = averageSurfaceValue(s.Surfacer, path)

		previousScore := currentScore
		currentScore = scoreSmoothingFactor*previousScore + (1-scoreSmoothingFactor)*pathScore

		delta = math.Abs(currentScore - previousScore)

		// break condition
		if loop >= s.MinLoops && delta < s.ThresholdEpsilon {
			break
		}
	}

	// shut down the workers
	close(payloads)
	workersWG.Wait()

	// simplify path
	path = path.Clone() // path is pointer, so may be in intermediateGeometries above

	return &Result{
		CorrectedGeometry:    []*geo.Path{path},
		IntermediateGeometry: intermediateGeometries,
		LoopsCompleted:       loop,
		LastLoopError:        delta,
		LastLoopScore:        pathScore,
	}, nil
}

func (s *Slide) refineWorker(payloads <-chan workerPayload, finish *sync.WaitGroup) {
	defer finish.Done()

	for load := range payloads {
		gradient := s.GradientContributionFunc(s.Surfacer, load.Path.GetAt(load.Index), s.GradientScale)
		distance := s.DistanceContributionFunc(load.Path, load.Index, s.DistanceScale)
		angle := s.AngleContributionFunc(load.Path, load.Index, s.AngleScale)

		// put them together
		correction := geo.NewPoint(0, 0).Add(distance).Add(angle).Add(gradient)
		correction.Add(load.Corrections[load.Index].Scale(s.MomentumScale))

		if s.DepthBasedReduction {
			v := s.Surfacer.ValueAt(load.Path.GetAt(load.Index))
			correction.Scale(math.Sqrt(1.0 - v))
		}

		load.NewPath.GetAt(load.Index).Add(correction)
		load.Corrections[load.Index] = *correction

		load.WG.Done()
	}
}

func gradientContribution(surfacer Surfacer, point *geo.Point, scale float64) *geo.Point {
	gradient := geo.NewPoint(0, 0)
	if scale != 0.0 {
		gradient = surfacer.GradientAt(point)
		gradient.Scale(scale)
	}

	return gradient
}

func distanceContribution(path *geo.Path, index int, scale float64) *geo.Point {
	distance := geo.NewPoint(0, 0)
	if scale != 0.0 {
		v := path.GetAt(index).Clone().Subtract(path.GetAt(index - 1))
		u := path.GetAt(index + 1).Clone().Subtract(path.GetAt(index - 1))

		dot := u.Dot(u)
		if dot != 0 {
			// normal case
			center := u.Clone().Scale(u.Dot(v) / dot).Add(path.GetAt(index - 1))

			m2 := path.GetAt(index + 1).Clone().Subtract(center)
			m1 := path.GetAt(index - 1).Clone().Subtract(center)
			distance = m1.Add(m2).Scale(scale)
		} else {
			// equal to zero if the points are the same
			// good times with round off error
		}
	}

	return distance
}

func angleContribution(path *geo.Path, index int, scale float64) *geo.Point {
	angle := geo.NewPoint(0, 0)

	if scale != 0.0 {
		n1 := path.GetAt(index - 1).Clone().Subtract(path.GetAt(index))
		n2 := path.GetAt(index + 1).Clone().Subtract(path.GetAt(index))

		len1 := n1.DistanceFrom(geo.NewPoint(0, 0))
		len2 := n2.DistanceFrom(geo.NewPoint(0, 0))

		n1.Normalize()
		n2.Normalize()

		// cbrt
		factor := math.Cbrt(n1.Dot(n2)) + 1
		angle = n1.Add(n2).Normalize().Scale(math.Min(len1, len2) * scale * factor)
	}

	return angle
}

func averageSurfaceValue(surfacer Surfacer, path *geo.Path) float64 {
	valueSum := 0.0
	for i := 0; i < path.Length(); i++ {
		valueSum += surfacer.ValueAt(path.GetAt(i))
	}

	return valueSum / float64(path.Length())
}
