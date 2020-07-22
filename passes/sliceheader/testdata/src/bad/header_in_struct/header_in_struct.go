package bad

import (
	"reflect"
	"unsafe"
)

type Protocol struct {
	Foo int
	Sh *reflect.SliceHeader
}

func UnsafeStringIntoProtocol(str string) []byte {
	strH := (*reflect.StringHeader)(unsafe.Pointer(&str))
	protocol := Protocol{}
	protocol.Sh.Len = strH.Len // want "assigning to incorrectly derived reflect header object" "assigning to incorrectly derived reflect header object"
	protocol.Sh.Cap = strH.Len // want "assigning to incorrectly derived reflect header object" "assigning to incorrectly derived reflect header object"
	protocol.Sh.Data = strH.Data // want "assigning to incorrectly derived reflect header object" "assigning to incorrectly derived reflect header object"
	return *(*[]byte)(unsafe.Pointer(protocol.Sh))
}
