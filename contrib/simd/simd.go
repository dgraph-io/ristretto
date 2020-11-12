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

	B.Comment("n")
	n := B.Load(B.Param("xs").Len(), B.GP64())

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
	x1 := B.YMM()
	x2 := B.YMM()
	x3 := B.YMM()
	x4 := B.YMM()
	XX := B.GP32()

	// xs[0] to xs[6] uses this
	tmpXs := B.GP64()

	// results
	tmp := B.GP64()

	B.Comment("load const 4 into a register; load n as max")
	four := B.GP32()
	max := B.GP32()
	B.MOVL(O.U32(4), four)
	B.MOVQ(n, max)

	//B.VPBROADCASTQ(key, pk)

	B.Comment("i := 0")
	i := B.GP64()
	B.XORQ(i, i)
	B.NOP()
	B.JMP(O.LabelRef("loop"))

	B.Label("plusplus")
	B.Comment("i+=8")
	B.ADDQ(O.Imm(56), i)

	B.Comment("For loop starts")
	B.Label("loop")
	B.CMPQ(i, n)
	B.JGE(O.LabelRef("NotFound"))

	B.Comment("Copy 4 keys into packed")
	mem := O.Mem{Base: ptr, Index: i, Scale: 8} // (ptr)(i*8)
	mem1 := mem.Offset(16)                      // skip 2 - (ptr)((i+2)*8)
	mem2 := mem.Offset(32)                      // skip 4 - (ptr)((i+4)*8)
	mem3 := mem.Offset(48)                      // skip 6 - (ptr)((i+6)*8)

	B.MOVQ(mem, tmpXs)
	B.MOVQ(tmpXs, packed)
	B.MOVQ(mem1, tmpXs)
	B.MOVQ(tmpXs, packed1)
	B.MOVQ(mem2, tmpXs)
	B.MOVQ(tmpXs, packed2)
	B.MOVQ(mem3, tmpXs)
	B.MOVQ(tmpXs, packed3)

	B.Comment("Move the packed keys into ymm; move key into pk")
	B.VMOVUPD(mem, x0)
	B.VPBROADCASTQ(key.Addr, pk)

	B.Comment("Check GTE")
	B.VPCMPEQQ(x0, x1, x2)
	B.VPCMPGTQ(x1, x0, x3)
	B.VPADDQ(x2, x3, x4)

	B.Comment("Move result out")
	B.VMOVMSKPD(x3, XX)

	B.Comment("Count trailing zeroes")
	B.TZCNTL(XX, XX)

	B.Comment("update max if tz < max")
	B.CMOVLLS(XX, max)

	B.Label("NotFound: return n/2")
	B.MOVQ(n, tmp)
	B.SHRQ(O.Imm(63), n)
	B.ADDQ(tmp, n)
	B.SARQ(O.Imm(1), n)
	B.Store(n, B.ReturnIndex(0))
	B.RET()

	B.Generate()

}
