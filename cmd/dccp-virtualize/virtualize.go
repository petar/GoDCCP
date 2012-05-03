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

// TODO:
//	* Remove import of "time" package if not used other than for Now and Sleep
//	* Ensure there is no other package imported as "vtime"
//	* fallthough in select statements is not supported. check for it.

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
	// Replace time.Now and time.Sleep calls
	fixCallExpr(file)
	// Replace chan operations
	fixChanOps(file)

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
							Fun: &ast.SelectorExpr{
								X:   &ast.Ident{ Name: "vtime" },
								Sel: &ast.Ident{ Name: "Die" },
							},
						},
					},
				},
			},
		},
	}
}

func fixCallExpr(file *ast.File) {
	walk(file, visitCallExpr)
}

func visitCallExpr(x interface{}) {
	callexpr, ok := x.(*ast.CallExpr)
	if !ok || callexpr.Fun == nil {
		return
	}
	sexpr, ok := callexpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	sx, ok := sexpr.X.(*ast.Ident)
	if !ok {
		return
	}
	// TODO: We are assuming that pkg 'time' is imported as 'time'
	if sx.Name != "time" {
		return
	}
	// TODO: We only catch direct calls of the form 'time.Now()'.
	// We would not catch indirect calls as in 'f := time.Now; f()'
	if sexpr.Sel.Name == "Now" || sexpr.Sel.Name == "Sleep" {
		sx.Name = "vtime"
	}
}

func fixChanOps(file *ast.File) {
	walk(file, visitChanOps)
}

// BlockStmt -> AssignStmt, SendStmt, SelectStmt

func visitChanOps(x interface{}) {
	?
}
