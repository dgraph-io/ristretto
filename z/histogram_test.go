/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package z

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPercentile00(t *testing.T) {
	size := int(math.Ceil((float64(514) - float64(32)) / float64(4)))
	bounds := make([]float64, size+1)
	for i := range bounds {
		if i == 0 {
			bounds[0] = 32
			continue
		}
		if i == size {
			bounds[i] = 514
			break
		}
		bounds[i] = bounds[i-1] + 4
	}

	h := NewHistogramData(bounds)
	for v := 16; v <= 1024; v = v + 4 {
		for i := 0; i < 1000; i++ {
			h.Update(int64(v))
		}
	}

	require.Equal(t, h.Percentile(0.0), 32.0)
}

func TestHistogramOverflowReport(t *testing.T) {
	h := NewHistogramData([]float64{1024, 2048})
	h.Update(4096)
	require.Contains(t, h.String(), "[2.0 KiB, infinity) 1 100.00% 100.00%")
}

func TestPercentile99(t *testing.T) {
	size := int(math.Ceil((float64(514) - float64(32)) / float64(4)))
	bounds := make([]float64, size+1)
	for i := range bounds {
		if i == 0 {
			bounds[0] = 32
			continue
		}
		if i == size {
			bounds[i] = 514
			break
		}
		bounds[i] = bounds[i-1] + 4
	}
	h := NewHistogramData(bounds)
	for v := 16; v <= 512; v = v + 4 {
		for i := 0; i < 1000; i++ {
			h.Update(int64(v))
		}
	}

	require.Equal(t, h.Percentile(0.99), 512.0)
}

func TestPercentile100(t *testing.T) {
	size := int(math.Ceil((float64(514) - float64(32)) / float64(4)))
	bounds := make([]float64, size+1)
	for i := range bounds {
		if i == 0 {
			bounds[0] = 32
			continue
		}
		if i == size {
			bounds[i] = 514
			break
		}
		bounds[i] = bounds[i-1] + 4
	}
	h := NewHistogramData(bounds)
	for v := 16; v <= 1024; v = v + 4 {
		for i := 0; i < 1000; i++ {
			h.Update(int64(v))
		}
	}
	require.Equal(t, h.Percentile(1.0), 514.0)
}
