package updater

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"

	"swagger-annotator/internal/annotations"
	"swagger-annotator/internal/config"
)

// UpdateComments вставляет / обновляет Swagger-комментарий для указанной функции.
func UpdateComments(fileName, funcName, recvType string, def *config.HandlerDef, dryRun bool, projectDir string) error {
	if dryRun {
		return nil
	}
	fullPath := filepath.Join(projectDir, filepath.FromSlash(fileName))

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("чтение %s: %w", fileName, err)
	}

	lines := strings.Split(string(content), "\n")
	insertLine := -1

	// Ищем строку с объявлением нужной функции
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if recvType == "" {
			// обычная функция
			if strings.HasPrefix(trimmed, "func "+funcName+"(") ||
				strings.HasPrefix(trimmed, "func "+funcName+" ") {
				insertLine = i
				break
			}
		} else {
			// метод
			if strings.Contains(trimmed, "func (") &&
				strings.Contains(trimmed, ") "+funcName+"(") &&
				strings.Contains(trimmed, recvType) {
				insertLine = i
				break
			}
		}
	}

	if insertLine == -1 {
		return fmt.Errorf("функция %s не найдена в файле", funcName)
	}

	newDoc := annotations.GenerateSwaggerDoc(def)
	var commentLines []string
	for _, c := range newDoc.List {
		commentLines = append(commentLines, c.Text)
	}

	funcLine := lines[insertLine]
	indent := ""
	if len(funcLine) > 0 {
		for _, ch := range funcLine {
			if ch == ' ' || ch == '\t' {
				indent += string(ch)
			} else {
				break
			}
		}
	}

	var newCommentLines []string
	for _, cl := range commentLines {
		newCommentLines = append(newCommentLines, indent+cl)
	}

	// Удаляем предыдущий doc-комментарий (если был)
	removeStart := insertLine
	for i := insertLine - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "//") || trimmed == "" {
			removeStart = i
		} else {
			break
		}
	}

	newLines := make([]string, 0, len(lines))
	newLines = append(newLines, lines[:removeStart]...)
	newLines = append(newLines, newCommentLines...)
	newLines = append(newLines, lines[insertLine:]...)

	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("запись %s: %w", fileName, err)
	}
	return nil
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
