package safe_cast_dereferenced_header

import (
	"reflect"
	"runtime"
	"unsafe"
)

func SafeCastString(str string) (b []byte) {
	strH := *(*reflect.StringHeader)(unsafe.Pointer(&str))
	sH := *(*reflect.SliceHeader)(unsafe.Pointer(&b))
	sH.Len = strH.Len // ok
	sH.Cap = strH.Len // ok
	sH.Data = strH.Data // ok
	runtime.KeepAlive(str)
	return
}

