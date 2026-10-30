package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/commands"
)

func TestEntAutoScaffoldingGenerator(t *testing.T) {
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Create mock Ent schema file in app/schema/invoice.go
	schemaDir := filepath.Join("app", "schema")
	_ = os.MkdirAll(schemaDir, 0755)

	mockSchema := `package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Invoice struct {
	ent.Schema
}

func (Invoice) Fields() []ent.Field {
	return []ent.Field{
		field.String("invoice_number").Unique().NotEmpty(),
		field.String("customer_name").NotEmpty(),
		field.String("customer_email").NotEmpty(),
		field.Enum("status").Values("draft", "sent", "paid").Default("draft"),
		field.Float("amount").Default(0),
		field.Time("due_date").Default(time.Now),
		field.Bool("is_archived").Default(false),
		field.String("secret_notes").Sensitive(),
	}
}

func (Invoice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("items", LineItem.Type),
	}
}
`
	if err := os.WriteFile(filepath.Join(schemaDir, "invoice.go"), []byte(mockSchema), 0644); err != nil {
		t.Fatalf("Failed to create mock schema: %v", err)
	}

	// Run artisan make:filament-resource Invoice --generate
	cmd := commands.MakeFilamentResourceCommand()
	if err := cmd.Run([]string{"Invoice", "--generate"}); err != nil {
		t.Fatalf("Command make:filament-resource --generate failed: %v", err)
	}

	// 1. Verify generated resource file
	resPath := filepath.Join("app", "filament", "resources", "invoice_resource.go")
	resBytes, err := os.ReadFile(resPath)
	if err != nil {
		t.Fatalf("Expected resource file %s to exist: %v", resPath, err)
	}
	content := string(resBytes)

	// Check Form schema
	if !strings.Contains(content, `forms.TextInput("invoice_number").Required()`) {
		t.Errorf("Expected invoice_number text input")
	}
	if !strings.Contains(content, `forms.TextInput("customer_email").Email().Required()`) {
		t.Errorf("Expected customer_email text input with Email() validator, got:\n%s", content)
	}
	if !strings.Contains(content, `forms.Select("status").Options(map[string]string{`) {
		t.Errorf("Expected status select input with options")
	}
	if !strings.Contains(content, `forms.Toggle("is_archived")`) {
		t.Errorf("Expected is_archived toggle input")
	}
	if !strings.Contains(content, `forms.TextInput("secret_notes").Password().RequiredOnCreate().Hash()`) {
		t.Errorf("Expected secret_notes password input with hash")
	}

	// Check Table schema
	if !strings.Contains(content, `tables.TextColumn[*Invoice]("invoice_number").Searchable().Sortable().Weight("bold")`) {
		t.Errorf("Expected invoice_number column to be searchable, sortable and bold")
	}
	if !strings.Contains(content, `tables.BadgeColumn[*Invoice]("status")`) {
		t.Errorf("Expected status badge column")
	}
	if !strings.Contains(content, `tables.TextColumn[*Invoice]("amount").Money("USD").Sortable()`) {
		t.Errorf("Expected amount money column")
	}
	if !strings.Contains(content, `tables.IconColumn[*Invoice]("is_archived").Boolean()`) {
		t.Errorf("Expected is_archived boolean icon column")
	}

	// Check Filters
	if !strings.Contains(content, `tables.SelectFilter("status")`) {
		t.Errorf("Expected select filter for status enum")
	}
	if !strings.Contains(content, `tables.TernaryFilter("is_archived")`) {
		t.Errorf("Expected ternary filter for is_archived boolean")
	}

	// 2. Verify generated Relation Manager for edge.To("items", LineItem.Type)
	relPath := filepath.Join("app", "filament", "resources", "invoice", "relations", "items_relation_manager.go")
	relBytes, err := os.ReadFile(relPath)
	if err != nil {
		t.Fatalf("Expected relation manager file %s to exist: %v", relPath, err)
	}
	relContent := string(relBytes)
	if !strings.Contains(relContent, `type ItemsRelationManager struct`) {
		t.Errorf("Expected ItemsRelationManager struct in generated relation manager")
	}
}
