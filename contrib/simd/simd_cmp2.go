// +build ignore

package main

import (
	"log"

	B "github.com/mmcloughlin/avo/build"
	O "github.com/mmcloughlin/avo/operand"
)

func main() {
	B.TEXT("cmp2", B.NOSPLIT, "func(twos [2]uint64, pk [2]uint64) int16")
	twosParam := B.Param("twos").Index(0)
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

	x0 := B.XMM()
	x1 := B.XMM()
	x2 := B.XMM()

	one := B.GP64()
	B.MOVQ(O.U64(18446744073709551615), one) // -1

	//bs := B.GP64()
	//B.MOVQ(O.U64(23), bs)

	B.Comment("Move twos and pk to xmm")
	B.MOVUPS(twos.Addr, x0)
	B.MOVUPS(pk.Addr, x1)

	B.Comment("Check Eq")
	B.VPCMPEQQ(x0, x1, x2)

	B.Comment("Move results out")
	ret1 := B.GP64()
	ret2 := B.GP64()
	B.MOVQ(x2, ret1)
	B.PUNPCKHQDQ(x2, x2) // move high bits to low bits because MOVQ can only move low bits
	B.MOVQ(x2, ret2)

	//B.Store(ret1, B.ReturnIndex(0))
	//B.Store(ret2, B.ReturnIndex(1))
	//B.RET()

	B.CMPQ(one, ret1)
	B.JE(O.LabelRef("zeroth"))

	B.CMPQ(one, ret2)
	B.JE(O.LabelRef("fst"))

	//B.Store(bs, B.ReturnIndex(0))
	B.MOVW(O.U16(2), ret.Addr)
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

	B.Generate()
}
