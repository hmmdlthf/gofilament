package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/payment"
)

// settingSep joins a driver and a setting name into a form field name, so
// two drivers can both have a "secret_key".
const settingSep = "__"

// SettingField is the form field holding a driver setting.
func SettingField(driver, setting string) string { return driver + settingSep + setting }

// GatewayRow is a configured gateway as the panel shows it. Its settings are
// flattened into "<driver>__<setting>" keys; secrets are never included.
type GatewayRow struct {
	ID          string
	Name        string
	Driver      string
	DriverLabel string
	Mode        payment.Mode
	Enabled     bool
	AutoCapture bool
	WebhookURL  string
	Features    string
	Settings    map[string]string
}

// MarshalJSON flattens the row for forms, tables and the API.
func (g *GatewayRow) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"id": g.ID, "name": g.Name, "driver": g.Driver, "driver_label": g.DriverLabel, "mode": string(g.Mode),
		"enabled": g.Enabled, "auto_capture": g.AutoCapture, "webhook_url": g.WebhookURL, "features": g.Features,
	}
	for k, v := range g.Settings {
		m[SettingField(g.Driver, k)] = v
	}
	return json.Marshal(m)
}

func featureList(d *payment.Driver) string {
	var out []string
	f := d.Features
	add := func(on bool, s string) {
		if on {
			out = append(out, s)
		}
	}
	add(f.Installments, "Pay later")
	add(f.Refunds, "Refunds")
	add(f.Capture, "Capture")
	add(f.Preapproval && f.Charging, "Saved cards")
	add(f.Subscriptions, "Subscriptions")
	add(f.StatusLookup, "Status sync")
	return strings.Join(out, ", ")
}

func gatewayRow(m *payment.Manager, g *payment.GatewayConfig) *GatewayRow {
	row := &GatewayRow{ID: g.ID, Name: g.Name, Driver: g.Driver, DriverLabel: g.Driver, Mode: g.Mode,
		Enabled: g.Enabled, AutoCapture: g.AutoCapture, WebhookURL: m.URL("webhooks/" + g.ID),
		Settings: map[string]string{}}
	d, ok := m.Registry().Driver(g.Driver)
	if ok {
		row.DriverLabel, row.Features = d.Label, featureList(d)
	}
	for k, v := range g.Settings {
		if ok {
			if f, found := d.Field(k); found && f.Secret() {
				continue // write-only
			}
		}
		row.Settings[k] = v
	}
	return row
}

// driverSettingsForm builds one section per installed driver, shown only
// while that driver is selected.
func driverSections(m *payment.Manager) []forms.Component {
	var out []forms.Component
	for _, d := range m.Drivers() {
		var fields []forms.Component
		for _, f := range d.Fields {
			name := SettingField(d.Name, f.Name)
			help := f.Help
			switch f.Type {
			case payment.FieldSecret:
				in := forms.TextInput(name).Label(f.Label).Password().Placeholder(f.Placeholder)
				if help != "" {
					help += " "
				}
				in.HelperText(help + "Leave blank to keep the saved value.")
				if f.Required {
					in.RequiredOnCreate()
				}
				fields = append(fields, in)
			case payment.FieldSelect:
				sel := forms.Select(name).Label(f.Label).Options(f.Options)
				if f.Default != "" {
					sel.Default(f.Default)
				}
				sel.HelperText(help)
				if f.Required {
					sel.Required()
				}
				fields = append(fields, sel)
			case payment.FieldToggle:
				tg := forms.Toggle(name).Label(f.Label).Default(f.Default == "true").HelperText(help)
				fields = append(fields, tg)
			case payment.FieldTextarea:
				ta := forms.Textarea(name).Label(f.Label).Placeholder(f.Placeholder)
				ta.HelperText(help)
				if f.Required {
					ta.Required()
				}
				fields = append(fields, ta)
			default:
				in := forms.TextInput(name).Label(f.Label).Placeholder(f.Placeholder).HelperText(help)
				if f.Default != "" {
					in.Default(f.Default)
				}
				if f.Required {
					in.Required()
				}
				fields = append(fields, in)
			}
		}
		desc := d.Description
		if d.Website != "" {
			desc += " " + d.Website
		}
		out = append(out, forms.Section(d.Label+" settings").Description(desc).Columns(2).
			VisibleWhen("driver", d.Name).Schema(fields...))
	}
	return out
}

