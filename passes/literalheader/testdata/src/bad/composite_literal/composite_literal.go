package composite_literal

import (
	"reflect"
	"unsafe"
)

func UnsafeCastString(str string) []byte {
	strH := (*reflect.StringHeader)(unsafe.Pointer(&str))
	sH := &reflect.SliceHeader{ // want "reflect header composite literal found" "reflect header composite literal found"
		Data: strH.Data,
		Cap: strH.Len,
		Len: strH.Len,
	}
	return *(*[]byte)(unsafe.Pointer(sH))
}
