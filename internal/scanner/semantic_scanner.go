package scanner

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type SemanticScanner struct {
	AllHandlers bool
}

func (s *SemanticScanner) Find(root string) ([]HandlerLocation, error) {
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  root,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("загрузка пакетов: %w", err)
	}

	// Собираем имена из маршрутов, если не AllHandlers
	var routeNames map[string]bool
	if !s.AllHandlers {
		routeNames = map[string]bool{}
		for _, pkg := range pkgs {
			if pkg.Types == nil {
				continue
			}
			for _, file := range pkg.Syntax {
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
					extractFuncName(call.Args[1], routeNames)
					return true
				})
			}
		}
	}

	var handlers []HandlerLocation
	for _, pkg := range pkgs {
		if pkg.Types == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			rel := filepath.ToSlash(relativePath(root, pkg.Fset.File(file.Pos()).Name()))
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				name := fn.Name.Name
				if !s.AllHandlers {
					if _, used := routeNames[name]; !used {
						continue
					}
				}
				if !isHTTPHandlerFuncSemantic(pkg, fn) {
					continue
				}
				info := HandlerLocation{
					FuncName: name,
					File:     rel,
				}
				if fn.Recv != nil && len(fn.Recv.List) > 0 {
					tv := pkg.TypesInfo.TypeOf(fn.Recv.List[0].Type)
					if tv != nil {
						info.RecvType = strings.TrimPrefix(tv.String(), "*")
					} else {
						info.RecvType = typeToString(fn.Recv.List[0].Type)
					}
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
		}
	}
	return handlers, nil
}

func isHTTPHandlerFuncSemantic(pkg *packages.Package, fn *ast.FuncDecl) bool {
	// Проверка прямого обработчика: ровно два параметра и соответствующие типы
	if fn.Type.Params != nil && len(fn.Type.Params.List) == 2 {
		t1 := pkg.TypesInfo.TypeOf(fn.Type.Params.List[0].Type)
		t2 := pkg.TypesInfo.TypeOf(fn.Type.Params.List[1].Type)
		if t1 != nil && t2 != nil {
			respWriter := lookupType(pkg, "net/http", "ResponseWriter")
			reqType := lookupType(pkg, "net/http", "Request")
			if respWriter != nil && reqType != nil {
				if types.Implements(t1, respWriter.Underlying().(*types.Interface)) {
					if ptr, ok := t2.(*types.Pointer); ok {
						if named, ok := ptr.Elem().(*types.Named); ok {
							if named.Obj() == reqType.Obj() {
								return true
							}
						}
					}
				}
			}
		}
	}
	// Проверка фабрики: возвращает http.HandlerFunc или http.Handler
	if fn.Type.Results != nil && len(fn.Type.Results.List) == 1 {
		retType := pkg.TypesInfo.TypeOf(fn.Type.Results.List[0].Type)
		if retType != nil {
			retTypeStr := retType.String()
			if strings.HasSuffix(retTypeStr, "http.HandlerFunc") ||
				strings.HasSuffix(retTypeStr, "http.Handler") {
				return true
			}
		}
	}
	return false
}

func lookupType(pkg *packages.Package, importPath, name string) *types.Named {
	if pkg.Types == nil {
		return nil
	}
	for _, imp := range pkg.Types.Imports() {
		if imp.Path() == importPath {
			obj := imp.Scope().Lookup(name)
			if obj != nil {
				if named, ok := obj.Type().(*types.Named); ok {
					return named
				}
			}
		}
	}
	return nil
}

func relativePath(root, absPath string) string {
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return absPath
	}
	return filepath.ToSlash(rel)
}
