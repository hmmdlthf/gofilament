package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golaravel/gofilament/pkg/codegen"
	"github.com/golaravel/gofilament/pkg/flutterclient"
	"github.com/golaravel/golaravel/pkg/artisan"
)

// RegisterFilamentCommands registers all filament code generator commands into the Artisan registry.
func RegisterFilamentCommands(reg *artisan.Registry) {
	reg.Register(MakeFilamentPanelCommand())
	reg.Register(MakeFilamentResourceCommand())
	reg.Register(MakeFilamentPageCommand())
	reg.Register(MakeFilamentWidgetCommand())
	reg.Register(MakeFilamentRelationCommand())
	reg.Register(MakeFilamentFlutterCommand())
	reg.Register(MakeFilamentFlutterAppCommand())
	reg.Register(MakeFilamentFlutterPageCommand())
}

// MakeFilamentPanelCommand creates a new admin panel configuration.
func MakeFilamentPanelCommand() *artisan.Command {
	return &artisan.Command{
		Name:        "make:filament-panel",
		Description: "Create a new GoFilament panel configuration (supports --engine=web|flutter)",
		Run: func(args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: artisan make:filament-panel <Name> [--engine=flutter]")
			}
			name := args[0]
			id := strings.ToLower(name)
			structName := strings.Title(id) + "Panel"

			isFlutter := false
			for _, arg := range args[1:] {
				if arg == "--engine=flutter" || arg == "--flutter" {
					isFlutter = true
				}
			}

			dir := filepath.Join("app", "filament", "panels")
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}

			filePath := filepath.Join(dir, fmt.Sprintf("%s_panel.go", id))
			engineChain := ""
			if isFlutter {
				engineChain = "\n\t\tEngine(filament.EngineFlutter)."
			}

			content := fmt.Sprintf(`package panels

import (
	"github.com/golaravel/gofilament/pkg/filament"
)

// %s creates and configures the %s panel.
func %s() *filament.Panel {
	return filament.NewPanel("%s").
		Path("/%s").%s
		BrandName("%s Portal").
		DarkMode(true).
		SidebarCollapsibleOnDesktop().
		MaxContentWidth(filament.Width7xl).
		GlobalSearch(true).
		DatabaseNotifications(true)
}
`, structName, name, structName, id, id, engineChain, strings.Title(id))

			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return err
			}
			fmt.Printf("Panel successfully created: %s\n", filePath)
			return nil
		},
	}
}

// MakeFilamentFlutterCommand scaffolds a responsive mobile-first Flutter admin client project.
func MakeFilamentFlutterCommand() *artisan.Command {
	return &artisan.Command{
		Name:        "make:filament-flutter",
		Description: "Scaffold a mobile-first responsive Flutter admin client for GoFilament",
		Run: func(args []string) error {
			name := "admin"
			targetDir := "client"
			if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
				name = args[0]
			}
			for _, a := range args {
				if strings.HasPrefix(a, "--dir=") {
					targetDir = strings.TrimPrefix(a, "--dir=")
				}
			}

			err := flutterclient.GenerateFlutterAdmin(targetDir, name, "")
			if err != nil {
				return fmt.Errorf("failed to generate Flutter admin client: %w", err)
			}
			fmt.Printf("Mobile-first responsive Flutter admin client created in: %s\n", targetDir)
			return nil
		},
	}
}

