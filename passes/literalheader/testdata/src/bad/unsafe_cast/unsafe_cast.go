package unsafe_cast

import (
	"reflect"
	"runtime"
	"unsafe"
)

func SaferCastString(str string) (b []byte) {
	strH := (*reflect.StringHeader)(unsafe.Pointer(&str))
	sH := (*reflect.SliceHeader)(nil)
	sH.Len = strH.Len // want "assigning to reflect header object" "assigning to reflect header object"
	sH.Cap = strH.Len // want "assigning to reflect header object" "assigning to reflect header object"
	sH.Data = strH.Data // want "assigning to reflect header object" "assigning to reflect header object"
	runtime.KeepAlive(str)
	return
}