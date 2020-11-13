// +build ignore

package main

import (
	B "github.com/mmcloughlin/avo/build"
	O "github.com/mmcloughlin/avo/operand"
)

//go:generate go run sse.go -out sse.s -stubs stub_sse.go

func main() {
	B.TEXT("SSESearch", B.NOSPLIT, "func(xs []uint64, key uint64) int16")
	packed := B.AllocLocal(4 * 8) // 4x 8bytes. Each uint64 has 8bytes
	packed1 := packed.Offset(8)   // packed[1]

	/*
		packedB := B.AllocLocal(4 * 8) // 4x 8bytes. Each uint64 has 8bytes
		packedB1 := packed.Offset(8)   // packedB[1]
		packedB2 := packed.Offset(16)  //packedB[2]
		packedB3 := packed.Offset(24)  //packedB[3]
	*/
	retInd := B.ReturnIndex(0)
	retVal, err := retInd.Resolve()
	if err != nil {
		panic(err)
	}

	B.Comment("n")
	n := B.GP32()
	length, err := B.Param("xs").Len().Resolve() // this bit is needed to move to a GP32
	if err != nil {
		panic(err)
	}
	B.MOVL(length.Addr, n)

	B.Comment("xs[0]")
	ptr := B.Load(B.Param("xs").Base(), B.GP64())

	B.Comment("key")
	key, err := B.Param("key").Resolve()
	if err != nil {
		panic(err)
	}

	// might have to move this closer to where all the other VPXXX instructions are made
	//B.Comment("Copy key into ymm")
	pk := B.XMM()
	x0 := B.XMM()
	x1 := B.XMM()
	x2 := B.XMM()
	x3 := B.XMM()

	res1 := B.GP32()
	res2 := B.GP32()

	one := B.GP32()
	B.MOVL(O.U32(4294967295), one) // -1

	XX := B.GP32()
	//YY := B.GP32()

	B.VPBROADCASTQ(key.Addr, pk)

	// xs[0] to xs[6] uses this
	tmpXs := B.GP64()

	// results
	tmp := B.GP64()

	B.Comment("load const 4 into a register; load n as max")
	four := B.GP32()
	max := B.GP32()
	B.MOVL(O.U32(4), four)
	B.MOVL(length.Addr, max)

	B.Comment("i := 0")
	i := B.GP32()
	B.XORL(i, i)
	B.NOP()
	B.JMP(O.LabelRef("loop"))

	B.Label("plusplus")
	B.Comment("i+=4")
	B.ADDL(O.Imm(4), i)

	B.Comment("For loop starts")
	B.Label("loop")
	B.CMPL(i, n)
	B.JGE(O.LabelRef("NotFound"))

	B.Comment("Copy 4 keys into packed")
	mem := O.Mem{Base: ptr, Index: i, Scale: 8} // (ptr)(i*8)
	mem1 := mem.Offset(16)                      // skip 2 - (ptr)((i+2)*8)
	/*
		memB := mem.Offset(64) // skip 8 - (ptr)((i+8)*8)
		memB1 := mem.Offset(80)
		memB2 := mem.Offset(96)
		memB3 := mem.Offset(112)
	*/

	B.MOVQ(mem, tmpXs)
	B.MOVQ(tmpXs, packed)
	B.MOVQ(mem1, tmpXs)
	B.MOVQ(tmpXs, packed1)

	B.Comment("Move the packed keys into ymm; move key into pk")
	B.MOVUPS(packed, x0)

	B.Comment("Check GTE")
	B.VPCMPEQQ(x0, pk, x1)
	B.VPCMPGTQ(pk, x0, x2)
	B.VPADDQ(x1, x2, x3)

	B.Comment("Move result out")

	B.MOVQ(x3, res1)
	B.PUNPCKHQDQ(x3, x3) // move high bits to low bits because MOVQ can only move low bits
	B.MOVQ(x3, res2)

	B.CMPL(one, res1)
	B.JE(O.LabelRef("FoundFst"))

	B.CMPL(one, res2)
	B.JNE(O.LabelRef("plusplus"))
	B.MOVL(O.U32(1), XX)
	B.JMP(O.LabelRef("Found"))

	B.Label("FoundFst")
	B.MOVL(O.U32(0), XX)

	B.Label("Found")
	B.Comment("we've found the results in fst")
	B.Comment("2*xx + i")
	B.SHLL(O.Imm(1), XX) // xx *=2
	B.ADDL(i, XX)
	B.Comment("div by 2")
	B.MOVL(XX, i) // i is no longer needed
	B.SHRL(O.Imm(31), i)
	B.ADDL(i, XX)
	B.SARL(O.Imm(1), XX)
	B.MOVL(XX, retVal.Addr)
	B.RET()

	B.Label("NotFound")
	B.Comment("Load len as a 64 bit number")
	n64 := B.GP64()
	B.MOVQ(length.Addr, n64)
	B.Comment("return n/2")
	B.MOVQ(n64, tmp)
	B.SHRQ(O.Imm(63), n64)
	B.ADDQ(tmp, n64)
	B.SARQ(O.Imm(1), n64)
	B.MOVQ(n64, retVal.Addr)
	B.RET()

	B.Generate()

}
