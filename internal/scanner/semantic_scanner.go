package scanner

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type SemanticScanner struct{}

func (s *SemanticScanner) Find(root string) ([]HandlerLocation, error) {
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  root,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("загрузка пакетов: %w", err)
	}

	// Сначала собираем имена хендлеров из маршрутов (можно использовать AST по загруженным файлам)
	routeNames := map[string]bool{}
	for _, pkg := range pkgs {
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

	// Теперь ищем функции с правильной сигнатурой, используя типы
	var handlers []HandlerLocation
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			rel := filepath.ToSlash(relativePath(root, pkg.Fset.File(file.Pos()).Name()))
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				name := fn.Name.Name
				if _, used := routeNames[name]; !used {
					continue
				}
				// Проверка сигнатуры с использованием типов
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
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 2 {
		return false
	}
	// Используем TypesInfo для точного сравнения
	t1 := pkg.TypesInfo.TypeOf(fn.Type.Params.List[0].Type)
	t2 := pkg.TypesInfo.TypeOf(fn.Type.Params.List[1].Type)
	if t1 == nil || t2 == nil {
		return false
	}
	// Проверяем, что t1 реализует http.ResponseWriter, а t2 — *http.Request
	respWriter := lookupType(pkg, "net/http", "ResponseWriter")
	reqType := lookupType(pkg, "net/http", "Request")
	if respWriter == nil || reqType == nil {
		return false
	}
	if !types.Implements(t1, respWriter.Underlying().(*types.Interface)) {
		return false
	}
	// Для *http.Request: t2 должен быть указателем на именованный тип, чей базовый тип — Request
	if ptr, ok := t2.(*types.Pointer); ok {
		if named, ok := ptr.Elem().(*types.Named); ok {
			return named.Obj() == reqType.Obj()
		}
	}
	return false
}

// lookupType ищет тип в импортированном пакете
func lookupType(pkg *packages.Package, importPath, name string) *types.Named {
	for _, imp := range pkg.Pkg.Imports() {
		if imp.Path() == importPath {
			obj := imp.Scope().Lookup(name)
			if obj != nil {
				return obj.Type().(*types.Named)
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
