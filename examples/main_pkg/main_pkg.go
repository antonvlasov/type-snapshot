package mainpkg

import (
	"github.com/antonvlasov/type-snapshot/examples/elements"
	"github.com/antonvlasov/type-snapshot/examples/pkga"
	"github.com/antonvlasov/type-snapshot/examples/pkgb"
)

type TopStruct struct {
	Field1 string            `json:"field1"`
	Field2 elements.Element1 `json:"field2"`
	Field3 elements.Element2 `json:"field3"`
}

type MainStruct struct {
	Field1 pkga.A
	Field2 pkgb.A
	field3 pkgb.B
	field4 A
}

type A struct {
	value string
	rec   Recursive
}

type Recursive struct {
	normalField string
	recField    *Recursive
}
