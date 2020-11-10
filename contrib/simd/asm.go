// +build ignore

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

//go:generate go run asm.go -out add.s -stubs stub.go

func main() {
	TEXT("Search", NOSPLIT, "func(xs []uint64, k uint64) int16")
	Doc("Search finds the first key >= k in xs.")
	ptr := Load(Param("xs").Base(), GP64())
	n := Load(Param("xs").Len(), GP64())
	key := Load(Param("k"), GP64())

	Comment("Initialize idx register to zero.")
	idx := GP16()
	XORW(idx, idx)

	Label("loop")
	Comment("Loop until zero bytes remain.")
	CMPQ(n, Imm(0))
	JLE(LabelRef("done"))

	Comment("Load from pointer and add to running sum.")
	CMPQ(Mem{Base: ptr}, key)
	JGE(LabelRef("done"))

	Comment("Advance pointer, decrement byte count.")
	ADDQ(Imm(16), ptr)
	DECQ(n)
	DECQ(n)
	ADDW(Imm(1), idx)
	JMP(LabelRef("loop"))

	Label("done")
	Comment("Store sum to return value.")
	Store(idx, ReturnIndex(0))
	RET()
	Generate()
}

// func main() {
// 	TEXT("Sum", NOSPLIT, "func(xs []uint64) uint64")
// 	Doc("Sum returns the sum of the elements in xs.")
// 	ptr := Load(Param("xs").Base(), GP64())
// 	n := Load(Param("xs").Len(), GP64())

// 	Comment("Initialize sum register to zero.")
// 	s := GP64()
// 	XORQ(s, s)

// 	Label("loop")
// 	Comment("Loop until zero bytes remain.")
// 	CMPQ(n, Imm(0))
// 	JE(LabelRef("done"))

// 	Comment("Load from pointer and add to running sum.")
// 	ADDQ(Mem{Base: ptr}, s)

// 	Comment("Advance pointer, decrement byte count.")
// 	ADDQ(Imm(8), ptr)
// 	DECQ(n)
// 	JMP(LabelRef("loop"))

// 	Label("done")
// 	Comment("Store sum to return value.")
// 	Store(s, ReturnIndex(0))
// 	RET()
// 	Generate()
// }
