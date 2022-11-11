package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	// Reference the gen package to be friendly to vendoring tools,
	// (The temporary bootstrapping code uses it.)
	// as it is an indirect dependency.
	"github.com/antonvlasov/type-snapshot/pkg/bootstrap"
	_ "github.com/antonvlasov/type-snapshot/pkg/generator"
)

var (
	pathsString = flag.String("paths", "", "required flag, paths to types that will be snapshotted")
	dst         = flag.String("dst", "", "required flag, destination file where snapshot will be written, overrides existing data")
	embedPkg    = flag.String("embed_pkg", "", "types from this package will always retain names")
	suffix      = flag.String("suffix", "", "suffix that will be added to all struct names")
	dstPkg      = flag.String("dst_pkg", "", "package name used in generated file, directory name by default")
	leaveTemps  = flag.Bool("leaveTemps", false, "whether sould leave temporary files")
)

func processFlags() []bootstrap.TypeInfo {
	flag.Parse()

	if *dst == "" {
		flag.Usage()
		os.Exit(1)
	}

	paths := strings.Split(*pathsString, " ")
	if len(paths) == 0 {
		log.Fatal("must provide path to at least one type")
	}

	res := make([]bootstrap.TypeInfo, 0, len(paths))
	for i, p := range paths {
		idx := strings.LastIndexByte(strings.Trim(p, " "), '.')
		if idx == -1 || idx == len(p)-1 {
			log.Fatal(fmt.Errorf("path %d is invalid", i))
		}

		fp, typename := p[:idx], p[idx+1:]

		fInto, err := os.Stat(fp)
		if err != nil {
			log.Fatal(err)
		}

		fp, err = filepath.Abs(fp)
		if err != nil {
			log.Fatal(err)
		}

		res = append(res, bootstrap.TypeInfo{
			Path:     fp,
			IsDir:    fInto.IsDir(),
			TypeName: typename,
		})
	}

	var err error

	*dst, err = filepath.Abs(*dst)
	if err != nil {
		log.Fatal(err)
	}

	if *dstPkg == "" {
		*dstPkg = filepath.Base(filepath.Dir(*dst))
	}

	return res
}

func main() {
	types := processFlags()

	if err := bootstrap.Bootstrap(types, *embedPkg, *dst, *suffix, *dstPkg, *leaveTemps); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
