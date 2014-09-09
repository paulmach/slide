package utils

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// FetchURL will download the data from the URL.
// Make sure to close the returned io.ReadCloser
func FetchURL(url string, retries int) (io.ReadCloser, error) {
	var err error
	var resp *http.Response

	for i := 0; i < retries; i++ {
		// TODO: consider adding a timeout to this request.
		resp, err = http.Get(url)

		if err == nil && resp.StatusCode == http.StatusOK {
			return resp.Body, err
		}
	}

	if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
}

// BuildTileURL will fill in a template URL of something like http://host.com/{z}/{x}/{y}.png
func BuildTileURL(template string, x, y, z uint64) string {
	// I'm assuming building a replacer is faster than doing strings.Replace 3 times.
	// However, I have not tested and could be very wrong.
	r := strings.NewReplacer(
		"{x}", strconv.Itoa(int(x)),
		"{y}", strconv.Itoa(int(y)),
		"{z}", strconv.Itoa(int(z)),
		"{zoom}", strconv.Itoa(int(z)),
	)
	return r.Replace(template)
}
