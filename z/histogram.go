package z

import (
	"fmt"
	"math"
)

// Creates bounds for an histogram. The bounds are powers of two of the form
// [2^min_exponent, ..., 2^max_exponent].
func HistogramBounds(minExponent, maxExponent uint32) []float64 {
	var bounds []float64
	for i := minExponent; i <= maxExponent; i++ {
		bounds = append(bounds, float64(int(1)<<i))
	}
	return bounds
}

// HistogramData stores the information needed to represent the sizes of the keys and values
// as a histogram.
type HistogramData struct {
	Bounds         []float64
	Count          int64
	CountPerBucket []int64
	Min            int64
	Max            int64
	Sum            int64
}

// NewHistogramData returns a new instance of HistogramData with properly initialized fields.
func NewHistogramData(bounds []float64) *HistogramData {
	return &HistogramData{
		Bounds:         bounds,
		CountPerBucket: make([]int64, len(bounds)+1),
		Max:            0,
		Min:            math.MaxInt64,
	}
}

// Update changes the Min and Max fields if value is less than or greater than the current values.
func (histogram *HistogramData) Update(value int64) {
	if value > histogram.Max {
		histogram.Max = value
	}
	if value < histogram.Min {
		histogram.Min = value
	}

	histogram.Sum += value
	histogram.Count++

	for index := 0; index <= len(histogram.Bounds); index++ {
		// Allocate value in the last buckets if we reached the end of the Bounds array.
		if index == len(histogram.Bounds) {
			histogram.CountPerBucket[index]++
			break
		}

		if value < int64(histogram.Bounds[index]) {
			histogram.CountPerBucket[index]++
			break
		}
	}
}

// PrintHistogram prints the histogram data in a human-readable format.
func (histogram *HistogramData) PrintHistogram() {
	if histogram == nil {
		return
	}

	fmt.Printf("Min value: %d\n", histogram.Min)
	fmt.Printf("Max value: %d\n", histogram.Max)
	fmt.Printf("Mean: %.2f\n", float64(histogram.Sum)/float64(histogram.Count))
	fmt.Printf("%24s %9s\n", "Range", "Count")

	numBounds := len(histogram.Bounds)
	for index, count := range histogram.CountPerBucket {
		if count == 0 {
			continue
		}

		// The last bucket represents the bucket that contains the range from
		// the last bound up to infinity so it's processed differently than the
		// other buckets.
		if index == len(histogram.CountPerBucket)-1 {
			lowerBound := int(histogram.Bounds[numBounds-1])
			fmt.Printf("[%10d, %10s) %9d\n", lowerBound, "infinity", count)
			continue
		}

		upperBound := int(histogram.Bounds[index])
		lowerBound := 0
		if index > 0 {
			lowerBound = int(histogram.Bounds[index-1])
		}

		fmt.Printf("[%10d, %10d) %9d\n", lowerBound, upperBound, count)
	}
}
