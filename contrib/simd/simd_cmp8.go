// +build ignore

package main

import (
	"log"

	B "github.com/mmcloughlin/avo/build"
	O "github.com/mmcloughlin/avo/operand"
)

func main() {
	B.TEXT("cmp8", B.NOSPLIT, "func(a [8]uint64, pk [4]uint64) int16")
	twosParam := B.Param("a").Index(0)
	pkParam := B.Param("pk").Index(0)

	twos, err := twosParam.Resolve()
	if err != nil {
		panic(err)
	}

	pk, err := pkParam.Resolve()
	if err != nil {
		panic(err)
	}

	ret, err := B.ReturnIndex(0).Resolve()
	if err != nil {
		panic(err)
	}
	log.Printf("%v | %#v", ret.Addr.Asm(), ret.Addr.Base.Asm())
	retReg := B.GP16()
	log.Printf("%v", retReg)

	// set 1
	x0 := B.YMM()
	x1 := B.YMM()
	x2 := B.YMM()
	x3 := B.YMM()

	// set2
	x4 := B.YMM()
	x5 := B.YMM()
	x6 := B.YMM()

	one := B.GP64()
	B.MOVQ(O.U64(18446744073709551615), one)

	//bs := B.GP32()
	//B.MOVL(O.U32(23), bs)

	B.Comment("Move twos and pk to xmm")
	B.VMOVUPD(twos.Addr, x0)
	B.VMOVUPD(pk.Addr, x1)
	B.VMOVUPD(twos.Addr.Offset(32), x4)

	B.Comment("Check Eq")
	B.VPCMPEQQ(x0, x1, x2)
	B.VPCMPGTQ(x1, x0, x3)
	B.VPADDQ(x2, x3, x3)

	B.VPCMPEQQ(x4, x1, x5)
	B.VPCMPGTQ(x1, x4, x6)
	B.VPADDQ(x6, x5, x6)

	B.Comment("Move results out")
	XX := B.GP32()
	YY := B.GP32()
	four := B.GP32()
	B.MOVL(O.U32(4), four)
	B.VMOVMSKPD(x3, XX)
	B.VMOVMSKPD(x6, YY)

	B.TZCNTL(XX, XX)
	B.CMPL(XX, four)
	B.JLE(O.LabelRef("XX"))
	B.TZCNTL(YY, YY)
	B.CMPL(YY, four)
	B.JLE(O.LabelRef("YY"))

	B.Label("XX")
	B.MOVL(XX, ret.Addr)
	B.RET()

	B.Label("YY")
	B.ADDL(four, YY)
	B.MOVL(YY, ret.Addr)
	B.RET()

	B.Generate()
}
