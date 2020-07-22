package bad

import (
	"reflect"
	"unsafe"
)

func AlsoUnsafeCastString(str string) []byte {
	strH := (*reflect.StringHeader)(unsafe.Pointer(&str))
	var sH *reflect.SliceHeader
	sH.Len = strH.Len // want "assigning to incorrectly derived reflect header object" "assigning to incorrectly derived reflect header object"
	sH.Cap = strH.Len // want "assigning to incorrectly derived reflect header object" "assigning to incorrectly derived reflect header object"
	sH.Data = strH.Data // want "assigning to incorrectly derived reflect header object" "assigning to incorrectly derived reflect header object"
	return *(*[]byte)(unsafe.Pointer(sH))
}

