package mainpkg

import (
	"github.com/antonvlasov/type-snapshot/examples/pkga"
	"github.com/antonvlasov/type-snapshot/examples/pkgb"
)

type MainStruct struct {
	Field1 pkga.A
	Field2 pkgb.A
	field3 pkgb.B
	field4 A
}

type A struct {
	value string
}
