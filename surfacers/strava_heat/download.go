package strava_heat

import (
	"encoding/json"
	"sync"

	"github.com/paulmach/slide/utils"
)

// tileData mimics the data returned by the strava endpoint.
type tileData struct {
	X uint32 `json:"x"`
	Y uint32 `json:"y"`
	Z uint32 `json:"z"`

	// Data is row-major heat values between [0, 1] for the given tile.
	Data []float64 `json:"data"`
}

// downloadTiles downloads the strava heat data and puts it in the correct locations in the surface.
// It starts up surfacer.DownloadGoroutines goroutines to download the data in parallel.
func (surfacer *StravaHeatSurface) downloadTiles() error {
	// for the tiles, 0,0 is northwest. For the surface, 0,0 is south west
	verticalFlipOffset := uint64(surfacer.surface.Height - 1)

	var fetchErr error
	var wait sync.WaitGroup

	numTiles := (surfacer.xTileMax - surfacer.xTileMax + 1) * (surfacer.yTileMax - surfacer.yTileMin + 1)
	tiles := make(chan [2]uint64, numTiles)

	wait.Add(surfacer.DownloadGoroutines)
	for thread := 0; thread < surfacer.DownloadGoroutines; thread++ {
		go func() {
			defer wait.Done()

			for t := range tiles {
				x, y := t[0], t[1]

				if fetchErr != nil {
					return
				}

				url := utils.BuildTileURL(surfacer.SourceURLTemplate, x, y, surfacer.level)
				body, err := utils.FetchURL(url, surfacer.DownloadRetries)
				if err == nil {
					var k, l uint64

					data := &tileData{}
					err = json.NewDecoder(body).Decode(&data)
					body.Close()

					if err != nil {
						fetchErr = err
						return
					}

					xStart := (x - surfacer.xTileMin) * 256
					yStart := (y - surfacer.yTileMin) * 256

					for k = 0; k < 256; k++ {
						offset := verticalFlipOffset - (yStart + k)
						for l = 0; l < 256; l++ {
							// while this does a matrix transpose. I did not get any speedup when
							// trying to mess with this.
							surfacer.surface.Grid[xStart+l][offset] = data.Data[k*256+l]
						}
					}
				} else {
					fetchErr = err
					return
				}
			}
		}()
	}

	for i := surfacer.xTileMin; i <= surfacer.xTileMax; i++ {
		for j := surfacer.yTileMin; j <= surfacer.yTileMax; j++ {
			tiles <- [2]uint64{i, j}
		}
	}
	close(tiles)
	wait.Wait()

	if fetchErr != nil {
		return fetchErr
	}

	return nil
}
