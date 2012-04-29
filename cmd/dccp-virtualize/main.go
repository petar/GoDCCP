// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

var (
	flagSrc  *string = flag.String("src", ".", "Path to source directory")
	flagDest *string = flag.String("dest", "", "Path to destination directory")
)

func usage() {
	fmt.Printf("%s\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func FilterGoFiles(fi os.FileInfo) bool {
	name := fi.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

func main() {
	flag.Parse()

	if *flagDest == "" {
		usage()
	}
	fileSet := token.NewFileSet()
	pkgs, err := parser.ParseDir(fileSet, *flagPath, FilterGoFiles, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %s\n", err)
		os.Exit(1)
	}

	for pkgName, pkg := range pkgs {
		fmt.Printf("Virtualizing package: %s\n", pkgName)
		VirtualizePackage(pkg, *flagDest)
	}
}