func boolOf(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x == "1" || x == "true" || x == "on"
	}
	return false
}

func strOf(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

// configFromData turns submitted form data into a gateway config.
func configFromData(m *payment.Manager, id string, data map[string]any) (*payment.GatewayConfig, error) {
	g := &payment.GatewayConfig{
		ID: id, Driver: strOf(data["driver"]), Name: strOf(data["name"]),
		Enabled: boolOf(data["enabled"]), AutoCapture: boolOf(data["auto_capture"]),
		Mode: payment.Mode(strOf(data["mode"])), Settings: map[string]string{},
	}
	d, ok := m.Registry().Driver(g.Driver)
	if !ok {
		return nil, fmt.Errorf("choose a payment driver")
	}
	for _, f := range d.Fields {
		v, ok := data[SettingField(d.Name, f.Name)]
		if !ok {
			continue
		}
		if f.Type == payment.FieldToggle {
			g.Settings[f.Name] = fmt.Sprint(boolOf(v))
		} else {
			g.Settings[f.Name] = strOf(v)
		}
	}
	return g, nil
}

// fieldError reports a save problem against a form field.
func fieldError(field string, err error) error {
	return &forms.ValidationError{Errors: map[string]string{field: strings.TrimPrefix(err.Error(), "payment: ")}}
}

// GatewaysResource is the "Payment Gateways" resource.
func GatewaysResource(m *payment.Manager) *filament.BaseResource[*GatewayRow] {
	res := filament.NewResource[*GatewayRow]("payment-gateways").
		ModelName("Payment Gateway").
		PluralModelName("Payment Gateways").
		NavigationGroup(Group).
		NavigationIcon("heroicon-credit-card").
		NavigationSort(90).
		RecordID(func(g *GatewayRow) string { return g.ID })

	driverOptions := func() map[string]string {
		opts := map[string]string{}
		for _, d := range m.Drivers() {
			opts[d.Name] = d.Label
		}
		return opts
	}

	res.Table(func(t *tables.Table[*GatewayRow]) *tables.Table[*GatewayRow] {
		return t.Columns(
			tables.TextColumn[*GatewayRow]("name").Label("Name").Weight("bold").Searchable(),
			tables.TextColumn[*GatewayRow]("driver_label").Label("Gateway"),
			tables.BadgeColumn[*GatewayRow]("mode").Label("Mode").
				Colors(map[string]string{"live": "success", "test": "warning"}).
				Labels(map[string]string{"live": "Live", "test": "Test"}),
			tables.BadgeColumn[*GatewayRow]("enabled").Label("Status").
				Value(func(g *GatewayRow) any {
					if g.Enabled {
						return "enabled"
					}
					return "disabled"
				}).
				Colors(map[string]string{"enabled": "success", "disabled": "gray"}).
				Labels(map[string]string{"enabled": "Enabled", "disabled": "Disabled"}),
			tables.TextColumn[*GatewayRow]("features").Label("Supports").ToggledHiddenByDefault(true),
			tables.TextColumn[*GatewayRow]("webhook_url").Label("Webhook URL").Copyable().ToggledHiddenByDefault(true),
		).Actions(
			tables.EditAction[*GatewayRow](),
			tables.Action[*GatewayRow]("test").Label("Test connection").Icon("heroicon-signal").
				Handle(func(ctx context.Context, g *GatewayRow, _ map[string]any) error {
					err := m.TestGateway(ctx, g.ID)
					if errors.Is(err, payment.ErrNotSupported) {
						return fmt.Errorf("%s has no connection test; make a test payment instead", g.DriverLabel)
					}
					if err != nil {
						return fmt.Errorf("connection failed: %s", strings.TrimPrefix(err.Error(), "payment: "))
					}
					return nil
				}).
				SuccessMessage("Connected"),
			tables.DeleteAction[*GatewayRow](),
		)
	})

	res.Form(func(f *forms.Form) *forms.Form {
		schema := []forms.Component{
			forms.Section("Gateway").Columns(2).Schema(
				forms.Select("driver").Label("Payment gateway").Options(driverOptions()).Required(),
				forms.TextInput("id").Label("Key").Required().
					Placeholder("stripe").
					HelperText("Lowercase letters, digits, - and _. Used in the webhook URL: "+m.URL("webhooks/<key>")+". It cannot change later."),
				forms.TextInput("name").Label("Name at checkout").Placeholder("Card payment"),
				forms.Select("mode").Label("Mode").Options(map[string]string{"test": "Test (sandbox)", "live": "Live"}).
					Default("test").Required(),
				forms.Toggle("enabled").Label("Enabled").HelperText("Offer it to customers at checkout."),
				forms.Toggle("auto_capture").Label("Capture automatically").
					HelperText("For gateways that authorize first (Tabby, Tamara, manual-capture Stripe)."),
			),
		}
		return f.Schema(append(schema, driverSections(m)...)...)
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, _ map[string]string) ([]*GatewayRow, int, error) {
		gs, err := m.GatewayConfigs(ctx.GoContext())
		if err != nil {
			return nil, 0, err
		}
		q := strings.ToLower(strings.TrimSpace(search))
		var rows []*GatewayRow
		for _, g := range gs {
			row := gatewayRow(m, g)
			if q != "" && !strings.Contains(strings.ToLower(row.Name+" "+row.ID+" "+row.DriverLabel), q) {
				continue
			}
			rows = append(rows, row)
		}
		if sortCol == "name" {
			sort.SliceStable(rows, func(i, j int) bool { return (rows[i].Name < rows[j].Name) != (sortDir == "desc") })
		}
		return pageOf(rows, page, perPage), len(rows), nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*GatewayRow, error) {
		g, err := m.GatewayConfig(ctx.GoContext(), id)
		if err != nil {
			return nil, err
		}
		return gatewayRow(m, g), nil
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		id := strOf(data["id"])
		if _, err := m.GatewayConfig(ctx.GoContext(), id); err == nil {
			return fieldError("id", fmt.Errorf("a gateway with the key %q already exists", id))
		}
		g, err := configFromData(m, id, data)
		if err != nil {
			return fieldError("driver", err)
		}
		if err := m.SaveGateway(ctx.GoContext(), g); err != nil {
			return fieldError("id", err)
		}
		return nil
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		cur, err := m.GatewayConfig(ctx.GoContext(), id)
		if err != nil {
			return err
		}
		if nid := strOf(data["id"]); nid != "" && nid != id {
			return fieldError("id", errors.New("the key cannot change: gateways call back on it. Create a new gateway instead"))
		}
		// a partial update (the API, an inline edit) keeps what it leaves out
		full := map[string]any{"driver": cur.Driver, "name": cur.Name, "mode": string(cur.Mode),
			"enabled": cur.Enabled, "auto_capture": cur.AutoCapture}
		for k, v := range cur.Settings {
			full[SettingField(cur.Driver, k)] = v
		}
		for k, v := range data {
			full[k] = v
		}
		g, err := configFromData(m, id, full)
		if err != nil {
			return fieldError("driver", err)
		}
		g.Sort = cur.Sort
		if err := m.SaveGateway(ctx.GoContext(), g); err != nil {
			return fieldError("driver", err)
		}
		return nil
	})
	res.OnDelete(func(ctx *filament.Context, id string) error {
		return m.DeleteGateway(ctx.GoContext(), id)
	})
	return res
}

func pageOf[T any](rows []T, page, perPage int) []T {
	if perPage <= 0 {
		return rows
	}
	if page < 1 {
		page = 1
	}
	start := (page - 1) * perPage
	if start >= len(rows) {
		return nil
	}
	end := min(start+perPage, len(rows))
	return rows[start:end]
}
