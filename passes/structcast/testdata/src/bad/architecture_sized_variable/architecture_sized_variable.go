package architecture_sized_variable

import "unsafe"

type PinkStruct struct  {
	A uint8
	B int
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

	violet := *(*VioletStruct)(unsafe.Pointer(&pink)) // want "unsafe cast between structs with mismatching count of platform dependent field sizes"

	_ = violet // ok
}