package simd

// Search finds the key using the naive way
func Naive(xs []uint64, k uint64) int16 {
	var i int
	for i = 0; i < len(xs); i += 2 {
		x := xs[i]
		if x >= k {
			return int16(i / 2)
		}
	}
	return int16(i / 2)
}

func Clever(xs []uint64, k uint64) int16 {
	var twos, pk, eq [2]uint64
	pk[0] = k
	pk[1] = k
	for i := 0; i < len(xs); i += 4 {
		twos[0] = xs[i]
		twos[1] = xs[i+2]

		// ALL THESE WILL BE VECTORIZED
		if twos[0] == pk[0] {
			eq[0] = 1
		}
		if twos[1] == pk[1] {
			eq[1] = 1
		}
		if eq[0] == 1 {
			return int16(i)
		}
		if eq[1] == 1 {
			return int16(i + 2)
		}
		if twos[0] > pk[0] {
			eq[0] += 1
		}
		if twos[1] > pk[1] {
			eq[1] += 1
		}
		if eq[0] == 1 {
			return int16(i)
		}
		if eq[1] == 1 {
			return int16(i + 2)
		}

	}
	return -1
}

func cmp2_native(twos, pk [2]uint64) int16 {
	if twos[0] == pk[0] {
		return 0
	}
	if twos[1] == pk[1] {
		return 1
	}
	return 2
}

func cmp4_native(fours, pk [4]uint64) int16 {
	for i := range fours {
		if fours[i] == pk[i] {
			return int16(i)
		}
	}
	return 4
}
