package scanner

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

type ASTScanner struct{}

func (s *ASTScanner) Find(root string) ([]HandlerLocation, error) {
	routeNames, err := collectRouteHandlersAST(root)
	if err != nil {
		return nil, fmt.Errorf("сбор маршрутов: %w", err)
	}

	var handlers []HandlerLocation
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "vendor" || d.Name() == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if !isHTTPHandlerFuncAST(fn) {
				continue
			}
			name := fn.Name.Name
			if _, used := routeNames[name]; !used {
				continue
			}
			info := HandlerLocation{
				FuncName: name,
				File:     rel,
			}
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				info.RecvType = typeToString(fn.Recv.List[0].Type)
				info.RecvType = strings.TrimPrefix(info.RecvType, "*")
			}
			if fn.Doc != nil {
				for _, c := range fn.Doc.List {
					if strings.HasPrefix(c.Text, "// @") {
						info.HasSwagger = true
						break
					}
				}
			}
			handlers = append(handlers, info)
		}
		return nil
	})
	return handlers, err
}

func collectRouteHandlersAST(root string) (map[string]bool, error) {
	names := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil // игнорируем ошибки парсинга отдельных файлов
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name != "Handle" && sel.Sel.Name != "HandleFunc" {
				return true
			}
			if len(call.Args) < 2 {
				return true
			}
			extractFuncName(call.Args[1], names)
			return true
		})
		return nil
	})
	return names, err
}

func isHTTPHandlerFuncAST(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 2 {
		return false
	}
	p0 := typeToString(fn.Type.Params.List[0].Type)
	p1 := typeToString(fn.Type.Params.List[1].Type)
	return (p0 == "http.ResponseWriter" || strings.HasSuffix(p0, ".ResponseWriter")) &&
		(p1 == "*http.Request" || strings.HasSuffix(p1, ".Request"))
}
