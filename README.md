# type-snapshot
This library is a code generator for go that allows to take a snapshot of type definition in a particular moment of project development. 
Type-snapshot creates types with same fields, same tags and similar names without depending on existing modules.
### Install
```
go install github.com/antonvlasov/type-snapshot/...@latest
```
# Example
### One source type
Given two input files:

```go
package mainpkg

import (
	"project/elements"
)

type TopStruct struct {
  Field1 string `json:"field1"`
  Field2 elements.Element1 `json:"field2"`
  Field3 elements.Element2 `json:"field3"`
}
```
```go
package elements

type Element1 struct {
  ElementField1 int `json:"int"`
}

type Element2 map[string]*[]int
```
We can snapshot TopStruct and write it to a new file:
```go
package copy

// These types were autogenerated by snapshotting existing types using github.com/antonvlasov/type-snapshot/pkg/generator
type (
	Element2Old map[string]*[]int

	Element1Old struct {
		ElementField1 int `json:"int"`
	}

	TopStructOld struct {
		Field1 string      `json:"field1"`
		Field2 Element1Old `json:"field2"`
		Field3 Element2Old `json:"field3"`
	}
)
```
To achieve this the following command was used:
```bash
type_snapshot -paths "examples/main_pkg.TopStruct" -dst ~/simple_copy/copy/copy.go -suffix Old
```
### Multiple source types
It is also possible to snapshot multiple types from different modules and output them to the same file.
The following command was used to produce content of [copy.go](examples/copy/copy.go) from multiple types defined in [examples](examples):
```bash
type_snapshot -paths "examples/main_pkg.MainStruct examples/main_pkg.A examples/pkgb.UnUsedA examples/pkgb.UnUsedB" -dst examples/copy/copy.go -suffix Old
```
### Motivation
It is often required to copy struct definition manually when writing a database migration to preserve the state of the type at the moment or when writing a client
to an external API it is not desired to require on. If the types depend on other types in can be tiresome to manually copy every field. This package provides automation
for this task.

### Use
See flags in [main.go](cmd/type_snapshot/main.go) for use.
Path to a type is provided as ```<path_to_module>.<struct_name>``` or ```<path_to_file>.<struct_name>```.
To provide many paths, separate by spaces as in [example](#multiple-source-types)

### Restrictions
* No generics support
* Can not snapshot function or interface definition, however struct members of type func and interface will be copied
* Interface methods are not snapshotted as method implementation will not be provided

This package is aimed to capture concrete type definitions primarily for marshalling and unmarshalling.