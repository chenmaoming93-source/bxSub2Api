//go:build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"regexp"
	"os"
	"path/filepath"
	"bytes"
)

var placeholderRE = regexp.MustCompile(`\$([1-9][0-9]*)`)

func literal(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING { return "", false }
		s, err := strconv.Unquote(e.Value); return s, err == nil
	case *ast.BinaryExpr:
		if e.Op != token.ADD { return "", false }
		a, ok := literal(e.X); if !ok { return "", false }
		b, ok := literal(e.Y); if !ok { return "", false }
		return a+b, true
	}
	return "", false
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "rewrite" {
		_ = filepath.Walk("internal", func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || filepath.Ext(path) != ".go" { return nil }
			base := filepath.Base(path)
			if len(base) >= 8 && base[len(base)-8:] == "_test.go" { return nil }
			data, err := os.ReadFile(path); if err != nil { return err }
			updated := placeholderRE.ReplaceAll(data, []byte("?"))
			if !bytes.Equal(data, updated) { return os.WriteFile(path, updated, info.Mode()) }
			return nil
		})
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "rewrite-dynamic" {
		_ = filepath.Walk("internal", func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || filepath.Ext(path) != ".go" { return nil }
			base := filepath.Base(path)
			if len(base) >= 8 && base[len(base)-8:] == "_test.go" { return nil }
			data, err := os.ReadFile(path); if err != nil { return err }
			updated := bytes.ReplaceAll(data, []byte("$%d"), []byte("?/*%d*/"))
			if !bytes.Equal(data, updated) { return os.WriteFile(path, updated, info.Mode()) }
			return nil
		})
		return
	}
	fset := token.NewFileSet()
	_ = filepath.Walk("internal", func(path string, info os.FileInfo, err error) error {
		base := filepath.Base(path)
		if err != nil || info.IsDir() || filepath.Ext(path) != ".go" || (len(base) >= 8 && base[len(base)-8:] == "_test.go") { return nil }
		f, err := parser.ParseFile(fset, path, nil, 0); if err != nil { return nil }
		ast.Inspect(f, func(n ast.Node) bool {
			expr, ok := n.(ast.Expr); if !ok { return true }
			s, ok := literal(expr); if !ok { return true }
			m := placeholderRE.FindAllStringSubmatch(s, -1); if len(m) == 0 { return true }
			seen := map[string]int{}; bad := false; last := 0
			for _, x := range m { seen[x[1]]++; v,_:=strconv.Atoi(x[1]); if v < last || seen[x[1]] > 1 { bad=true }; last=v }
			if bad { fmt.Printf("%s:%d %v\n", path, fset.Position(expr.Pos()).Line, m) }
			return false
		})
		return nil
	})
}
