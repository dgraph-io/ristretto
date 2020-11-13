// +build ignore

package main

import (
	B "github.com/mmcloughlin/avo/build"
	O "github.com/mmcloughlin/avo/operand"
)

func main() {
	B.TEXT("skernel", B.NOSPLIT, "func(xs []uint64, key uint64) int16")
	packed := B.AllocLocal(4 * 8) // 4x 8bytes. Each uint64 has 8bytes
	packed1 := packed.Offset(8)   // packed[1]
	packed2 := packed.Offset(16)  //packed[2]
	packed3 := packed.Offset(24)  //packed[3]

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
	pk := B.YMM()
	x0 := B.YMM()
	x2 := B.YMM()
	x3 := B.YMM()
	x4 := B.YMM()

	/*
		x5 := B.YMM()
		x6 := B.YMM()
		x7 := B.YMM()
		x8 := B.YMM()
	*/
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
	B.Comment("i+=8")
	B.ADDL(O.Imm(8), i)

	B.Comment("For loop starts")
	B.Label("loop")
	B.CMPL(i, n)
	B.JGE(O.LabelRef("NotFound"))

	B.Comment("Copy 4 keys into packed")
	mem := O.Mem{Base: ptr, Index: i, Scale: 8} // (ptr)(i*8)
	mem1 := mem.Offset(16)                      // skip 2 - (ptr)((i+2)*8)
	mem2 := mem.Offset(32)                      // skip 4 - (ptr)((i+4)*8)
	mem3 := mem.Offset(48)                      // skip 6 - (ptr)((i+6)*8)

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
	B.MOVQ(mem2, tmpXs)
	B.MOVQ(tmpXs, packed2)
	B.MOVQ(mem3, tmpXs)
	B.MOVQ(tmpXs, packed3)
	/*
		B.MOVQ(memB, tmpXs)
		B.MOVQ(tmpXs, packedB)
		B.MOVQ(memB1, tmpXs)
		B.MOVQ(tmpXs, packedB1)
		B.MOVQ(memB2, tmpXs)
		B.MOVQ(tmpXs, packedB2)
		B.MOVQ(memB3, tmpXs)
		B.MOVQ(tmpXs, packedB3)
	*/

	B.Comment("Move the packed keys into ymm; move key into pk")
	B.VMOVUPD(packed, x0)
	//B.VMOVUPD(packedB, x5)
	//B.VPBROADCASTQ(key.Addr, pk)

	B.Comment("Check GTE")
	B.VPCMPEQQ(x0, pk, x2)
	B.VPCMPGTQ(pk, x0, x3)
	B.VPADDQ(x2, x3, x4)

	//B.VPCMPEQQ(x5, pk, x6)
	//B.VPCMPGTW(pk, x5, x7)
	//B.VPADDQ(x6, x7, x8)

	B.Comment("Move result out")
	B.VMOVMSKPD(x4, XX)
	//B.VMOVMSKPD(x8, YY)

	B.Comment("Count trailing zeroes XX")
	B.TZCNTL(XX, XX)

	B.Comment("if tz < 4 we got a result")
	B.CMPL(XX, four)
	//B.JLE(O.LabelRef("FoundXX"))
	B.JGE(O.LabelRef("plusplus"))

	//B.Comment("Count trailing zeroes YY")
	//B.TZCNTL(YY, YY)
	//B.CMPL(YY, four)
	//B.JGE(O.LabelRef("plusplus"))

	/*
		B.Comment("weve found the results in YY")
		B.Comment("2*(tz+4)+i")
		B.ADDL(four, YY)
		B.SHLL(O.Imm(1), YY)
		B.ADDL(i, YY)
		B.Comment("div by 2")
		B.MOVL(YY, i) // i is no longer needed
		B.SHRL(O.Imm(31), i)
		B.ADDL(i, YY)
		B.SARL(O.Imm(1), YY)
		B.MOVL(YY, retVal.Addr)
		B.RET()
	*/

	B.Label("FoundXX")
	B.Comment("we've found the results in XX")
	B.Comment("2*tz + i")
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
