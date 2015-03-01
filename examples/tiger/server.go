package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/paulmach/go.geo"
	"github.com/paulmach/reducer"
	"github.com/paulmach/slide"
	"github.com/paulmach/slide/reducers"
	"github.com/paulmach/slide/surfacers/tileoverlay"
)

// some defaults
const (
	port = 8080

	defaultSmoothingSD      = 16.0 // meters that is one sd or smoothing
	defaultThresholdEpsilon = 0.0001
)

// SlideResult is the structure delivered to the browser after the sliding process
// TODO: GeoJSON this!
type SlideResult struct {
	Corrected          string        `json:"corrected_path"`
	LoopsCompleted     int           `json:"loops_completed"`
	GetSurfaceDuration time.Duration `json:"get_surface_duration"`
	CorrectionDuration time.Duration `json:"correction_duration"`
	NumPoints          int           `json:"num_points"`
	IntermediatePaths  []string      `json:"intermediate_paths"`
	Error              string        `json:"error,omitempty"`
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	http.Handle("/", http.FileServer(http.Dir("www")))
	http.HandleFunc("/slide", slideHandler)

	/**********************************/
	// listen and serve
	log.Printf("Listening on port %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func slideHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	paths := []*geo.Path{geo.Decode(r.FormValue("path"), 1e6)}

	/**********************************
	 * build the surface */
	smoothingSD := defaultSmoothingSD
	if r.FormValue("smoothing_sd") != "" {
		smoothingSD, _ = strconv.ParseFloat(r.FormValue("smoothing_sd"), 64)
	}

	tigerSurfacer := tileoverlay.New(
		paths[0].Bound(),
		"http://a.tiles.mapbox.com/v3/enf.y5c4ygb9,enf.ho20a3n1,enf.game1617/{z}/{x}/{y}.png", // tile source
		smoothingSD,
		color.NRGBA{R: 255, G: 255, B: 0, A: 255}, // target color (yellow)
	)

	surfacerStartTime := time.Now()
	err := tigerSurfacer.Build() // actual fetch tiles and setup everything
	if err != nil {
		http.Error(w, fmt.Sprintf("Server Error: %v", err), http.StatusServiceUnavailable)
		return
	}
	surfaceBuildTime := time.Since(surfacerStartTime)

	/**********************************
	 * slider and options */
	slider := slide.New(paths, tigerSurfacer)
	slider.GeoReducer = reducers.NewTrim(reducer.New())
	slider.ThresholdEpsilon = defaultThresholdEpsilon

	if r.FormValue("gradient_scale") != "" {
		slider.GradientScale, _ = strconv.ParseFloat(r.FormValue("gradient_scale"), 64)
	}

	if r.FormValue("distance_scale") != "" {
		slider.DistanceScale, _ = strconv.ParseFloat(r.FormValue("distance_scale"), 64)
	}

	if r.FormValue("angle_scale") != "" {
		slider.AngleScale, _ = strconv.ParseFloat(r.FormValue("angle_scale"), 64)
	}

	if r.FormValue("momentum_scale") != "" {
		slider.MomentumScale, _ = strconv.ParseFloat(r.FormValue("momentum_scale"), 64)
	}

	if r.FormValue("number_intermediate_paths") != "" {
		numberIntermediatePaths, _ := strconv.ParseInt(r.FormValue("number_intermediate_paths"), 10, 64)
		slider.NumberIntermediateGeometries = int(numberIntermediatePaths)
	}

	// run the slide
	slideResult, err := slider.Do()
	if err != nil {
		http.Error(w, fmt.Sprintf("Server Error: %v", err), http.StatusServiceUnavailable)
		return
	}

	// massage the results
	result := &SlideResult{
		Corrected:          slideResult.CorrectedGeometry[0].Encode(1e6),
		LoopsCompleted:     slideResult.LoopsCompleted,
		GetSurfaceDuration: surfaceBuildTime,
		CorrectionDuration: slideResult.Runtime,
		NumPoints:          slideResult.CorrectedGeometry[0].Length(),
	}

	result.IntermediatePaths = make([]string, len(slideResult.IntermediateGeometry))
	for i, g := range slideResult.IntermediateGeometry {
		result.IntermediatePaths[i] = g[0].Encode(1e6)
	}

	/**********************************
	 * more slides with progressively sharper surfaces */
	for i := tigerSurfacer.SmoothingStdDev - 1; i >= 1.0; i -= 1.0 {
		tigerSurfacer.SmoothingStdDev = i
		tigerSurfacer.Resmooth()

		slider = slide.New(slideResult.CorrectedGeometry, tigerSurfacer)

		// different parameters for this refinement step.
		slider.DepthBasedReduction = true
		slider.ThresholdEpsilon = defaultThresholdEpsilon
		slider.ResampleInterval = 3.0
		slider.GradientScale /= 3.0
		slider.MomentumScale = 0.0

		slider.NumberIntermediateGeometries = 0
		slider.MaxLoops = slider.MinLoops // cap loops

		slideResult, err = slider.Do()
		if err != nil {
			http.Error(w, fmt.Sprintf("Server Error: %v", err), http.StatusServiceUnavailable)
			return
		}
	}

	result.Corrected = slideResult.CorrectedGeometry[0].Encode(1e6)

	/**********************************
	 * deliver the result */
	json, _ := json.Marshal(result)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(json)
}
