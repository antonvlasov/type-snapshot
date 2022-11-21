package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode"
)

type Generator struct {
	out          bytes.Buffer
	suffix       string
	embedPkgName string
	dstPkgName   string
	endline      string
	knownTypes   map[reflect.Type]struct{}
	pkgPrefixes  map[string]string
}

func (r *Generator) addType(t reflect.Type) bool {
	// don't add basic types
	if t.Name() == t.Kind().String() {
		return false
	}

	_, ok := r.knownTypes[t]

	r.knownTypes[t] = struct{}{}

	return !ok
}

func (r *Generator) collectDefinitionsRecursive(t reflect.Type) {
	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			fieldType := t.Field(i).Type
			if r.addType(fieldType) {
				r.collectDefinitionsRecursive(fieldType)
			}
		}
	case reflect.Slice, reflect.Array, reflect.Chan, reflect.Pointer:
		elem := t.Elem()

		if r.addType(elem) {
			r.collectDefinitionsRecursive(elem)
		}
	case reflect.Map:
		key := t.Key()
		if r.addType(key) {
			r.collectDefinitionsRecursive(key)
		}

		elem := t.Elem()
		if r.addType(elem) {
			r.collectDefinitionsRecursive(elem)
		}
	case reflect.Func:
		r.addType(t)
		for i := 0; i < t.NumIn(); i++ {
			arg := t.In(i)
			if r.addType(arg) {
				r.collectDefinitionsRecursive(arg)
			}
		}

		for i := 0; i < t.NumOut(); i++ {
			arg := t.Out(i)
			if r.addType(arg) {
				r.collectDefinitionsRecursive(arg)
			}
		}
	default:
		r.addType(t)
	}
}

func (r *Generator) collectCollisions() []string {
	collisions := make(map[string][]string)
	for t := range r.knownTypes {
		collisions[t.Name()] = append(collisions[t.Name()], t.PkgPath())
	}

	var allCollisions []string
	for _, c := range collisions {
		if len(c) > 1 {
			allCollisions = append(allCollisions, c...)
		}
	}

	return allCollisions
}

func (r *Generator) solveCollisions() {
	packages := r.collectCollisions()
	r.pkgPrefixes = make(map[string]string, len(packages))

	parts := make(map[string][]string, len(packages))
	for _, p := range packages {
		parts[p] = strings.Split(p, "/")
		for i := 0; i < len(parts[p])/2; i++ {
			parts[p][i], parts[p][len(parts[p])-1-i] = parts[p][len(parts[p])-1-i], parts[p][i]
		}
	}

	type collision struct {
		packageName string
		count       int
	}

	for j := 0; len(parts) != 0; j++ {
		collisions := make(map[string]*collision)
		for p, pp := range parts {
			r.pkgPrefixes[p] = constructPrefix(r.pkgPrefixes[p], pp[j])

			if _, ok := collisions[r.pkgPrefixes[p]]; !ok {
				collisions[r.pkgPrefixes[p]] = &collision{
					packageName: p,
				}
			}

			collisions[r.pkgPrefixes[p]].count += 1
		}

		for _, v := range collisions {
			if v.count == 1 {
				delete(parts, v.packageName)
			}
		}
	}

	for k := range r.pkgPrefixes {
		if k == r.embedPkgName {
			r.pkgPrefixes[k] = ""
			break
		}
	}
}

func constructPrefix(have, prefix string) string {
	return strings.ReplaceAll(strings.Title(prefix), "-", "") + have
}

func (r *Generator) dropAnonymous() {
	var forDrop []reflect.Type
	for t := range r.knownTypes {
		if t.Name() == "" {
			forDrop = append(forDrop, t)
		}
	}

	for _, fd := range forDrop {
		delete(r.knownTypes, fd)
	}
}

func shouldAddSuffix(name string) bool {
	switch name {
	case
		"bool",
		"int",
		"int8",
		"int16",
		"int32",
		"int64",
		"uint",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"uintptr",
		"float32",
		"float64",
		"complex64",
		"complex128",
		"string":
		return false
	default:
		return true
	}
}

func (r *Generator) getObjName(t reflect.Type) string {
	ptrs, digged := digPointers(t)
	var name string

	if digged.Name() != "" {
		if p := r.pkgPrefixes[digged.PkgPath()]; p != "" {
			name = p + strings.Title(digged.Name())
			if !unicode.IsUpper([]rune(digged.Name())[0]) {
				name = strings.ToLower(name[0:1]) + name[1:]
			}
		} else {
			name = digged.Name()
		}

		if shouldAddSuffix(name) {
			name += r.suffix
		}
	} else {
		name = r.getAnonymousObjectDefinition(digged)
	}

	return ptrs + name
}

func digPointers(t reflect.Type) (string, reflect.Type) {
	i := 0
	for {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
			i++
		} else {
			break
		}
	}

	return strings.Repeat("*", i), t
}

func (r *Generator) getAnonymousObjectDefinition(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", r.getObjName(t.Key()), r.getObjName(t.Elem()))
	case reflect.Slice:
		return "[]" + r.getObjName(t.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), r.getObjName(t.Elem()))
	case reflect.Chan:
		switch t.ChanDir() {
		case reflect.RecvDir:
			return "chan<- " + r.getObjName(t.Elem())
		case reflect.SendDir:
			return "<-chan " + r.getObjName(t.Elem())
		default:
			return "chan " + r.getObjName(t.Elem())
		}
	case reflect.Struct:
		return r.generateStructDefinition(t)
	case reflect.Func:
		return r.generateFuncDefinition(t)
	default:
		return t.Kind().String()
	}
}

