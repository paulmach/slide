package tileoverlay

import (
	"image"
	"sync"

	"github.com/paulmach/slide/utils"
)

import _ "image/png" // to support tiles in these formats automatically
import _ "image/jpeg"
import _ "image/gif"

func (surfacer *TileOverlaySurface) downloadTiles() error {
	// for the tiles, 0,0 is northwest. For the surface, 0,0 is south west
	verticalFlipOffset := uint64(surfacer.Surface.Height - 1)

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

				// fetch tile data
				url := utils.BuildTileURL(surfacer.SourceURLTemplate, x, y, surfacer.level)
				body, err := utils.FetchURL(url, surfacer.DownloadRetries)
				if err == nil {
					var k, l uint64

					img, _, err := image.Decode(body)
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
							// while this does a matrix transpose, I did not get any speedup when trying to mess with this.
							surfacer.Surface.Grid[xStart+l][offset] = surfacer.ColorValueFunc(img.At(int(l), int(k)), surfacer.targetColor)
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
