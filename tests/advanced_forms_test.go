package tests

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/forms"
)

func TestFileUploadAndImageUpload(t *testing.T) {
	// Standard File Upload
	fileField := forms.FileUpload("attachment").
		Label("Invoice Attachment").
		AcceptedFileTypes("application/pdf", "image/png").
		MaxSize(5120).
		Required()

	html := fileField.Render(context.Background(), nil, nil)

	if !strings.Contains(html, `accept="application/pdf,image/png"`) {
		t.Errorf("Expected accept attribute for pdf and png, got:\n%s", html)
	}
	if !strings.Contains(html, "Invoice Attachment") {
		t.Errorf("Expected label 'Invoice Attachment'")
	}
	if !strings.Contains(html, "Click to upload") || !strings.Contains(html, "drag and drop") {
		t.Errorf("Expected drag and drop prompt")
	}
	if !strings.Contains(html, "5120 KB") {
		t.Errorf("Expected max size hint '5120 KB'")
	}

	// Avatar Image Upload
	avatarField := forms.ImageUpload("avatar").
		Label("Profile Picture").
		Avatar().
		HelperText("Square image recommended")

	avatarHTML := avatarField.Render(context.Background(), map[string]any{
		"avatar": "/storage/avatars/user_1.png",
	}, nil)

	if !strings.Contains(avatarHTML, "file-upload-avatar-wrapper") {
		t.Errorf("Expected avatar wrapper, got:\n%s", avatarHTML)
	}
	if !strings.Contains(avatarHTML, "/storage/avatars/user_1.png") {
		t.Errorf("Expected existing avatar image URL")
	}
	if !strings.Contains(avatarHTML, "Square image recommended") {
		t.Errorf("Expected helper text")
	}
	if !strings.Contains(avatarHTML, "border-radius: 9999px") {
		t.Errorf("Expected circular avatar border radius")
	}
}

func TestRichEditorAndMarkdownEditor(t *testing.T) {
	// Rich Text Editor
	richEditor := forms.RichEditor("body").
		Label("Article Content").
		MinHeight("300px").
		Required()

	richHTML := richEditor.Render(context.Background(), map[string]any{
		"body": "<p>Hello <strong>World</strong>!</p>",
	}, nil)

	if !strings.Contains(richHTML, "rich-editor-wrapper") {
		t.Errorf("Expected rich-editor-wrapper, got:\n%s", richHTML)
	}
	if !strings.Contains(richHTML, "contenteditable=\"true\"") {
		t.Errorf("Expected contenteditable area")
	}
	if !strings.Contains(richHTML, "min-height: 300px") {
		t.Errorf("Expected custom min-height 300px")
	}
	if !strings.Contains(richHTML, "<p>Hello <strong>World</strong>!</p>") {
		t.Errorf("Expected initial HTML content rendered")
	}
	if !strings.Contains(richHTML, `title="Bold"`) || !strings.Contains(richHTML, `title="Italic"`) {
		t.Errorf("Expected toolbar buttons for formatting")
	}

	// Markdown Editor
	mdEditor := forms.MarkdownEditor("notes").
		Label("Release Notes").
		Rows(10).
		Previewable(true)

	mdHTML := mdEditor.Render(context.Background(), map[string]any{
		"notes": "## Features\n- New Search\n- Faster rendering",
	}, nil)

	if !strings.Contains(mdHTML, "markdown-editor-wrapper") {
		t.Errorf("Expected markdown-editor-wrapper, got:\n%s", mdHTML)
	}
	if !strings.Contains(mdHTML, "tab === 'write'") || !strings.Contains(mdHTML, "tab === 'preview'") {
		t.Errorf("Expected Write and Preview tab toggles")
	}
	if !strings.Contains(mdHTML, "rows=\"10\"") {
		t.Errorf("Expected rows=\"10\" on textarea")
	}
	if !strings.Contains(mdHTML, "renderMarkdown") {
		t.Errorf("Expected client markdown parser function")
	}
}

func TestRepeaterComponent(t *testing.T) {
	repeater := forms.Repeater("line_items").
		Label("Order Line Items").
		ItemLabel("Item").
		Collapsible(true).
		Reorderable(true).
		Grid(2).
		Schema(
			forms.TextInput("title").Label("Item Name"),
			forms.TextInput("price").Label("Unit Price"),
		)

	// Render with existing data
	existingData := map[string]any{
		"line_items": []map[string]any{
			{"title": "Mechanical Keyboard", "price": "149.00"},
			{"title": "Ergonomic Mouse", "price": "79.00"},
		},
	}

	html := repeater.Render(context.Background(), existingData, nil)

	if !strings.Contains(html, "repeater-wrapper") {
		t.Errorf("Expected repeater-wrapper, got:\n%s", html)
	}
	if !strings.Contains(html, "Item #1") || !strings.Contains(html, "Item #2") {
		t.Errorf("Expected item titles 'Item #1' and 'Item #2'")
	}
	if !strings.Contains(html, "Mechanical Keyboard") || !strings.Contains(html, "149.00") {
		t.Errorf("Expected filled state for Mechanical Keyboard and price")
	}
	if !strings.Contains(html, `name="line_items[0][title]"`) {
		t.Errorf("Expected scoped name 'line_items[0][title]'")
	}
	if !strings.Contains(html, `name="line_items[1][price]"`) {
		t.Errorf("Expected scoped name 'line_items[1][price]'")
	}
	// Verify reorderable controls exist
	if !strings.Contains(html, "title=\"Move Up\"") || !strings.Contains(html, "title=\"Move Down\"") {
		t.Errorf("Expected move up and move down buttons")
	}
	// Verify grid layout inside row
	if !strings.Contains(html, "grid-template-columns: repeat(2, minmax(0, 1fr))") {
		t.Errorf("Expected 2-column grid inside repeater row")
	}
	// Verify dynamic template exists
	if !strings.Contains(html, "<template id=\"tmpl_line_items\">") {
		t.Errorf("Expected template element for adding new rows")
	}
}

