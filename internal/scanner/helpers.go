package scanner

import (
	"go/ast"
	"strings"
)

// extractFuncName рекурсивно извлекает имя функции-обработчика из выражения,
// учитывая цепочки вызовов и .ServeHTTP.
func extractFuncName(expr ast.Expr, names map[string]bool) {
	switch e := expr.(type) {
	case *ast.Ident:
		names[e.Name] = true
	case *ast.SelectorExpr:
		if e.Sel.Name == "ServeHTTP" {
			extractFuncName(e.X, names)
			return
		}
		names[e.Sel.Name] = true
	case *ast.CallExpr:
		funName := ""
		switch fun := e.Fun.(type) {
		case *ast.Ident:
			funName = fun.Name
		case *ast.SelectorExpr:
			funName = typeToString(fun)
		}
		wrapFuncs := map[string]bool{
			"auth.Wrap":    true,
			"calcTiming":   true,
			"pregenTiming": true,
		}
		if wrapFuncs[funName] && len(e.Args) > 0 {
			extractFuncName(e.Args[0], names)
		} else {
			if funName != "" {
				if dot := strings.LastIndexByte(funName, '.'); dot != -1 {
					names[funName[dot+1:]] = true
				} else {
					names[funName] = true
				}
			}
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
