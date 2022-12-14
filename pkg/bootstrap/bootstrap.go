package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/antonvlasov/type-snapshot/pkg/parser"
)

type TypeInfo struct {
	Path     string
	IsDir    bool
	PkgName  string
	TypeName string
}

func Bootstrap(types []TypeInfo, emdebPkg, dst, suffix, dstPkg string, leaveTemps bool) (err error) {
	for i := range types {
		srcPkg, err := parser.GetPkgPath(types[i].Path, types[i].IsDir)
		if err != nil {
			return err
		}

		types[i].PkgName = srcPkg
	}

	var path string
	if types[0].IsDir {
		path = types[0].Path
	} else {
		path = filepath.Dir(types[0].Path)
	}

	mainPath, err := writeMain(types, path)
	if err != nil {
		return err
	}

	if !leaveTemps {
		defer os.Remove(mainPath)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o775); err != nil {
		return err
	}

	f, err := os.OpenFile(dst+".tmp", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}

	if !leaveTemps {
		defer os.Remove(f.Name()) // will not remove after rename
	}

	execArgs := []string{
		"run",
		"-mod=mod",
		mainPath,
		"-embed_pkg", emdebPkg,
		"-dst_pkg", dstPkg,
		"-suffix", suffix,
	}
	cmd := exec.Command("go", execArgs...)

	cmd.Stdout = f
	cmd.Stderr = os.Stderr
	cmd.Dir = path
	if err = cmd.Run(); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return os.Rename(f.Name(), dst)
}

func writeMain(types []TypeInfo, path string) (string, error) {
	f, err := os.CreateTemp(path, "typesnapshot-bootstrap") // can use any path in source module
	if err != nil {
		return "", err
	}

	fmt.Fprintln(f, "// +build ignore")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "// TEMPORARY AUTOGENERATED FILE: typesnapshot bootstapping code to launch")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "package main")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "import (")
	fmt.Fprintln(f, `  "flag"`)
	fmt.Fprintln(f, `  "fmt"`)
	fmt.Fprintln(f, `  "os"`)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "  %q\n", generatorPkg)
	fmt.Fprintln(f)
	for i, t := range types {
		fmt.Fprintf(f, "  pkg%d %q\n", i, t.PkgName)
	}

	fmt.Fprintln(f, ")")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "var (")
	fmt.Fprintln(f, `	embedPkg = flag.String("embed_pkg", "", "")`)
	fmt.Fprintln(f, `	dstPkg  = flag.String("dst_pkg", "", "")`)
	fmt.Fprintln(f, `	suffix  = flag.String("suffix", "", "")`)
	fmt.Fprintln(f, ")")
	fmt.Fprintln(f)

	fmt.Fprintln(f, "func main() {")
	fmt.Fprintln(f, "	flag.Parse()")
	fmt.Fprintln(f, "")

	fmt.Fprintf(f, "	g := generator.New(*embedPkg, *suffix, *dstPkg)")
	fmt.Fprintln(f)
	for i, t := range types {
		fmt.Fprintf(f, "	g.Add(pkg%d.%s{})", i, t.TypeName)
		fmt.Fprintln(f)
	}
	fmt.Fprintln(f, "	if err := g.SaveSnapshot(os.Stdout); err != nil {")
	fmt.Fprintln(f, "		fmt.Fprintln(os.Stderr, err)")
	fmt.Fprintln(f, "		os.Exit(1)")
	fmt.Fprintln(f, "	}")
	fmt.Fprintln(f, "}")

	src := f.Name()
	if err := f.Close(); err != nil {
		return src, err
	}

	dest := src + ".go"
	return dest, os.Rename(src, dest)
}
