// +build ignore

package main

import (
	"log"

	B "github.com/mmcloughlin/avo/build"
	O "github.com/mmcloughlin/avo/operand"
)

func main() {
	B.TEXT("cmp4", B.NOSPLIT, "func(fours [4]uint64, pk [4]uint64) int16")
	twosParam := B.Param("fours").Index(0)
	pkParam := B.Param("pk").Index(0)

	scratch := B.AllocLocal(4 * 8)
	scratch1 := scratch.Offset(8)
	scratch2 := scratch.Offset(16)
	scratch3 := scratch.Offset(24)

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

	x0 := B.YMM()
	x1 := B.YMM()
	x2 := B.YMM()

	one := B.GP64()
	B.MOVQ(O.U64(18446744073709551615), one)

	//bs := B.GP64()
	//B.MOVQ(O.U64(23), bs)

	B.Comment("Move twos and pk to xmm")
	B.VMOVUPD(twos.Addr, x0)
	B.VMOVUPD(pk.Addr, x1)

	B.Comment("Check Eq")
	B.VPCMPEQQ(x0, x1, x2)

	B.Comment("Move results out")
	B.VMOVDQU(x2, scratch)

	/*
		// debug
		ret1 := B.GP64()
		ret2 := B.GP64()
		ret3 := B.GP64()
		ret4 := B.GP64()
		B.MOVQ(scratch, ret1)
		B.MOVQ(scratch1, ret2)
		B.MOVQ(scratch2, ret3)
		B.MOVQ(scratch3, ret4)
		B.Store(ret1, B.ReturnIndex(0))
		B.Store(ret2, B.ReturnIndex(1))
		B.Store(ret3, B.ReturnIndex(2))
		B.Store(ret4, B.ReturnIndex(3))
		B.RET()
	*/

	B.CMPQ(one, scratch)
	B.JE(O.LabelRef("zeroth"))

	B.CMPQ(one, scratch1)
	B.JE(O.LabelRef("fst"))

	B.CMPQ(one, scratch2)
	B.JE(O.LabelRef("snd"))

	B.CMPQ(one, scratch3)
	B.JE(O.LabelRef("thd"))

	//B.Store(bs, B.ReturnIndex(0))
	B.MOVW(O.U16(4), ret.Addr)
	//B.MOVW(O.I16(-1), retReg)
	//B.Store(retReg, B.ReturnIndex(0))
	B.RET()

	B.Label("zeroth")
	//B.Store(ret1, B.ReturnIndex(0))
	B.MOVW(O.U16(0), ret.Addr)
	//B.MOVW(O.I16(0), retReg)
	//B.Store(retReg, B.ReturnIndex(0))
	B.RET()

	B.Label("fst")
	//B.Store(ret2, B.ReturnIndex(0))
	B.MOVW(O.U16(1), ret.Addr)
	//B.MOVW(O.I16(1), retReg)
	//B.Store(retReg, B.ReturnIndex(0))
	B.RET()

	B.Label("snd")
	B.MOVW(O.U16(2), ret.Addr)
	B.RET()

	B.Label("thd")
	B.MOVW(O.U16(3), ret.Addr)
	B.RET()

	B.Generate()
}
