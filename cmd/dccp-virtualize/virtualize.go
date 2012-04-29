// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/ast"
	//"go/token"
	//"os"
)

func VirtualizePackage(pkg *ast.Package, destDir string) {
	for _, fileFile := range pkg.Files {
		VirtualizeFile(fileFile, destDir)
	}
}

func VirtualizeFile(file *ast.File, destDir string) {
	file.Imports = append(file.Imports, &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: "github.com/petar/GoDCCP/vtime",
		},
	})
}

/*
type PrintVisitor struct{}

func (PrintVisitor) Visit(node ast.Node) ast.Visitor {
	fmt.Printf("%v\n", node)
	return PrintVisitor{}
}
*/
