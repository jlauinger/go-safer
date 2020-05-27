package type_alias

import (
	"reflect"
	"unsafe"
)

type Header reflect.SliceHeader

func LiteralDefinition(s string) {
	strH := (*reflect.StringHeader)(unsafe.Pointer(&s))
	sH := &Header{ // want "reflect header composite literal found" "reflect header composite literal found"
		Data: strH.Data,
		Len:  strH.Len,
		Cap:  strH.Len,
	}
	_ = sH
}
