// Package analytics provides fast, dependency-free statistics for things
// the Go backend wants to compute in-process — KPI cards, sparkline
// trends, percent-change badges — without a round trip to the Python
// analytics-service. Deeper analysis (regression, ARIMA, anomaly
// detection) stays in analytics-service, which has the library depth
// (statsmodels, PyOD) that doesn't belong in a hand-rolled Go package.
//
// This intentionally uses only the standard library rather than a
// third-party Go DataFrame/stats package: without a Go toolchain or
// network access in the environment these files were written in, an
// external dependency's exact API couldn't be verified before shipping.
// Swapping in something like github.com/montanaflynn/stats or
// gonum.org/v1/gonum/stat later is a reasonable follow-up once that can
// be checked with `go doc` / a real build.
package analytics

import (
	"fmt"
	"math"
	"sort"
)

func Mean(xs []float64) (float64, error) {
	if len(xs) == 0 {
		return 0, fmt.Errorf("mean: empty input")
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs)), nil
}

func Median(xs []float64) (float64, error) {
	if len(xs) == 0 {
		return 0, fmt.Errorf("median: empty input")
	}
	sorted := append([]float64(nil), xs...)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2], nil
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2, nil
}

// StdDev returns the sample standard deviation (N-1 denominator).
func StdDev(xs []float64) (float64, error) {
	if len(xs) < 2 {
		return 0, fmt.Errorf("stddev: need at least 2 points")
	}
	mean, _ := Mean(xs)

	var sumSq float64
	for _, x := range xs {
		d := x - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(xs)-1)), nil
}

// Correlation returns the Pearson correlation coefficient between two
// equal-length series, in [-1, 1].
func Correlation(xs, ys []float64) (float64, error) {
	if len(xs) != len(ys) {
		return 0, fmt.Errorf("correlation: series must be the same length")
	}
	if len(xs) < 2 {
		return 0, fmt.Errorf("correlation: need at least 2 points")
	}

	meanX, _ := Mean(xs)
	meanY, _ := Mean(ys)

	var sumXY, sumX2, sumY2 float64
	for i := range xs {
		dx := xs[i] - meanX
		dy := ys[i] - meanY
		sumXY += dx * dy
		sumX2 += dx * dx
		sumY2 += dy * dy
	}

	denom := math.Sqrt(sumX2 * sumY2)
	if denom == 0 {
		return 0, fmt.Errorf("correlation: undefined (zero variance in a series)")
	}
	return sumXY / denom, nil
}

// PercentChange returns the percentage change from `from` to `to`, the
// number that drives a dashboard KPI card's up/down badge.
func PercentChange(from, to float64) (float64, error) {
	if from == 0 {
		return 0, fmt.Errorf("percent change: base value is zero")
	}
	return (to - from) / math.Abs(from) * 100, nil
}

// Sparkline downsamples a series to at most n points by simple striding,
// enough for a small trend chart on a KPI card without shipping the full
// series to the frontend.
func Sparkline(xs []float64, n int) []float64 {
	if n <= 0 || len(xs) <= n {
		return xs
	}
	step := float64(len(xs)) / float64(n)
	out := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		idx := int(float64(i) * step)
		if idx >= len(xs) {
			idx = len(xs) - 1
		}
		out = append(out, xs[idx])
	}
	return out
}
