package no_cast

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
	pink := PinkStruct{} // ok
	violet := VioletStruct{} // ok

	_ = pink // ok
	_ = violet // ok
}