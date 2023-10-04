package generator

import "reflect"

const (
	generatorPkg = "github.com/antonvlasov/type-snapshot"
)

var typesOrder = []reflect.Kind{
	reflect.Chan,
	reflect.Array,
	reflect.Slice,
	reflect.Map,
	reflect.Func,
	reflect.Interface,
	reflect.Struct,
	reflect.Invalid, // for other types
}
