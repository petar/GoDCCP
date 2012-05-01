package main
import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [go_source_file]\n", os.Args[0])
		os.Exit(1)
	}
	
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, os.Args[1], nil, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %s\n", err)
		os.Exit(1)
	}

	ast.Print(fileSet, file)
}
