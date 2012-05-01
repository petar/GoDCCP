// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
)

func VirtualizePackage(fileSet *token.FileSet, pkg *ast.Package, destDir string) {
	for fileName, fileFile := range pkg.Files {
		fmt.Printf("——— virtualizing '%s' ———\n", fileName)
		VirtualizeFile(fileSet, fileFile, destDir)
	}
}

func VirtualizeFile(fileSet *token.FileSet, file *ast.File, destDir string) {
	// Add import of "vtime" package
	addImport(file, "github.com/petar/GoDCCP/vtime")
	printer.Fprint(os.Stdout, fileSet, file)
}
