package codegen

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
)

// GenerateResourceCode generates full Resource code from an inspected Ent schema.
func GenerateResourceCode(schema *ParsedSchema) (string, error) {
	modelName := schema.ModelName
	pluralName := modelName + "s"
	slug := strings.ToLower(pluralName)

	var formFields strings.Builder
	var tableColumns strings.Builder
	var tableFilters strings.Builder
	var infolistEntries strings.Builder

	// ID column
	tableColumns.WriteString(fmt.Sprintf("\n\t\t\t\ttables.TextColumn[*%s](\"id\").Sortable(),", modelName))

	for _, f := range schema.Fields {
		fieldName := f.Name
		isEmail := strings.Contains(strings.ToLower(fieldName), "email")
		isPassword := f.IsSensitive || strings.Contains(strings.ToLower(fieldName), "password")
		isMoney := strings.Contains(strings.ToLower(fieldName), "price") || strings.Contains(strings.ToLower(fieldName), "amount") || strings.Contains(strings.ToLower(fieldName), "total")

		// 1. Form Field
		switch f.Type {
		case "String":
			if isEmail {
				formFields.WriteString(fmt.Sprintf("\n\t\t\t\t\tforms.TextInput(\"%s\").Email().Required().Placeholder(\"%s@example.com\"),", fieldName, fieldName))
			} else if isPassword {
				formFields.WriteString(fmt.Sprintf("\n\t\t\t\t\tforms.TextInput(\"%s\").Password().RequiredOnCreate().Hash(),", fieldName))
			} else {
				reqStr := ""
				if f.IsRequired {
					reqStr = ".Required()"
				}
				formFields.WriteString(fmt.Sprintf("\n\t\t\t\t\tforms.TextInput(\"%s\")%s,", fieldName, reqStr))
			}
		case "Text":
			formFields.WriteString(fmt.Sprintf("\n\t\t\t\t\tforms.Textarea(\"%s\").Rows(4),", fieldName))
		case "Enum":
			var opts strings.Builder
			for _, v := range f.EnumValues {
				cleanVal := strings.Trim(v, `"`)
				opts.WriteString(fmt.Sprintf(`"%s": "%s", `, cleanVal, strings.Title(cleanVal)))
			}
			defStr := ""
			if f.DefaultVal != "" {
				defStr = fmt.Sprintf(`.Default(%s)`, f.DefaultVal)
			}
			formFields.WriteString(fmt.Sprintf("\n\t\t\t\t\tforms.Select(\"%s\").Options(map[string]string{%s})%s.Required(),", fieldName, opts.String(), defStr))
		case "Bool":
			formFields.WriteString(fmt.Sprintf("\n\t\t\t\t\tforms.Toggle(\"%s\"),", fieldName))
		case "Time":
			formFields.WriteString(fmt.Sprintf("\n\t\t\t\t\tforms.DateTimePicker(\"%s\"),", fieldName))
		case "Float", "Int", "Int64":
			prefixStr := ""
			if isMoney {
				prefixStr = `.Prefix("$")`
			}
			formFields.WriteString(fmt.Sprintf("\n\t\t\t\t\tforms.TextInput(\"%s\").Numeric()%s,", fieldName, prefixStr))
		case "JSON":
			formFields.WriteString(fmt.Sprintf("\n\t\t\t\t\tforms.KeyValue(\"%s\"),", fieldName))
		default:
			formFields.WriteString(fmt.Sprintf("\n\t\t\t\t\tforms.TextInput(\"%s\"),", fieldName))
		}

		// 2. Table Column
		switch f.Type {
		case "String":
			if isEmail {
				tableColumns.WriteString(fmt.Sprintf("\n\t\t\t\ttables.TextColumn[*%s](\"%s\").Searchable().Copyable(),", modelName, fieldName))
			} else if !isPassword {
				if f.IsUnique || strings.Contains(strings.ToLower(fieldName), "name") || strings.Contains(strings.ToLower(fieldName), "title") || strings.Contains(strings.ToLower(fieldName), "number") {
					tableColumns.WriteString(fmt.Sprintf("\n\t\t\t\ttables.TextColumn[*%s](\"%s\").Searchable().Sortable().Weight(\"bold\"),", modelName, fieldName))
				} else {
					tableColumns.WriteString(fmt.Sprintf("\n\t\t\t\ttables.TextColumn[*%s](\"%s\").Sortable(),", modelName, fieldName))
				}
			}
		case "Enum":
			tableColumns.WriteString(fmt.Sprintf("\n\t\t\t\ttables.BadgeColumn[*%s](\"%s\"),", modelName, fieldName))
			// Add filter for enum
			tableFilters.WriteString(fmt.Sprintf("\n\t\t\t\ttables.SelectFilter(\"%s\"),", fieldName))
		case "Bool":
			tableColumns.WriteString(fmt.Sprintf("\n\t\t\t\ttables.IconColumn[*%s](\"%s\").Boolean(),", modelName, fieldName))
			tableFilters.WriteString(fmt.Sprintf("\n\t\t\t\ttables.TernaryFilter(\"%s\"),", fieldName))
		case "Time":
			tableColumns.WriteString(fmt.Sprintf("\n\t\t\t\ttables.TextColumn[*%s](\"%s\").DateTime().Sortable(),", modelName, fieldName))
		case "Float", "Int", "Int64":
			if isMoney {
				tableColumns.WriteString(fmt.Sprintf("\n\t\t\t\ttables.TextColumn[*%s](\"%s\").Money(\"USD\").Sortable(),", modelName, fieldName))
			} else {
				tableColumns.WriteString(fmt.Sprintf("\n\t\t\t\ttables.TextColumn[*%s](\"%s\").Sortable(),", modelName, fieldName))
			}
		}

		// 3. Infolist Entry
		if !isPassword {
			switch f.Type {
			case "Enum":
				infolistEntries.WriteString(fmt.Sprintf("\n\t\t\t\t\t\tinfolists.BadgeEntry(\"%s\"),", fieldName))
			case "Bool":
				infolistEntries.WriteString(fmt.Sprintf("\n\t\t\t\t\t\tinfolists.IconEntry(\"%s\").Boolean(),", fieldName))
			case "Time":
				infolistEntries.WriteString(fmt.Sprintf("\n\t\t\t\t\t\tinfolists.TextEntry(\"%s\").DateTime(),", fieldName))
			default:
				if isMoney {
					infolistEntries.WriteString(fmt.Sprintf("\n\t\t\t\t\t\tinfolists.TextEntry(\"%s\").Money(\"USD\"),", fieldName))
				} else {
					infolistEntries.WriteString(fmt.Sprintf("\n\t\t\t\t\t\tinfolists.TextEntry(\"%s\"),", fieldName))
				}
			}
		}
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(`package resources

import (
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/tables"
)

type %s struct {
	ID string
}

type %sResource struct {
	*filament.BaseResource[*%s]
}

func New%sResource() *%sResource {
	res := filament.NewResource[*%s]("%s").
		ModelName("%s").
		PluralModelName("%s").
		NavigationIcon("heroicon-rectangle-stack").
		Form(func(form *forms.Form) *forms.Form {
			return form.Schema(
				forms.Section("%s Information").
					Columns(2).
					Schema(%s
					),
			)
		}).
		Table(func(table *tables.Table[*%s]) *tables.Table[*%s] {
			return table.
				Columns(%s
				).
				Filters(%s
				).
				Actions(
					tables.ViewAction[*%s](),
					tables.EditAction[*%s](),
					tables.DeleteAction[*%s](),
				).
				BulkActions(
					tables.DeleteBulkAction(),
					tables.ExportBulkAction(),
				)
		}).
		Infolist(func(infolist *infolists.Infolist) *infolists.Infolist {
			return infolist.Schema(
				infolists.Section("%s Details").
					Columns(2).
					Schema(%s
					),
			)
		})

	return &%sResource{BaseResource: res}
}
`,
		modelName, modelName, modelName,
		modelName, modelName, modelName, slug, modelName, pluralName,
		modelName, formFields.String(),
		modelName, modelName, tableColumns.String(), tableFilters.String(),
		modelName, modelName, modelName,
		modelName, infolistEntries.String(),
		modelName,
	))

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String(), nil
	}
	return string(formatted), nil
}

// GenerateRelationManagerCode scaffolds a relationship manager for an edge.
func GenerateRelationManagerCode(parentModel string, edge SchemaEdge) (string, error) {
	relName := strings.Title(edge.Name)
	targetModel := edge.TargetModel
	if targetModel == "" {
		targetModel = relName
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(`package relations

import (
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
)

type %sItem struct {
	ID string
}

type %sRelationManager struct {
	*filament.BaseRelationManager[*%sItem]
}

func New%sRelationManager() *%sRelationManager {
	rm := filament.NewRelationManager[*%sItem]("%s", "%s")
	rm.Table().Columns(
		tables.TextColumn[*%sItem]("id").Sortable(),
	)
	return &%sRelationManager{BaseRelationManager: rm}
}
`, relName, relName, relName, relName, relName, relName, edge.Name, relName, relName, relName))

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String(), nil
	}
	return string(formatted), nil
}
