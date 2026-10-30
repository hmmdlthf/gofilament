package codegen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// SchemaField represents an inspected Ent field.
type SchemaField struct {
	Name        string
	Type        string // String, Text, Int, Float, Bool, Time, Enum, JSON
	IsUnique    bool
	IsRequired  bool
	IsSensitive bool
	DefaultVal  string
	EnumValues  []string
}

// SchemaEdge represents an inspected Ent graph edge.
type SchemaEdge struct {
	Name        string // items, owner, invoices
	EdgeType    string // To, From
	TargetModel string // LineItem, User, Invoice
	IsUnique    bool   // belongsTo if true, hasMany if false
}

// ParsedSchema contains the parsed representation of an Ent schema file.
type ParsedSchema struct {
	ModelName string
	Fields    []SchemaField
	Edges     []SchemaEdge
}

// FindEntSchemaFile searches standard locations for the given model schema.
func FindEntSchemaFile(modelName string) (string, error) {
	lower := strings.ToLower(modelName)
	// Try camel_case to snake_case
	var snake strings.Builder
	for i, r := range modelName {
		if i > 0 && r >= 'A' && r <= 'Z' {
			snake.WriteRune('_')
		}
		snake.WriteRune(r)
	}
	snakeName := strings.ToLower(snake.String())

	searchDirs := []string{
		filepath.Join("app", "schema"),
		filepath.Join("schema"),
		filepath.Join("app", "ent", "schema"),
		filepath.Join("ent", "schema"),
	}

	for _, dir := range searchDirs {
		candidates := []string{
			filepath.Join(dir, fmt.Sprintf("%s.go", lower)),
			filepath.Join(dir, fmt.Sprintf("%s.go", snakeName)),
			filepath.Join(dir, fmt.Sprintf("%s.go", modelName)),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c, nil
			}
		}
	}

	return "", fmt.Errorf("schema file for model %q not found in app/schema/ or schema/", modelName)
}

// ParseEntSchema parses an Ent schema file AST.
func ParseEntSchema(filePath string) (*ParsedSchema, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema file %s: %w", filePath, err)
	}

	schema := &ParsedSchema{
		Fields: make([]SchemaField, 0),
		Edges:  make([]SchemaEdge, 0),
	}

	// 1. Identify model struct name
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				for _, f := range structType.Fields.List {
					if sel, ok := f.Type.(*ast.SelectorExpr); ok {
						if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "ent" && sel.Sel.Name == "Schema" {
							schema.ModelName = typeSpec.Name.Name
						}
					}
				}
			}
		}
	}

	if schema.ModelName == "" {
		// Fallback to filename
		base := filepath.Base(filePath)
		schema.ModelName = strings.Title(strings.TrimSuffix(base, ".go"))
	}

	// 2. Parse Fields() and Edges()
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		switch fn.Name.Name {
		case "Fields":
			schema.Fields = parseFieldsBlock(fn.Body)
		case "Edges":
			schema.Edges = parseEdgesBlock(fn.Body)
		}
	}

	return schema, nil
}

func parseFieldsBlock(body *ast.BlockStmt) []SchemaField {
	var fields []SchemaField
	for _, stmt := range body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			continue
		}
		comp, ok := ret.Results[0].(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, elt := range comp.Elts {
			if call, ok := elt.(*ast.CallExpr); ok {
				f := parseFieldCall(call)
				if f.Name != "" {
					fields = append(fields, f)
				}
			}
		}
	}
	return fields
}

func parseFieldCall(call *ast.CallExpr) SchemaField {
	var fieldDef SchemaField
	fieldDef.IsRequired = true // Default in Ent

	// Walk up method chain (e.g. field.String("name").NotEmpty().Unique())
	curr := call
	for curr != nil {
		sel, ok := curr.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}

		methodName := sel.Sel.Name
		switch methodName {
		case "Unique":
			fieldDef.IsUnique = true
		case "Sensitive":
			fieldDef.IsSensitive = true
		case "Optional":
			fieldDef.IsRequired = false
		case "NotEmpty":
			fieldDef.IsRequired = true
		case "Default":
			if len(curr.Args) > 0 {
				fieldDef.DefaultVal = exprToString(curr.Args[0])
			}
		case "Values":
			for _, arg := range curr.Args {
				fieldDef.EnumValues = append(fieldDef.EnumValues, exprToString(arg))
			}
		case "String", "Text", "Int", "Int64", "Float", "Bool", "Time", "Enum", "JSON", "Bytes":
			fieldDef.Type = methodName
			if len(curr.Args) > 0 {
				fieldDef.Name = strings.Trim(exprToString(curr.Args[0]), `"`)
			}
		}

		if nextCall, ok := sel.X.(*ast.CallExpr); ok {
			curr = nextCall
		} else {
			break
		}
	}

	return fieldDef
}

func parseEdgesBlock(body *ast.BlockStmt) []SchemaEdge {
	var edges []SchemaEdge
	for _, stmt := range body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			continue
		}
		comp, ok := ret.Results[0].(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, elt := range comp.Elts {
			if call, ok := elt.(*ast.CallExpr); ok {
				edge := parseEdgeCall(call)
				if edge.Name != "" {
					edges = append(edges, edge)
				}
			}
		}
	}
	return edges
}

func parseEdgeCall(call *ast.CallExpr) SchemaEdge {
	var edgeDef SchemaEdge
	curr := call
	for curr != nil {
		sel, ok := curr.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}
		methodName := sel.Sel.Name
		switch methodName {
		case "Unique":
			edgeDef.IsUnique = true
		case "To", "From":
			edgeDef.EdgeType = methodName
			if len(curr.Args) >= 1 {
				edgeDef.Name = strings.Trim(exprToString(curr.Args[0]), `"`)
			}
			if len(curr.Args) >= 2 {
				target := exprToString(curr.Args[1])
				target = strings.TrimSuffix(target, ".Type")
				edgeDef.TargetModel = target
			}
		}
		if nextCall, ok := sel.X.(*ast.CallExpr); ok {
			curr = nextCall
		} else {
			break
		}
	}
	return edgeDef
}

func exprToString(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		return v.Value
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", exprToString(v.X), v.Sel.Name)
	default:
		return ""
	}
}
