package good

import (
	"reflect"
	"runtime"
	"unsafe"
)

func SaferCastString(str string) (b []byte) {
	strH := (*reflect.StringHeader)(unsafe.Pointer(&str))
	sH := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sH.Len = strH.Len // ok
	sH.Cap = strH.Len // ok
	sH.Data = strH.Data // ok
	runtime.KeepAlive(str)
	return
}
