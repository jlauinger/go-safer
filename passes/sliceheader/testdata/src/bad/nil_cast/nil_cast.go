package nil_cast

import (
	"reflect"
	"runtime"
	"unsafe"
)

func SaferCastString(str string) (b []byte) {
	strH := (*reflect.StringHeader)(unsafe.Pointer(&str))
	sH := (*reflect.SliceHeader)(unsafe.Pointer(nil))
	sH.Len = strH.Len // want "assigning to incorrectly derived reflect header object" "assigning to incorrectly derived reflect header object"
	sH.Cap = strH.Len // want "assigning to incorrectly derived reflect header object" "assigning to incorrectly derived reflect header object"
	sH.Data = strH.Data // want "assigning to incorrectly derived reflect header object" "assigning to incorrectly derived reflect header object"
	runtime.KeepAlive(str)
	return
}
