package z

import (
	"fmt"
	"math"
	"strings"
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

func (histogram *HistogramData) Copy() *HistogramData {
	if histogram == nil {
		return nil
	}
	return &HistogramData{
		Bounds:         append([]float64{}, histogram.Bounds...),
		CountPerBucket: append([]int64{}, histogram.CountPerBucket...),
		Count:          histogram.Count,
		Min:            histogram.Min,
		Max:            histogram.Max,
		Sum:            histogram.Sum,
	}
}

// Update changes the Min and Max fields if value is less than or greater than the current values.
func (histogram *HistogramData) Update(value int64) {
	if histogram == nil {
		return
	}
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

// String converts the histogram data into human-readable string.
func (histogram *HistogramData) String() string {
	if histogram == nil {
		return ""
	}
	var b strings.Builder

	b.WriteString(" -- Histogram: ")
	b.WriteString(fmt.Sprintf("Min value: %d ", histogram.Min))
	b.WriteString(fmt.Sprintf("Max value: %d ", histogram.Max))
	b.WriteString(fmt.Sprintf("Mean: %.2f ",
		float64(histogram.Sum)/float64(histogram.Count)))

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
			b.WriteString(fmt.Sprintf("[%d, %s) %d %.2f%% ", lowerBound, "infinity",
				count, float64(count*100)/float64(histogram.Count)))
			continue
		}

		upperBound := int(histogram.Bounds[index])
		lowerBound := 0
		if index > 0 {
			lowerBound = int(histogram.Bounds[index-1])
		}

		b.WriteString(fmt.Sprintf("[%d, %d) %d %.2f%% ", lowerBound, upperBound,
			count, float64(count*100)/float64(histogram.Count)))
	}
	b.WriteString(" --")
	return b.String()
}
