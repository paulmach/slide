Slide: Vector to Raster Map Conflation
======================================

Slide is an algorithm/approach for conflating vector data with raster data. 
The idea is to take a coarse approximation to the raster data and have the algorithm slide the polyline to the "image."
The result is a properly sampled vector polyline matching the contours of the raster data.

The algorithm is presented as a Go (golang) library. The [examples](examples) directory has some example integrations.
Most of the heavy geometry stuff is done with [go.geo](https://github.com/paulmach/go.geo), a go geography/geometry library.

### Demos

Slide supports the concept of [surfacers](surfacers) that can be based on any datasource.

* [Strava Global Heat](http://labs.strava.com/slide/demo.html) <br />
	Uses the [Strava Global Heatmap](http://labs.strava.com/heatmap/)
	to speed map tracing.

Background
----------

Slide was first developed as a tool to slide [Open Street Map](http://www.openstreetmap.org/) map geometry to the 
[Strava Global Heatmap](http://labs.strava.com/heatmap) dataset.
The 200,000,000,000 [Strava](http://strava.com) GPS points were bucketed by pixel to create the heatmap and essentially
creating a raster like density distribution of GPS data. For more information take a look at the links below:

* [Strava Slide Overview](http://labs.strava.com/slide)
* [Interactive Demo](http://labs.strava.com/slide/demo.html)
* [OSM iD Editor Integration](http://strava.github.io/iD/#background=Bing&map=16.97/-122.54464/38.05472)
* Slide [presented](http://stateofthemap.us/session/slide/) at State of the Map US 2014.

<img src="http://i.imgur.com/rbi2kDz.gif" width="728" height="330" alt="Slide Animation" style="float: right" />
<br />
**Above:**
The black polyline is being "slided" to the green path, matching the Strava global heatmap data.
Red lines are the intermediate steps of the refinement.

Algorithm Overview
------------------

At a high level, Slide works by modeling an input line, or string, sliding into the valleys of a surface. 
The surface can be built from any datasource.

One can imagine a coarse input "string of beads" being placed on the surface and letting gravity pull it downward.
When movement stops, the string should follow the valleys.

#### Details

To mimic the flexibility of a "string of beads," or something similar, the input line is resampled.
This allows the discretely sampled line to still behave like a naturally flexible object.

This resampled line is then ran through a loop where each vertex or point is corrected based on a cost function.
This cost function has 3 main parts:

* Depth with respect to the surface
* Equal distance between resampled points (smooth parametric derivative)
* Maximize vertex angles (smooth parametric second derivative)

The components are weighted with the surface getting the most. The other parts are to ensure the line doesn't
collapse in on itself and maintains some sense of rigidity.
To speed conversion of the process, a momentum component is added where
a fraction of the correction from the previous loop is added in.

Once the process converges, the line is simplified again and sent back as the result.

<img src="http://i.imgur.com/WCjdlsc.png" width="728" height="407" alt="Slide Algorithm Overview" />

#### Potential improvement

Many! There are the basic things like improving the cost function weightings 
as well as more challenging things such as incorporating more information from the input data, such as direction.
I'd also like to support the sliding of more complex goemetries, such as a road grid.

Related Work
------------

As they say, "There is nothing new under the sun."

* [Google correcting Street View sensor data](http://google-opensource.blogspot.com/2012/05/introducing-ceres-solver-nonlinear.html)
	using a nonlinear least squares solver.
* [Active Contours, Deformable Models, and Gradient Vector Flow](http://www.iacl.ece.jhu.edu/static/gvf/)
* [Snakes: Active Contour Models](http://www.cs.ucla.edu/~dt/papers/ijcv88/ijcv88.pdf), published 1988