// MakeFilamentResourceCommand generates a typed Resource with Form and Table definitions.
func MakeFilamentResourceCommand() *artisan.Command {
	return &artisan.Command{
		Name:        "make:filament-resource",
		Description: "Generate a full Ent CRUD resource with Form and Table schemas",
		Run: func(args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: artisan make:filament-resource <Name> [--generate]")
			}
			name := args[0]
			autoGenerate := false
			for _, a := range args[1:] {
				if a == "--generate" {
					autoGenerate = true
				}
			}

			modelName := strings.Title(name)
			pluralName := modelName + "s"
			slug := strings.ToLower(pluralName)

			dir := filepath.Join("app", "filament", "resources")
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}

			filePath := filepath.Join(dir, fmt.Sprintf("%s_resource.go", strings.ToLower(name)))

			if autoGenerate {
				schemaPath, err := codegen.FindEntSchemaFile(name)
				if err != nil {
					return fmt.Errorf("failed to locate schema for %s: %w", name, err)
				}
				parsed, err := codegen.ParseEntSchema(schemaPath)
				if err != nil {
					return fmt.Errorf("failed to parse schema %s: %w", schemaPath, err)
				}
				content, err := codegen.GenerateResourceCode(parsed)
				if err != nil {
					return fmt.Errorf("failed to generate resource code: %w", err)
				}
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					return err
				}
				fmt.Printf("Resource generated from Ent schema (%s): %s\n", schemaPath, filePath)

				// Generate Relation Managers for edges
				for _, edge := range parsed.Edges {
					if edge.EdgeType == "To" && !edge.IsUnique {
						relDir := filepath.Join(dir, strings.ToLower(name), "relations")
						_ = os.MkdirAll(relDir, 0755)
						relFile := filepath.Join(relDir, fmt.Sprintf("%s_relation_manager.go", strings.ToLower(edge.Name)))
						relContent, _ := codegen.GenerateRelationManagerCode(modelName, edge)
						_ = os.WriteFile(relFile, []byte(relContent), 0644)
						fmt.Printf("Relation manager generated for edge %s: %s\n", edge.Name, relFile)
					}
				}
				return nil
			}

			content := fmt.Sprintf(`package resources

import (
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

type %s struct {
	ID        string
	Name      string
	Email     string
	Status    string
	CreatedAt string
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
				forms.Section("General Details").
					Columns(2).
					Schema(
						forms.TextInput("name").Required().Placeholder("Enter name..."),
						forms.TextInput("email").Email().Required().Placeholder("user@example.com"),
						forms.Select("status").Options(map[string]string{
							"active":   "Active",
							"inactive": "Inactive",
						}).Default("active"),
					),
			)
		}).
		Table(func(table *tables.Table[*%s]) *tables.Table[*%s] {
			return table.
				Columns(
					tables.TextColumn[*%s]("id").Sortable(),
					tables.TextColumn[*%s]("name").Searchable().Sortable().Weight("bold"),
					tables.TextColumn[*%s]("email").Searchable().Copyable(),
					tables.BadgeColumn[*%s]("status").Colors(map[string]string{
						"active":   "success",
						"inactive": "danger",
					}),
					tables.TextColumn[*%s]("created_at").DateTime().Sortable(),
				).
				Actions(
					tables.ViewAction[*%s](),
					tables.EditAction[*%s](),
					tables.DeleteAction[*%s](),
				).
				BulkActions(
					tables.DeleteBulkAction(),
				)
		})

	return &%sResource{BaseResource: res}
}
`, modelName, modelName, modelName, modelName, modelName, modelName, slug, modelName, pluralName,
				modelName, modelName, modelName, modelName, modelName, modelName, modelName,
				modelName, modelName, modelName, modelName)

			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return err
			}
			fmt.Printf("Resource successfully created: %s\n", filePath)
			return nil
		},
	}
}

