/*
 * SPDX-FileCopyrightText: Copyright 2026 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package humanize

import (
	"math"
	"strconv"
	"testing"
)

// TestIBytes checks IEC unit boundaries, rounding, and the full uint64 range.
func TestIBytes(t *testing.T) {
	for _, test := range []struct {
		size uint64
		want string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{9, "9 B"},
		{10, "10 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{10188, "9.9 KiB"},
		{10189, "10 KiB"},
		{10240, "10 KiB"},
		{10701, "10 KiB"},
		{11725, "12 KiB"},
		{1<<20 - 1, "1024 KiB"},
		{1 << 20, "1.0 MiB"},
		{1<<20 + 1, "1.0 MiB"},
		{1<<30 - 1, "1024 MiB"},
		{1 << 30, "1.0 GiB"},
		{1<<30 + 1, "1.0 GiB"},
		{1<<40 - 1, "1024 GiB"},
		{1 << 40, "1.0 TiB"},
		{1<<40 + 1, "1.0 TiB"},
		{1<<50 - 1, "1024 TiB"},
		{1 << 50, "1.0 PiB"},
		{1<<50 + 1, "1.0 PiB"},
		{1<<60 - 1, "1024 PiB"},
		{1 << 60, "1.0 EiB"},
		{1<<60 + 1, "1.0 EiB"},
		{math.MaxUint64, "16 EiB"},
	} {
		t.Run(strconv.FormatUint(test.size, 10), func(t *testing.T) {
			if got := IBytes(test.size); got != test.want {
				t.Fatalf("IBytes(%d) = %q, want %q", test.size, got, test.want)
			}
		})
	}
}
