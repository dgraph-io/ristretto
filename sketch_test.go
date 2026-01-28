/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package ristretto

import (
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSketch(t *testing.T) {
	defer func() {
		require.NotNil(t, recover())
	}()

	s := newCmSketch(5)
	require.Equal(t, uint64(7), s.mask)
	newCmSketch(0)
}

// Temporary comment. This test is fixed, kind of.
// It will now report when two or more rows are identical but that isn't what
// should be tested. What should be tested is that the rows end up having unique
// values from each other. If the rows have identical counters, but they are in
// different positions, that doesn't help; no reason to have multiple rows if
// the increment for one row always changes counters like to does for the next
// row. So the actual test that shows whether row counters are being incremented
// uniquely is the new one below: TestSketchRowUniqueness. There the counters in
// a row are sorted before being compared.
func TestSketchIncrement(t *testing.T) {
	s := newCmSketch(16)
	pseudoRandomIncrements(s, anyseed, anycount)
	for i := 0; i < cmDepth; i++ {
		rowi := s.rows[i].string()
		for j := 0; j < i; j++ {
			rowj := s.rows[j].string()
			require.False(t, rowi == rowj, "identical rows, bad hashing")
		}
	}
}

const anyseed = int64(990099)
const anycount = int(100) // Hopefully not large enough to max out a counter

func pseudoRandomIncrements(s *cmSketch, seed int64, count int) {
	r := rand.New(rand.NewSource(anyseed))
	for n := 0; n < count; n++ {
		s.Increment(r.Uint64())
	}
}

// Bad hashing increments because there is very little entropy
// between the values. This is used to test how well the sketch
// uses multiple rows when there is little entropy between the hashes
// being incremented with.
func badHashingIncrements(s *cmSketch, count int) {
	for hashed := 0; hashed < count; hashed++ {
		s.Increment(uint64(hashed + 1))
	}
}

func TestSketchRowUniqueness(t *testing.T) {
	// Test the row uniqueness twice.
	// Once when the hashes being added are pretty random
	// which we would normally expect.
	// And once when the hashes added are not normal, maybe
	// they are all low numbers for example.
	t.Run("WithGoodHashing", func(t *testing.T) {
		s := newCmSketch(16)
		pseudoRandomIncrements(s, anyseed, anycount)
		testSketchRowUniqueness1(t, s)
	})
	t.Run("WithBadHashing", func(t *testing.T) {
		s := newCmSketch(16)
		badHashingIncrements(s, anycount)
		testSketchRowUniqueness1(t, s)
	})
}

func testSketchRowUniqueness1(t *testing.T, s *cmSketch) {
	// Perform test like the one above, TestSketchIncrement, but
	// compare the rows counters, once the counters are sorted.
	// If after several insertions, the rows have the same counters
	// in them, the hashing scheme is likely not actually
	// creating uniqueness between rows.

	var unswiv [cmDepth]string
	for i := 0; i < cmDepth; i++ {
		unswiv[i] = s.rows[i].string()
	}
	// Now perform a kind of unswivel on each row, so counters are in ascending order.
	for i := 0; i < cmDepth; i++ {
		fields := strings.Split(unswiv[i], " ")
		sort.Strings(fields)
		unswiv[i] = strings.Join(fields, " ")
	}
	identical := 0
	for i := 0; i < cmDepth; i++ {
		t.Logf("s.rows[%d] %s, unswiv[%d] %s", i, s.rows[i].string(), i, unswiv[i])

		for j := 0; j < i; j++ {
			if unswiv[i] == unswiv[j] {
				// Even one would be suspect. But count here so we can see how many rows look the same.
				identical++
				break // break out of j loop, knowing i looks like any earlier is enough, don't want to double count
			}
		}
	}
	require.Zero(t, identical, "%d unswiveled rows looked like earlier unswiveled rows", identical)
}

func TestSketchEstimate(t *testing.T) {
	s := newCmSketch(16)
	s.Increment(1)
	s.Increment(1)
	require.Equal(t, int64(2), s.Estimate(1))
	require.Equal(t, int64(0), s.Estimate(0))
}

func TestSketchReset(t *testing.T) {
	s := newCmSketch(16)
	s.Increment(1)
	s.Increment(1)
	s.Increment(1)
	s.Increment(1)
	s.Reset()
	require.Equal(t, int64(2), s.Estimate(1))
}

func TestSketchClear(t *testing.T) {
	s := newCmSketch(16)
	for i := 0; i < 16; i++ {
		s.Increment(uint64(i))
	}
	s.Clear()
	for i := 0; i < 16; i++ {
		require.Equal(t, int64(0), s.Estimate(uint64(i)))
	}
}

func TestNext2Power(t *testing.T) {
	sz := int64(12) << 30
	szf := float64(sz) * 0.01
	val := int64(szf)
	t.Logf("szf = %.2f val = %d\n", szf, val)
	pow := next2Power(val)
	t.Logf("pow = %d. mult 4 = %d\n", pow, pow*4)
}

func BenchmarkSketchIncrement(b *testing.B) {
	s := newCmSketch(16)
	b.SetBytes(1)
	for n := 0; n < b.N; n++ {
		s.Increment(1)
	}
}

func BenchmarkSketchEstimate(b *testing.B) {
	s := newCmSketch(16)
	s.Increment(1)
	b.SetBytes(1)
	for n := 0; n < b.N; n++ {
		s.Estimate(1)
	}
}
