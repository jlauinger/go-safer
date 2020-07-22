package strictly_sized_struct

import "unsafe"

type PinkStruct struct  {
	A uint8
	B int64
	C int64
}

type VioletStruct struct  {
	A uint8
	B int64
	C int64
}

func UnsafeCast() {
	pink := PinkStruct{
		A: 1,
		B: 42,
		C: 9000,
	}

	violet := *(*VioletStruct)(unsafe.Pointer(&pink)) // ok

	_ = violet // ok
}