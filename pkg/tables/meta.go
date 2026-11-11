package tables

// ActionDTO describes a row, header or bulk action to API clients.
type ActionDTO struct {
	Name                 string   `json:"name"`
	Label                string   `json:"label"`
	Icon                 string   `json:"icon,omitempty"`
	Color                string   `json:"color,omitempty"`
	RequiresConfirmation bool     `json:"requires_confirmation,omitempty"`
	ModalHeading         string   `json:"modal_heading,omitempty"`
	ModalDescription     string   `json:"modal_description,omitempty"`
	Formats              []string `json:"formats,omitempty"`
	// Custom marks actions run through the API (not view / edit / delete).
	Custom  bool `json:"custom,omitempty"`
	HasForm bool `json:"has_form,omitempty"`
}

// ToDTO describes the action.
func (a *TableAction[T]) ToDTO() ActionDTO {
	return ActionDTO{
		Name: a.name, Label: a.label, Icon: a.icon, Color: a.color,
		RequiresConfirmation: a.requiresConfirmation,
		ModalHeading:         a.modalHeading, ModalDescription: a.modalDescription,
		Custom:  a.runner != nil,
		HasForm: a.formBuilder != nil,
	}
}

// ToDTO describes the bulk action.
func (b *BulkActionDef) ToDTO() ActionDTO {
	return ActionDTO{
		Name: b.name, Label: b.label, Icon: b.icon, Color: b.color,
		RequiresConfirmation: b.requiresConfirmation, Formats: b.formats,
	}
}

// TableMeta lists the actions a table offers.
type TableMeta struct {
	Actions       []ActionDTO `json:"actions"`
	BulkActions   []ActionDTO `json:"bulk_actions"`
	HeaderActions []ActionDTO `json:"header_actions"`
	Reorderable   bool        `json:"reorderable,omitempty"`
}

// Meta describes the table's actions.
func (t *Table[T]) Meta() TableMeta {
	m := TableMeta{Actions: []ActionDTO{}, BulkActions: []ActionDTO{}, HeaderActions: []ActionDTO{}, Reorderable: t.IsReorderable()}
	for _, a := range t.GetActions() {
		m.Actions = append(m.Actions, a.ToDTO())
	}
	for _, a := range t.GetBulkActions() {
		m.BulkActions = append(m.BulkActions, a.ToDTO())
	}
	for _, a := range t.GetHeaderActions() {
		m.HeaderActions = append(m.HeaderActions, a.ToDTO())
	}
	return m
}

// columnDescriber lets a column add type-specific details to its DTO.
type columnDescriber interface {
	describeColumn(dto *ColumnDTO)
}

func (c *BaseColumn[T]) describeColumn(dto *ColumnDTO) {
	dto.Weight = c.weight
	dto.Icon = c.icon
	dto.Copyable = c.copyable
	dto.Editable = c.editable
}

func (c *TextColumnDef[T]) describeColumn(dto *ColumnDTO) {
	c.BaseColumn.describeColumn(dto)
	dto.DateTime = c.isDateTime
	dto.DateTimeFormat = c.dateTimeFmt
	dto.Currency = c.currency
	dto.Limit = c.limit
	switch {
	case c.isDateTime:
		dto.Type = "datetime"
	case c.currency != "" || c.isMoney:
		dto.Type = "money"
	}
}

func (c *BadgeColumnDef[T]) describeColumn(dto *ColumnDTO) {
	c.BaseColumn.describeColumn(dto)
	dto.Type = "badge"
	dto.Colors = c.colors
	dto.Options = c.labels
}

func (c *ToggleColumnDef[T]) describeColumn(dto *ColumnDTO) {
	c.BaseColumn.describeColumn(dto)
	dto.Type = "toggle"
	dto.Editable = true
}

func (c *IconColumnDef[T]) describeColumn(dto *ColumnDTO) {
	c.BaseColumn.describeColumn(dto)
	dto.Type = "icon"
}

func (c *ImageColumnDef[T]) describeColumn(dto *ColumnDTO) {
	c.BaseColumn.describeColumn(dto)
	dto.Type = "image"
	dto.Circular = c.isCircular
}

func (c *SelectColumnDef[T]) describeColumn(dto *ColumnDTO) {
	c.BaseColumn.describeColumn(dto)
	dto.Type = "select"
	dto.Editable = true
	dto.Options = c.options
}

func (c *TextInputColumnDef[T]) describeColumn(dto *ColumnDTO) {
	c.BaseColumn.describeColumn(dto)
	dto.Type = "text_input"
	dto.Editable = true
}

func (c *ColorColumnDef[T]) describeColumn(dto *ColumnDTO) {
	c.BaseColumn.describeColumn(dto)
	dto.Type = "color"
}
