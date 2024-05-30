package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

type (
	LayerStatus struct {
		Status string
		Time   float64 // Time in seconds; only set for DONE layers
	}

	CacheMetrics struct {
		TotalLayers int                 `json:"total_layers"`
		Done        int                 `json:"done"`
		Cached      int                 `json:"cached"`
		Errored     int                 `json:"errored"`
		Canceled    int                 `json:"canceled"`
		Layers      map[int]LayerStatus `json:"layers"`
	}
)

func parseCacheMetrics(ch <-chan string) (CacheMetrics, error) {
	var cacheMetrics CacheMetrics
	cacheMetrics.Layers = make(map[int]LayerStatus) // Initialize the map

	re := regexp.MustCompile(`#(\d+) (DONE|CACHED|ERRORED|CANCELED)(?: ([0-9.]+)s)?`)

	for line := range ch {
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) <= 2 {
				continue
			}
			layerIndex, err := strconv.Atoi(match[1])
			if err != nil {
				return cacheMetrics, fmt.Errorf("failed to convert layer index %s to int", match[1])
			}
			status := match[2]
			layerStatus := LayerStatus{Status: status}

			switch status {
			case "DONE":
				cacheMetrics.Done++
				if len(match) == 4 && match[3] != "" {
					if duration, err := strconv.ParseFloat(match[3], 64); err == nil {
						layerStatus.Time = duration
					}
				}
			case "CACHED":
				cacheMetrics.Cached++
			case "ERRORED":
				cacheMetrics.Errored++
			case "CANCELED":
				cacheMetrics.Canceled++
			}
			cacheMetrics.Layers[layerIndex] = layerStatus
		}
	}

	cacheMetrics.TotalLayers = cacheMetrics.Done + cacheMetrics.Cached + cacheMetrics.Errored + cacheMetrics.Canceled

	return cacheMetrics, nil
}

func writeCacheMetrics(data CacheMetrics, filename string) error {
	b, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return fmt.Errorf("failed with err %s to marshal output %+v", err, data)
	}

	dir := filepath.Dir(filename)
	err = os.MkdirAll(dir, 0644)
	if err != nil {
		return fmt.Errorf("failed with err %s to create %s directory for cache metrics file", err, dir)
	}

	err = os.WriteFile(filename, b, 0644)
	if err != nil {
		return fmt.Errorf("failed to write cache metrics to cache metrics file %s", filename)
	}
	return nil
}