// MakeFilamentPageCommand creates a custom panel page.
func MakeFilamentPageCommand() *artisan.Command {
	return &artisan.Command{
		Name:        "make:filament-page",
		Description: "Create a custom panel page",
		Run: func(args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: artisan make:filament-page <Name>")
			}
			name := args[0]
			title := strings.Title(name)
			slug := strings.ToLower(name)

			dir := filepath.Join("app", "filament", "pages")
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}

			filePath := filepath.Join(dir, fmt.Sprintf("%s.go", slug))
			content := fmt.Sprintf(`package pages

import (
	"fmt"
	"net/http"

	"github.com/golaravel/gofilament/pkg/filament"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
)

type %sPage struct {
	slug  string
	title string
}

func New%sPage() *%sPage {
	return &%sPage{
		slug:  "%s",
		title: "%s",
	}
}

func (p *%sPage) ID() string    { return p.slug }
func (p *%sPage) Title() string { return p.title }
func (p *%sPage) Slug() string  { return p.slug }

func (p *%sPage) NavigationItem(ctx *filament.Context) *filament.NavigationItem {
	return &filament.NavigationItem{
		Label: p.title,
		Icon:  "heroicon-document-text",
	}
}

func (p *%sPage) RegisterRoutes(panel *filament.Panel, r *router.Router) {
	path := fmt.Sprintf("%%s/%%s", panel.GetPath(), p.slug)
	r.GET(path, func(c *gcontext.Context) error {
		ctx := filament.NewContext(c, panel)
		return c.HTML(http.StatusOK, fmt.Sprintf("<h1>%%s</h1>", p.title))
	})
}
`, title, title, title, title, slug, title, title, title, title, title, title)

			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return err
			}
			fmt.Printf("Page successfully created: %s\n", filePath)
			return nil
		},
	}
}

// MakeFilamentWidgetCommand creates a dashboard KPI or chart widget.
func MakeFilamentWidgetCommand() *artisan.Command {
	return &artisan.Command{
		Name:        "make:filament-widget",
		Description: "Create a KPI stats or chart widget",
		Run: func(args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: artisan make:filament-widget <Name>")
			}
			name := args[0]
			title := strings.Title(name)
			slug := strings.ToLower(name)

			dir := filepath.Join("app", "filament", "widgets")
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}

			filePath := filepath.Join(dir, fmt.Sprintf("%s_widget.go", slug))
			content := fmt.Sprintf(`package widgets

import (
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/widgets"
)

type %sWidget struct {
	*widgets.StatsOverviewWidget
}

func New%sWidget() *%sWidget {
	w := widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
		return []widgets.Stat{
			*widgets.NewStat("Total Signups", "1,240").
				Description("12%% increase").
				Color("success").
				Chart([]int{10, 20, 35, 45, 60, 95}),
		}
	})
	return &%sWidget{StatsOverviewWidget: w}
}
`, title, title, title, title)

			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return err
			}
			fmt.Printf("Widget successfully created: %s\n", filePath)
			return nil
		},
	}
}

// MakeFilamentRelationCommand creates a relationship manager.
func MakeFilamentRelationCommand() *artisan.Command {
	return &artisan.Command{
		Name:        "make:filament-relation",
		Description: "Generate an interactive relationship manager",
		Run: func(args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("usage: artisan make:filament-relation <Resource> <Relation>")
			}
			resName := strings.ToLower(args[0])
			relName := args[1]
			relTitle := strings.Title(relName)

			dir := filepath.Join("app", "filament", "resources", resName, "relations")
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}

			filePath := filepath.Join(dir, fmt.Sprintf("%s_relation_manager.go", strings.ToLower(relName)))
			content := fmt.Sprintf(`package relations

import (
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
)

type %sItem struct {
	ID    string
	Title string
}

type %sRelationManager struct {
	*filament.BaseRelationManager[*%sItem]
}

func New%sRelationManager() *%sRelationManager {
	rm := filament.NewRelationManager[*%sItem]("%s", "%s")
	rm.Table().Columns(
		tables.TextColumn[*%sItem]("id"),
		tables.TextColumn[*%sItem]("title").Searchable(),
	)
	return &%sRelationManager{BaseRelationManager: rm}
}
`, relTitle, relTitle, relTitle, relTitle, relTitle, relTitle, strings.ToLower(relName), relTitle,
				relTitle, relTitle, relTitle)

			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return err
			}
			fmt.Printf("Relation manager successfully created: %s\n", filePath)
			return nil
		},
	}
}
