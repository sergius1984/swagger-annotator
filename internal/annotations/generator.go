package annotations

import (
	"fmt"
	"go/ast"
	"sort"
	"strings"

	"swagger-annotator/internal/config"
)

func GenerateSwaggerDoc(def *config.HandlerDef) *ast.CommentGroup {
	var lines []string
	add := func(s string) {
		lines = append(lines, "// "+s)
	}

	add(def.Function + " godoc")
	if def.Summary != "" {
		add(fmt.Sprintf("@Summary      %s", def.Summary))
	}
	if def.Description != "" {
		add(fmt.Sprintf("@Description  %s", def.Description))
	}
	if len(def.Tags) > 0 {
		add(fmt.Sprintf("@Tags         %s", strings.Join(def.Tags, ", ")))
	}
	consumes := "json"
	if def.Consumes != "" {
		consumes = def.Consumes
	}
	add(fmt.Sprintf("@Accept       %s", consumes))
	produces := "json"
	if def.Produces != "" {
		produces = def.Produces
	}
	add(fmt.Sprintf("@Produce      %s", produces))

	for _, p := range def.Parameters {
		if line := buildParamLine(p); line != "" {
			add(line)
		}
	}

	codes := make([]string, 0, len(def.Responses))
	for code := range def.Responses {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, codeStr := range codes {
		resp := def.Responses[codeStr]
		if line := buildResponseLine(codeStr, &resp); line != "" {
			add(line)
		}
	}

	for _, sec := range def.Security {
		for key := range sec {
			add(fmt.Sprintf("@Security     %s", key))
		}
	}
	if def.Deprecated {
		add("@Deprecated")
	}
	if def.Path != "" && def.Method != "" {
		add(fmt.Sprintf("@Router       %s [%s]", def.Path, strings.ToLower(def.Method)))
	} else if def.Path != "" {
		add(fmt.Sprintf("@Router       %s", def.Path))
	}

	var list []*ast.Comment
	for _, l := range lines {
		list = append(list, &ast.Comment{Text: l})
	}
	return &ast.CommentGroup{List: list}
}

func buildParamLine(p config.ParameterDef) string {
	if p.Name == "" {
		return ""
	}
	typeStr := ""
	if p.Schema != nil {
		if p.Schema.Ref != "" {
			typeStr = "{object} " + refName(p.Schema.Ref)
		} else if p.Schema.Type != "" {
			typeStr = p.Schema.Type
			if p.Schema.Type == "array" && p.Schema.Items != nil {
				if p.Schema.Items.Ref != "" {
					typeStr = "{array} " + refName(p.Schema.Items.Ref)
				} else if p.Schema.Items.Type != "" {
					typeStr = "{array} " + p.Schema.Items.Type
				}
			}
		}
	}
	if typeStr == "" && p.Type != "" {
		typeStr = p.Type
	}
	if typeStr == "" {
		typeStr = "string"
	}
	req := "false"
	if p.Required {
		req = "true"
	}
	desc := p.Description
	if desc == "" {
		desc = p.Name
	}
	return fmt.Sprintf("@Param        %s %s %s %s \"%s\"", p.Name, p.In, typeStr, req, desc)
}

func buildResponseLine(codeStr string, resp *config.Response) string {
	code, _ := parseInt(codeStr)
	prefix := "@Success"
	if code >= 300 {
		prefix = "@Failure"
	}
	typePart := ""
	if resp.Schema != nil {
		if resp.Schema.Ref != "" {
			typePart = "{object} " + refName(resp.Schema.Ref)
		} else if resp.Schema.Type != "" {
			typePart = "{object} " + resp.Schema.Type
		}
	}
	desc := resp.Description
	if typePart == "" && desc == "" {
		return ""
	}
	line := fmt.Sprintf("%s      %d", prefix, code)
	if typePart != "" {
		line += " " + typePart
	}
	if desc != "" {
		line += fmt.Sprintf(" \"%s\"", desc)
	}
	return line
}

func refName(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ref
}

func parseInt(s string) (int, error) {
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}
