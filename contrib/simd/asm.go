// +build ignore

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

//go:generate go run asm.go -out add.s -stubs stub.go

func main() {
	TEXT("Search", NOSPLIT, "func(xs []uint64, k uint64) int16")
	Doc("Search finds the first idx for which xs[idx] >= k in xs.")
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
	JAE(LabelRef("done")) // Use JAE to compare unsigned. JGE uses signed.

	Comment("Advance pointer, decrement byte count.")
	ADDQ(Imm(16), ptr)
	SUBQ(Imm(2), n)
	ADDW(Imm(1), idx)
	JMP(LabelRef("loop"))

	Label("done")
	Comment("Store sum to return value.")
	Store(idx, ReturnIndex(0))
	RET()
	Generate()
}
