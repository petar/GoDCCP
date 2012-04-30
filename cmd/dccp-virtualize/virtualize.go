// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"go/ast"
	//"go/printer"
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
	file.Imports = append(file.Imports, &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: "\"github.com/petar/GoDCCP/vtime\"",
		},
	})
	u := bufio.NewWriter(os.Stdout)
	xform(u, file)
	u.Flush()
}

func xform(u *bufio.Writer, t_ ast.Node) {
	switch t := t_.(type) {
	case *ast.ArrayType:
		u.WriteByte('[')
		xform(u, t.Len)
		u.WriteByte(']')
		xform(u, t.Elt)
	case *ast.AssignStmt:
		for i, lhe := range t.Lhs {
			xform(u, lhe)
			if i+1 < len(t.Lhs) {
				u.WriteString(", ")
			}
		}
		u.WriteString(t.Tok.String())
		for i, rhe := range t.Rhs {
			xform(u, rhe)
			if i+1 < len(t.Lhs) {
				u.WriteString(", ")
			}
		}
		u.WriteByte('\n')
	case *ast.Ident:
		u.WriteString(t.Name)
	case *ast.BadDecl, *ast.BadExpr, *ast.BadStmt:
		u.WriteRune('¢')
	case *ast.BasicLit:
		u.WriteString(t.Value)
	case *ast.BinaryExpr:
		xform(u, t.X)
		u.WriteString(" " + t.Op.String() + " ")
		xform(u, t.Y)
	case *ast.BlockStmt:
		u.WriteString("{\n")
		for _, stmt := range t.List {
			xform(u, stmt)
			u.WriteByte('\n')
		}
		u.WriteString("}\n")
	case *ast.BranchStmt:
		u.WriteString(t.Tok.String())
		if t.Label != nil {
			u.WriteByte(' ')
			u.WriteString(t.Label.Name)
		}
	case *ast.CallExpr:
		xform(u, t.Fun)
		u.WriteByte('(')
		for i, arg := range t.Args {
			xform(u, arg)
			if i+1 < len(t.Args) {
				u.WriteString(", ")
			}
		}
		u.WriteByte(')')
	case *ast.CaseClause:
		if t.List == nil {
			u.WriteString("default:\n")
		} else {
			u.WriteString("case ")
			for i, arg := range t.List {
				xform(u, arg)
				if i+1 < len(t.List) {
					u.WriteString(", ")
				}
			}
			u.WriteString(":\n")
		}
		for _, stmt := range t.Body {
			xform(u, stmt)
			u.WriteByte('\n')
		}
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			u.WriteString("chan<- ")
		case ast.RECV:
			u.WriteString("<-chan ")
		default:
			u.WriteString("chan ")
		}
		xform(u, t.Value)
	case *ast.CommClause:
		?
	case *ast.File:
		u.WriteString("package ")
		u.WriteString(t.Name.Name)
		u.WriteString("\n\n")
		if len(t.Imports) > 0 {
			u.WriteString("import (\n")
		}
		for _, imp := range t.Imports {
			if imp.Name != nil {
				u.WriteString(imp.Name.Name + " ")
			}
			u.WriteString(imp.Path.Value)
			u.WriteByte('\n')
		}
		if len(t.Imports) > 0 {
			u.WriteString(")\n")
		}
		u.WriteByte('\n')
		for _, decl := range t.Decls {
			xform(u, decl)
			u.WriteByte('\n')
		}
	default:
		u.WriteRune('·')
	}
}
