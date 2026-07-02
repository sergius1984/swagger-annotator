package updater

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"swagger-annotator/internal/annotations"
	"swagger-annotator/internal/config"
)

// UpdateComments вставляет/обновляет Swagger-комментарий для указанной функции.
// dryRun – если true, то файл не перезаписывается.
func UpdateComments(fileName, funcName, recvType string, def *config.HandlerDef, dryRun bool, projectDir string) error {
	if dryRun {
		return nil
	}
	fullPath := filepath.Join(projectDir, filepath.FromSlash(fileName))
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, fullPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("парсинг %s: %w", fileName, err)
	}

	var target *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == funcName {
			if recvType != "" {
				if fn.Recv == nil || len(fn.Recv.List) == 0 {
					continue
				}
				rType := typeToString(fn.Recv.List[0].Type)
				rType = strings.TrimPrefix(rType, "*")
				if rType != recvType {
					continue
				}
			}
			target = fn
			break
		}
	}
	if target == nil {
		return fmt.Errorf("функция %s не найдена", funcName)
	}

	newDoc := annotations.GenerateSwaggerDoc(def)
	target.Doc = newDoc

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return fmt.Errorf("форматирование: %w", err)
	}
	return os.WriteFile(fullPath, buf.Bytes(), 0644)
}

// Вспомогательная функция (можно вынести в общий util, но для простоты здесь)
func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	default:
		return ""
	}
}
