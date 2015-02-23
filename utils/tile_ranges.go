package utils

import "github.com/paulmach/go.geo"

// TileRanges returns the ranges of tiles that enclose the given latlng bound.
func TileRanges(lnglatBound *geo.Bound, maxTileDim uint64) (xTileMin, xTileMax, yTileMin, yTileMax, zoomLevel uint64) {

	// this is as far in as we'll go down in detail. ie. no more detail.
	// This seems to work well in practice.
	zoomLevel = uint64(17)

	sw := lnglatBound.SouthWest()
	ne := lnglatBound.NorthEast()

	// loop the levels until we are within our maxTileDim x maxTileDim tile maximum
	xTileMin, xTileMax, yTileMin, yTileMax = 0, 100, 0, 100
	for xTileMax-xTileMin+1 > maxTileDim || yTileMax-yTileMin+1 > maxTileDim {
		if zoomLevel == 0 {
			panic("unable to find a tile range containing the bound")
		}

		// the +-1 shifting of the values guarantees at least half a tile around the edges.
		// Vertical or horizontal lines near a tile boundary would just not have enough surface
		// to do the calculation right.
		shift := geo.ScalarMercator.Level - (zoomLevel + 1)

		// NOTE: y-tiles increase from top down
		xTileMin, yTileMax = geo.ScalarMercator.Project(sw.Lng(), sw.Lat())
		xTileMax, yTileMin = geo.ScalarMercator.Project(ne.Lng(), ne.Lat())

		xTileMin >>= shift
		xTileMax >>= shift
		yTileMin >>= shift
		yTileMax >>= shift

		xTileMin = (xTileMin - 1) >> 1
		yTileMin = (yTileMin - 1) >> 1

		xTileMax = (xTileMax + 1) >> 1
		yTileMax = (yTileMax + 1) >> 1

		zoomLevel--
	}
	zoomLevel++

	return xTileMin, xTileMax, yTileMin, yTileMax, zoomLevel
}