type typesByName struct {
	types []reflect.Type
	names []string
}

func (r typesByName) Len() int {
	return len(r.types)
}

func (r typesByName) Swap(i, j int) {
	r.types[i], r.types[j] = r.types[j], r.types[i]
	r.names[i], r.names[j] = r.names[j], r.names[i]
}

func (r typesByName) Less(i, j int) bool {
	return r.names[i] < r.names[j]
}

func (r *Generator) getTypesOfKind(kind reflect.Kind) ([]string, []reflect.Type) {
	var res []reflect.Type

	for t := range r.knownTypes {
		_, digged := digPointers(t)
		if digged.Kind() == kind || kind == reflect.Invalid { // invalid is used for left out types
			res = append(res, t)
		}
	}

	names := make([]string, len(res))
	for i, t := range res {
		names[i] = r.getObjName(t)
	}

	sort.Sort(typesByName{
		types: res,
		names: names,
	})

	for _, t := range res {
		delete(r.knownTypes, t)
	}

	return names, res
}

func (r *Generator) generateDefinition(t reflect.Type) string {
	ptrs, digged := digPointers(t)

	var definition string
	switch digged.Kind() {
	case reflect.Func:
		definition = r.generateFuncDefinition(t)
	case reflect.Struct:
		definition = r.generateStructDefinition(t)
	case reflect.Interface:
		definition = r.generateInterfaceDefinition(t)
	default:
		definition = r.generateSimpleDefinition(t)
	}

	return ptrs + definition
}

func (r *Generator) generateInterfaceDefinition(t reflect.Type) string {
	b := strings.Builder{}
	b.WriteString("interface{}") // omit interface methods as we don't want to implement methods

	return b.String()
}

func (r *Generator) generateStructDefinition(t reflect.Type) string {
	b := strings.Builder{}
	b.WriteString("struct{")
	b.WriteString(r.endline)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		b.WriteRune('\t')

		if !f.Anonymous {
			b.WriteString(f.Name)
			b.WriteRune(' ')
		}

		b.WriteString(r.getObjName(f.Type))

		if string(f.Tag) != "" {
			b.WriteRune(' ')
			b.WriteString("`")
			b.WriteString(string(f.Tag))
			b.WriteString("`")
		}

		b.WriteString(r.endline)
	}

	b.WriteString("}")

	return b.String()
}

func (r *Generator) generateSimpleDefinition(t reflect.Type) string {
	return r.getAnonymousObjectDefinition(t)
}

func (r *Generator) generateFuncDefinition(t reflect.Type) string {
	b := strings.Builder{}
	b.WriteString("func(")
	for i := 0; i < t.NumIn(); i++ {
		b.WriteString(r.getObjName(t.In(i)))
		if i != t.NumIn()-1 {
			b.WriteString(", ")
		}
	}

	b.WriteString(")")

	if t.NumOut() > 1 {
		b.WriteString("(")
	}

	for i := 0; i < t.NumOut(); i++ {
		b.WriteString(r.getObjName(t.Out(i)))
		if i != t.NumOut()-1 {
			b.WriteString(", ")
		}
	}

	if t.NumOut() > 1 {
		b.WriteString(")")
	}

	return b.String()
}

func (r *Generator) outputSnapshot() {
	r.out.WriteString("package ")
	r.out.WriteString(fmt.Sprintf("%s", r.dstPkgName))
	r.out.WriteString(r.endline)
	r.out.WriteString(r.endline)

	r.out.WriteString("//")
	r.out.WriteString(r.endline)
	r.out.WriteString("// These types were autogenerated by snapshotting existing types using " + generatorPkg)
	r.out.WriteString(r.endline)
	r.out.WriteString("//")
	r.out.WriteString(r.endline)

	r.out.WriteString("type (")
	r.out.WriteString(r.endline)

	for _, t := range typesOrder {
		names, types := r.getTypesOfKind(t)

		for i, f := range types {
			r.out.WriteString(names[i])
			r.out.WriteRune(' ')
			r.out.WriteString(r.generateDefinition(f))
			r.out.WriteString(r.endline)
			r.out.WriteString(r.endline)
		}
	}

	r.out.WriteString(")")
}

func New(embedPkgName, suffix, dstPkg string) *Generator {
	return &Generator{
		knownTypes:   make(map[reflect.Type]struct{}),
		suffix:       suffix,
		embedPkgName: embedPkgName, // types from this package will always retain names
		dstPkgName:   dstPkg,
		endline:      "\n",
	}
}

func (r *Generator) Add(obj interface{}) {
	t := reflect.TypeOf(obj)
	r.addType(t)
	r.collectDefinitionsRecursive(t)
}

func (r *Generator) SaveSnapshot(out io.Writer) error {
	r.dropAnonymous()
	r.solveCollisions()
	r.outputSnapshot()

	formatted, err := format.Source(r.out.Bytes())
	if err != nil {
		return err
	}

	_, err = out.Write(formatted)

	return err
}
