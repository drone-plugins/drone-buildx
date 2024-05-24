package docker

import (
	"reflect"
	"testing"
)

func TestParseCacheMetrics(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		expected  CacheMetrics
		expectErr bool
	}{
		{
			name:     "empty channel",
			input:    []string{},
			expected: CacheMetrics{Layers: make(map[int]LayerStatus)},
		},
		{
			name: "valid metrics with multiple on same line",
			input: []string{
				"#1 DONE 0.5s #2 CACHED #3 ERRORED #4 CANCELLED",
				"#5 DONE 1.0s #6 CACHED",
			},
			expected: CacheMetrics{
				Layers: map[int]LayerStatus{
					1: {Status: "DONE", Time: 0.5},
					2: {Status: "CACHED"},
					3: {Status: "ERRORED"},
					4: {Status: "CANCELLED"},
					5: {Status: "DONE", Time: 1.0},
					6: {Status: "CACHED"},
				},
				Done:        2,
				Cached:      2,
				Errored:     1,
				Cancelled:   1,
				TotalLayers: 6,
			},
		},
		{
			name: "mixed valid and invalid lines with multiple on same line",
			input: []string{
				"#1 DONE 0.5s #2 INVALID #3 CACHED",
				"#4 DONE #invalid_line",
			},
			expected: CacheMetrics{
				Layers: map[int]LayerStatus{
					1: {Status: "DONE", Time: 0.5},
					3: {Status: "CACHED"},
					4: {Status: "DONE"},
				},
				Done:        2,
				Cached:      1,
				TotalLayers: 3,
			},
		},
		{
			name: "various statuses with multiple on same line",
			input: []string{
				"#1 DONE 1.2s #2 DONE 0.8s",
				"#3 CACHED #4 ERRORED #5 CANCELLED",
				"#6 DONE",
			},
			expected: CacheMetrics{
				Layers: map[int]LayerStatus{
					1: {Status: "DONE", Time: 1.2},
					2: {Status: "DONE", Time: 0.8},
					3: {Status: "CACHED"},
					4: {Status: "ERRORED"},
					5: {Status: "CANCELLED"},
					6: {Status: "DONE"},
				},
				Done:        3,
				Cached:      1,
				Errored:     1,
				Cancelled:   1,
				TotalLayers: 6,
			},
		},
		{
			name: "invalid formats with multiple on same line",
			input: []string{
				"#1 DONE 0.5s #2 INVALID",
				"#3 CACHED #invalid_line",
			},
			expected: CacheMetrics{
				Layers: map[int]LayerStatus{
					1: {Status: "DONE", Time: 0.5},
					3: {Status: "CACHED"},
				},
				Done:        1,
				Cached:      1,
				TotalLayers: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan string, len(tt.input))
			go func() {
				for _, line := range tt.input {
					ch <- line
				}
				close(ch)
			}()

			actual, err := parseCacheMetrics(ch)
			if (err != nil) != tt.expectErr {
				t.Errorf("parseCacheMetrics() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("parseCacheMetrics() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}
