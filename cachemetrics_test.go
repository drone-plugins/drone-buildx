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
			expected: CacheMetrics{},
		},
		{
			name: "valid metrics with multiple on same line",
			input: []string{
				"#1 DONE 0.5s #2 CACHED #3 ERRORED #4 CANCELED",
				"#5 DONE 1.0s #6 CACHED",
			},
			expected: CacheMetrics{
				Layers: []Layer{
					{Index: 1, LayerStatus: LayerStatus{Status: "DONE", Time: 0.5}},
					{Index: 2, LayerStatus: LayerStatus{Status: "CACHED"}},
					{Index: 3, LayerStatus: LayerStatus{Status: "ERRORED"}},
					{Index: 4, LayerStatus: LayerStatus{Status: "CANCELED"}},
					{Index: 5, LayerStatus: LayerStatus{Status: "DONE", Time: 1.0}},
					{Index: 6, LayerStatus: LayerStatus{Status: "CACHED"}},
				},
				Done:        2,
				Cached:      2,
				Errored:     1,
				Canceled:    1,
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
				Layers: []Layer{
					{Index: 1, LayerStatus: LayerStatus{Status: "DONE", Time: 0.5}},
					{Index: 3, LayerStatus: LayerStatus{Status: "CACHED"}},
					{Index: 4, LayerStatus: LayerStatus{Status: "DONE"}},
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
				"#3 CACHED #4 ERRORED #5 CANCELED",
				"#6 DONE",
			},
			expected: CacheMetrics{
				Layers: []Layer{
					{Index: 1, LayerStatus: LayerStatus{Status: "DONE", Time: 1.2}},
					{Index: 2, LayerStatus: LayerStatus{Status: "DONE", Time: 0.8}},
					{Index: 3, LayerStatus: LayerStatus{Status: "CACHED"}},
					{Index: 4, LayerStatus: LayerStatus{Status: "ERRORED"}},
					{Index: 5, LayerStatus: LayerStatus{Status: "CANCELED"}},
					{Index: 6, LayerStatus: LayerStatus{Status: "DONE"}},
				},
				Done:        3,
				Cached:      1,
				Errored:     1,
				Canceled:    1,
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
				Layers: []Layer{
					{Index: 1, LayerStatus: LayerStatus{Status: "DONE", Time: 0.5}},
					{Index: 3, LayerStatus: LayerStatus{Status: "CACHED"}},
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
