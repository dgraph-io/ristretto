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

	Comment("Base")
	ptr := Load(Param("xs").Base(), GP64())

	Comment("n")
	n := Load(Param("xs").Len(), GP64())

	Comment("Key")
	key := Load(Param("k"), GP64())
	si := GP64()
	tmp := GP16()

	Comment("Initialize idx register to zero.")
	idx := GP32()
	XORL(idx, idx)
	JMP(LabelRef("check"))

	Label("plusplus")
	INCL(idx)

	Label("check")
	Comment("REPLACE THE FOLLOWING TWO INSTRUCTIONS:  CMPQ idx n ")
	INCQ(n)
	DECQ(n)
	JGE(LabelRef("NotFound"))

	Label("loop")
	MOVQ(Mem{Base: ptr, Index: idx, Scale: 8}, si)
	NOP()
	CMPQ(si, key)
	JCS(LabelRef("plusplus"))
	Comment("Replace instructions with MOVW, tmp with idx")
	Store(tmp, ReturnIndex(0))
	RET()

	Label("NotFound")
	Comment("Replace instructions with MOVW, tmp with $-1")
	Store(tmp, ReturnIndex(0))
	RET()
	Generate()
}
