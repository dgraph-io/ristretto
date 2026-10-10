/*
 * SPDX-FileCopyrightText: Copyright 2026 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package humanize

import (
	"fmt"
	"math"
	"math/bits"
)

// IBytes formats a byte count using IEC units for diagnostic output.
func IBytes(size uint64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := [...]string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	exponent := (bits.Len64(size) - 1) / 10
	value := math.Round(float64(size)/float64(uint64(1)<<(10*exponent))*10) / 10
	if value < 10 {
		return fmt.Sprintf("%.1f %s", value, units[exponent])
	}
	return fmt.Sprintf("%.0f %s", value, units[exponent])
}
