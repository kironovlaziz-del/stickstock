package analytics

import (
	"math"
	"testing"
)

func TestMean(t *testing.T) {
	tests := []struct {
		name string
		data []float64
		want float64
		err  bool
	}{
		{"normal", []float64{1, 2, 3, 4, 5}, 3, false},
		{"empty", []float64{}, 0, true},
		{"single", []float64{10}, 10, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Mean(tt.data)
			if (err != nil) != tt.err {
				t.Errorf("Mean() error = %v, wantErr %v", err, tt.err)
				return
			}
			if !tt.err && math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("Mean() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedian(t *testing.T) {
	tests := []struct {
		name string
		data []float64
		want float64
		err  bool
	}{
		{"odd", []float64{1, 2, 3, 4, 5}, 3, false},
		{"even", []float64{1, 2, 3, 4}, 2.5, false},
		{"empty", []float64{}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Median(tt.data)
			if (err != nil) != tt.err {
				t.Errorf("Median() error = %v, wantErr %v", err, tt.err)
				return
			}
			if !tt.err && math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("Median() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStdDev(t *testing.T) {
	tests := []struct {
		name string
		data []float64
		want float64
		err  bool
	}{
		{"normal", []float64{1, 2, 3, 4, 5}, 1.5811388300841898, false},
		{"less2", []float64{1}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StdDev(tt.data)
			if (err != nil) != tt.err {
				t.Errorf("StdDev() error = %v, wantErr %v", err, tt.err)
				return
			}
			if !tt.err && math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("StdDev() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCorrelation(t *testing.T) {
	tests := []struct {
		name string
		x    []float64
		y    []float64
		want float64
		err  bool
	}{
		{"perfect_positive", []float64{1, 2, 3}, []float64{2, 4, 6}, 1, false},
		{"perfect_negative", []float64{1, 2, 3}, []float64{6, 4, 2}, -1, false},
		{"zero_correlation", []float64{1, 2, 3}, []float64{1, 1, 1}, 0, true},
		{"diff_len", []float64{1, 2}, []float64{1, 2, 3}, 0, true},
		{"less2", []float64{1}, []float64{1}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Correlation(tt.x, tt.y)
			if (err != nil) != tt.err {
				t.Errorf("Correlation() error = %v, wantErr %v", err, tt.err)
				return
			}
			if !tt.err && math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("Correlation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPercentChange(t *testing.T) {
	tests := []struct {
		name string
		from float64
		to   float64
		want float64
		err  bool
	}{
		{"positive", 100, 120, 20, false},
		{"negative", 100, 80, -20, false},
		{"zero_from", 0, 10, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PercentChange(tt.from, tt.to)
			if (err != nil) != tt.err {
				t.Errorf("PercentChange() error = %v, wantErr %v", err, tt.err)
				return
			}
			if !tt.err && math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("PercentChange() = %v, want %v", got, tt.want)
			}
		})
	}
}
