package scanner

import (
	"go/ast"
)

// extractFuncName и typeToString перенесены из ast_scanner.go для переиспользования.
// Можно вынести в общий файл, но для простоты продублированы.
func extractFuncName(expr ast.Expr, names map[string]bool) {
	switch e := expr.(type) {
	case *ast.Ident:
		names[e.Name] = true
	case *ast.SelectorExpr:
		names[e.Sel.Name] = true
	case *ast.CallExpr:
		if len(e.Args) > 0 {
			extractFuncName(e.Args[0], names)
		}
	}
}

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
