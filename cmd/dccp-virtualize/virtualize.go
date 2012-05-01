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
	// Replace go statements
	fixGoStmt(file)

	printer.Fprint(os.Stdout, fileSet, file)
}

func fixGoStmt(file *ast.File) {
	walk(file, visitGoStmt)
}

func visitGoStmt(x interface{}) {
	gostmt, ok := x.(*ast.GoStmt)
	if !ok {
		return
	}
	origcall := gostmt.Call
	gostmt.Call = &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{ X: origcall },
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.Ident{ Name: "vtime.Die" },
						},
					},
				},
			},
		},
	}
}
