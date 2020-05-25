package composite_in_composite

import (
	"reflect"
	"unsafe"
)

type Foo struct {
	Bar *reflect.SliceHeader
}

func UnsafeCastString(str string) []byte {
	strH := (*reflect.StringHeader)(unsafe.Pointer(&str))
	foo := Foo {
		Bar: &reflect.SliceHeader{ // want "reflect header composite literal found" "reflect header composite literal found"
			Data: strH.Data,
			Cap: strH.Len,
			Len: strH.Len,
		},
	}
	return *(*[]byte)(unsafe.Pointer(foo.Bar))
}
