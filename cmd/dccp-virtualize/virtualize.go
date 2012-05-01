// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io"
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
	//Transform(os.Stdout, file)
}

func Transform(w io.Writer, node ast.Node) {
	u := &transform{ Writer: bufio.NewWriter(w) }
	u.xform(node)
	u.Flush()
}

type transform struct {
	*bufio.Writer
	indent int
}

func (u *transform) Indent() {
	u.indent++
}

func (u *transform) Unindent() {
	u.indent--
}

func (u *transform) NL() {
	u.WriteByte('\n')
	for i := 0; i < u.indent; i++ {
		u.WriteByte('\t')
	}
}

func (u *transform) xform(t_ ast.Node) {
	switch t := t_.(type) {
	case *ast.ArrayType:
		u.WriteByte('[')
		u.xform(t.Len)
		u.WriteByte(']')
		u.xform(t.Elt)
	case *ast.AssignStmt:
		for i, lhe := range t.Lhs {
			u.xform(lhe)
			if i+1 < len(t.Lhs) {
				u.WriteString(", ")
			}
		}
		u.WriteString(t.Tok.String())
		for i, rhe := range t.Rhs {
			u.xform(rhe)
			if i+1 < len(t.Lhs) {
				u.WriteString(", ")
			}
		}
		u.WriteByte('\n')
	case *ast.BadDecl, *ast.BadExpr, *ast.BadStmt:
		u.WriteRune('¢')
	case *ast.BasicLit:
		u.WriteString(t.Value)
	case *ast.BinaryExpr:
		u.xform(t.X)
		u.WriteString(" " + t.Op.String() + " ")
		u.xform(t.Y)
	case *ast.BlockStmt:
		u.WriteString("{\n")
		for _, stmt := range t.List {
			u.xform(stmt)
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
		u.xform(t.Fun)
		u.WriteByte('(')
		for i, arg := range t.Args {
			u.xform(arg)
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
				u.xform(arg)
				if i+1 < len(t.List) {
					u.WriteString(", ")
				}
			}
			u.WriteString(":\n")
		}
		for _, stmt := range t.Body {
			u.xform(stmt)
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
		u.xform(t.Value)
	case *ast.CommClause:
		if t.Comm == nil {
			u.WriteString("default:\n")
		} else {
			u.WriteString("case ")
			u.xform(t.Comm)
			u.WriteString(":\n")
		}
		for _, stmt := range t.Body {
			u.xform(stmt)
			u.WriteByte('\n')
		}
	case *ast.Comment:
	case *ast.CommentGroup:
	case *ast.CompositeLit:
		if t.Type != nil {
			u.xform(t.Type)
		}
		u.WriteString("{\n")
		for _, elt := range t.Elts {
			u.xform(elt)
			u.WriteByte('\n')
		}
		u.WriteString("}\n")
	case *ast.DeclStmt:
		u.xform(t.Decl)
	case *ast.DeferStmt:
		u.WriteString("defer ")
		u.xform(t.Call)
	case *ast.Ellipsis:
		u.WriteString("...")
		if t.Elt != nil {
			u.xform(t.Elt)
		}
	case *ast.EmptyStmt:
		u.WriteString("; ")
	case *ast.ExprStmt:
		u.xform(t.X)
	case *ast.Field:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.FieldList:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.File:
		u.WriteString("package ")
		u.WriteString(t.Name.Name)
		u.NL(); u.NL()
		if len(t.Imports) > 0 {
			u.WriteString("import (")
			u.Indent(); u.NL()
		}
		for _, imp := range t.Imports {
			u.xform(imp)
			u.NL()
		}
		if len(t.Imports) > 0 {
			u.WriteString(")")
			u.Unindent(); u.NL()
		}
		u.NL()
		for _, decl := range t.Decls {
			u.xform(decl)
			u.NL()
		}
	case *ast.ForStmt:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.FuncDecl:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.FuncLit:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.FuncType:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.GenDecl:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.GoStmt:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.Ident:
		u.WriteString(t.String())
	case *ast.IfStmt:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.ImportSpec:
		if t.Name != nil {
			u.WriteString(t.Name.Name + " ")
		}
		u.WriteString(t.Path.Value)
	case *ast.IncDecStmt:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.IndexExpr:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.InterfaceType:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.KeyValueExpr:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.LabeledStmt:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.MapType:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.Package:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.ParenExpr:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.RangeStmt:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.ReturnStmt:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.SelectStmt:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.SelectorExpr:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.SendStmt:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.SliceExpr:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.StarExpr:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.StructType:
		u.WriteString(fmt.Sprintf("%T", t))
	case *ast.SwitchStmt:
		u.WriteString("switch ")
		if t.Init != nil {
			u.xform(t.Init)
			u.WriteString("; ")
		}
		u.xform(t.Tag)
		u.WriteString(" ")
		u.xform(t.Body)
	case *ast.TypeAssertExpr:
		u.xform(t.X)
		u.WriteString(".(")
		if t.Type == nil {
			u.WriteString("type")
		} else {
			u.xform(t.Type)
		}
		u.WriteString(")")
	case *ast.TypeSpec:
		u.WriteString("type ")
		u.xform(t.Name)
		u.WriteString(" ")
		u.xform(t.Type)
	case *ast.TypeSwitchStmt:
		u.WriteString("switch ")
		if t.Init != nil {
			u.xform(t.Init)
			u.WriteString("; ")
		}
		u.xform(t.Assign)
		u.WriteString(" ")
		if t.Body != nil {
			u.xform(t.Body)
		}
	case *ast.UnaryExpr:
		u.WriteString(t.Op.String())
		u.xform(t.X)
	case *ast.ValueSpec:
		u.WriteString(fmt.Sprintf("%T", t))
	default:
		u.WriteRune('·')
	}
}