func TestTagsInputAndColorPicker(t *testing.T) {
	// TagsInput
	tags := forms.TagsInput("skills").
		Label("Developer Skills").
		Suggestions("Go", "Laravel", "Vue", "Flutter", "Docker").
		MaxTags(5)

	tagsHTML := tags.Render(context.Background(), map[string]any{
		"skills": []string{"Go", "Docker"},
	}, nil)

	if !strings.Contains(tagsHTML, "tags-input-wrapper") {
		t.Errorf("Expected tags-input-wrapper, got:\n%s", tagsHTML)
	}
	if !strings.Contains(tagsHTML, `"Go","Docker"`) {
		t.Errorf("Expected initial tags JSON, got:\n%s", tagsHTML)
	}
	if !strings.Contains(tagsHTML, "Suggested:") {
		t.Errorf("Expected suggestions section")
	}
	if !strings.Contains(tagsHTML, "addTag") || !strings.Contains(tagsHTML, "removeTag") {
		t.Errorf("Expected addTag and removeTag methods")
	}

	// ColorPicker
	color := forms.ColorPicker("brand_color").
		Label("Brand Color").
		Presets("#6366f1", "#10b981", "#f59e0b", "#ef4444")

	colorHTML := color.Render(context.Background(), map[string]any{
		"brand_color": "#10b981",
	}, nil)

	if !strings.Contains(colorHTML, "color-picker-wrapper") {
		t.Errorf("Expected color-picker-wrapper, got:\n%s", colorHTML)
	}
	if !strings.Contains(colorHTML, "#10b981") {
		t.Errorf("Expected active color value #10b981")
	}
	if !strings.Contains(colorHTML, `type="color"`) {
		t.Errorf("Expected native color input")
	}
	if !strings.Contains(colorHTML, "maxlength=\"7\"") {
		t.Errorf("Expected hex text input")
	}
}

func TestFormMultipartDetectionAndParseFormData(t *testing.T) {
	// 1. Verify Form automatically adds enctype="multipart/form-data" when FileUpload is used
	formWithFile := forms.New().
		Schema(
			forms.TextInput("name"),
			forms.FileUpload("avatar"),
		)

	html := formWithFile.Render(context.Background())
	if !strings.Contains(html, `enctype="multipart/form-data"`) {
		t.Errorf("Expected multipart enctype on form with FileUpload, got:\n%s", html)
	}

	formWithoutFile := forms.New().
		Schema(
			forms.TextInput("name"),
			forms.TextInput("email"),
		)

	htmlNoFile := formWithoutFile.Render(context.Background())
	if strings.Contains(htmlNoFile, `enctype="multipart/form-data"`) {
		t.Errorf("Did not expect multipart enctype on form without FileUpload, got:\n%s", htmlNoFile)
	}

	// 2. Verify ParseFormData converts bracket notation into nested repeaters
	postValues := url.Values{
		"customer_name":            []string{"Acme Corp"},
		"status":                   []string{"active"},
		"tags[]":                   []string{"enterprise", "b2b"},
		"line_items[0][title]":     []string{"Product A"},
		"line_items[0][qty]":       []string{"10"},
		"line_items[0][price]":     []string{"25.50"},
		"line_items[1][title]":     []string{"Product B"},
		"line_items[1][qty]":       []string{"2"},
		"line_items[1][price]":     []string{"99.00"},
	}

	parsed := forms.ParseFormData(postValues)

	if parsed["customer_name"] != "Acme Corp" {
		t.Errorf("Expected customer_name 'Acme Corp', got: %v", parsed["customer_name"])
	}

	tags, ok := parsed["tags"].([]string)
	if !ok || len(tags) != 2 || tags[0] != "enterprise" {
		t.Errorf("Expected tags slice [enterprise, b2b], got: %v", parsed["tags"])
	}

	items, ok := parsed["line_items"].([]map[string]any)
	if !ok {
		t.Fatalf("Expected line_items to be parsed as []map[string]any, got: %T (%v)", parsed["line_items"], parsed["line_items"])
	}

	if len(items) != 2 {
		t.Fatalf("Expected 2 line items, got: %d", len(items))
	}

	if items[0]["title"] != "Product A" || items[0]["price"] != "25.50" {
		t.Errorf("Item 0 mismatch: %v", items[0])
	}
	if items[1]["title"] != "Product B" || items[1]["qty"] != "2" {
		t.Errorf("Item 1 mismatch: %v", items[1])
	}
}
