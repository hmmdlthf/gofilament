package filament

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/forms"
)

// Import: CSV, TSV and Excel (.xlsx) files become records through the
// resource's create path (form validation, CanCreate, OnCreate). A preview
// (dry run) maps the file's columns to form fields and reports every row's
// validation errors before anything is saved.

// MaxImportBytes is the largest import file accepted.
const MaxImportBytes = 10 << 20

// ImportRowError describes one row that was not imported.
type ImportRowError struct {
	Row     int               `json:"row"`
	Message string            `json:"message,omitempty"`
	Errors  map[string]string `json:"errors,omitempty"`
}

// ImportField is a form field rows can be imported into.
type ImportField struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Type     string `json:"type,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// ImportPreviewRow is one row of an import preview.
type ImportPreviewRow struct {
	Row    int               `json:"row"`
	Values map[string]string `json:"values"`
	Errors map[string]string `json:"errors,omitempty"`
}

// ImportPreview is the result of a dry run.
type ImportPreview struct {
	Headers []string `json:"headers"`
	// Mapping maps each header to a field name ("" = not imported).
	Mapping map[string]string  `json:"mapping"`
	Fields  []ImportField      `json:"fields"`
	Rows    []ImportPreviewRow `json:"rows"`
	Total   int                `json:"total"`
	Valid   int                `json:"valid"`
	Invalid int                `json:"invalid"`
	// Problems lists the rows that would fail (at most 200).
	Problems []ImportRowError `json:"problems,omitempty"`
}

const (
	importPreviewRows = 20
	importMaxProblems = 200
)

// readImportTable reads a CSV, TSV or .xlsx file into rows of cells; the
// first row holds the headers.
func readImportTable(rdr io.Reader) ([][]string, error) {
	raw, err := io.ReadAll(io.LimitReader(rdr, MaxImportBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxImportBytes {
		return nil, fmt.Errorf("the file is larger than %d MB", MaxImportBytes>>20)
	}
	if bytes.HasPrefix(raw, []byte("PK\x03\x04")) {
		return readXLSX(raw)
	}
	raw = bytes.TrimPrefix(raw, []byte("\xEF\xBB\xBF"))
	first := raw
	if i := bytes.IndexByte(raw, '\n'); i >= 0 {
		first = raw[:i]
	}
	r := csv.NewReader(bytes.NewReader(raw))
	if bytes.Count(first, []byte("\t")) > bytes.Count(first, []byte(",")) {
		r.Comma = '\t'
	} else if bytes.Count(first, []byte(";")) > bytes.Count(first, []byte(",")) {
		r.Comma = ';'
	}
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("the file could not be read: %w", err)
	}
	if len(rows) == 0 {
		return nil, errors.New("the file is empty")
	}
	return rows, nil
}

// readXLSX reads the first worksheet of an .xlsx file.
func readXLSX(raw []byte) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, errors.New("the Excel file could not be read")
	}
	files := map[string]*zip.File{}
	var sheets []string
	for _, f := range zr.File {
		files[f.Name] = f
		if strings.HasPrefix(f.Name, "xl/worksheets/") && path.Ext(f.Name) == ".xml" {
			sheets = append(sheets, f.Name)
		}
	}
	read := func(name string) ([]byte, error) {
		f := files[name]
		if f == nil {
			return nil, nil
		}
		if f.UncompressedSize64 > 8*MaxImportBytes {
			return nil, errors.New("the Excel file is too large")
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, 8*MaxImportBytes))
	}
	// The first sheet in workbook order, else the first file.
	sheet := ""
	if wb, _ := read("xl/workbook.xml"); wb != nil {
		var w struct {
			Sheets []struct {
				RID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
			} `xml:"sheets>sheet"`
		}
		var rels struct {
			Rel []struct {
				ID     string `xml:"Id,attr"`
				Target string `xml:"Target,attr"`
			} `xml:"Relationship"`
		}
		relRaw, _ := read("xl/_rels/workbook.xml.rels")
		if xml.Unmarshal(wb, &w) == nil && xml.Unmarshal(relRaw, &rels) == nil && len(w.Sheets) > 0 {
			for _, rel := range rels.Rel {
				if rel.ID == w.Sheets[0].RID {
					sheet = path.Join("xl", strings.TrimPrefix(rel.Target, "/xl/"))
				}
			}
		}
	}
	if files[sheet] == nil {
		sort.Strings(sheets)
		if len(sheets) == 0 {
			return nil, errors.New("the Excel file has no worksheet")
		}
		sheet = sheets[0]
	}

	var shared []string
	if ss, _ := read("xl/sharedStrings.xml"); ss != nil {
		var sst struct {
			SI []struct {
				T string `xml:"t"`
				R []struct {
					T string `xml:"t"`
				} `xml:"r"`
			} `xml:"si"`
		}
		if err := xml.Unmarshal(ss, &sst); err != nil {
			return nil, errors.New("the Excel file could not be read")
		}
		for _, si := range sst.SI {
			s := si.T
			for _, r := range si.R {
				s += r.T
			}
			shared = append(shared, s)
		}
	}

	data, err := read(sheet)
	if err != nil || data == nil {
		return nil, errors.New("the Excel file could not be read")
	}
	var ws struct {
		Rows []struct {
			C []struct {
				R  string `xml:"r,attr"`
				T  string `xml:"t,attr"`
				V  string `xml:"v"`
				IS struct {
					T string `xml:"t"`
					R []struct {
						T string `xml:"t"`
					} `xml:"r"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(data, &ws); err != nil {
		return nil, errors.New("the Excel file could not be read")
	}
	var rows [][]string
	for _, row := range ws.Rows {
		var cells []string
		for i, c := range row.C {
			col := i
			if c.R != "" {
				col = xlsxColumn(c.R)
			}
			v := c.V
			switch c.T {
			case "s":
				if n, err := strconv.Atoi(c.V); err == nil && n >= 0 && n < len(shared) {
					v = shared[n]
				}
			case "inlineStr":
				v = c.IS.T
				for _, r := range c.IS.R {
					v += r.T
				}
			case "b":
				v = map[string]string{"1": "true", "0": "false"}[c.V]
			}
			for len(cells) <= col {
				cells = append(cells, "")
			}
			cells[col] = v
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return nil, errors.New("the Excel sheet is empty")
	}
	return rows, nil
}

// xlsxColumn turns a cell reference ("C7") into a 0-based column index.
func xlsxColumn(ref string) int {
	n := 0
	for _, ch := range ref {
		if ch < 'A' || ch > 'Z' {
			break
		}
		n = n*26 + int(ch-'A'+1)
	}
	return n - 1
}

// excelSerialDate turns an Excel day number into "2006-01-02" (and the
// time, for datetime fields).
func excelSerialDate(v string, withTime bool) (string, bool) {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 1 || f > 2958465 {
		return "", false
	}
	t := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).Add(time.Duration(math.Round(f*86400)) * time.Second)
	if withTime {
		return t.Format("2006-01-02 15:04:05"), true
	}
	return t.Format("2006-01-02"), true
}

// importFields lists the form fields rows can fill.
func (r *BaseResource[T]) importFields(ctx *Context) []ImportField {
	var out []ImportField
	var walk func(ns []forms.Node)
	walk = func(ns []forms.Node) {
		for _, n := range ns {
			if n.Kind == "field" && n.Name != "" && !n.Disabled {
				switch n.Type {
				case "file", "image", "key_value", "hidden", "password":
					continue
				}
				label := n.Label
				if label == "" {
					label = n.Name
				}
				out = append(out, ImportField{Name: n.Name, Label: label, Type: n.Type, Required: n.Required || n.RequiredOnCreate})
				continue
			}
			if n.Kind == "repeater" {
				continue
			}
			walk(n.Children)
		}
	}
	walk(r.DescribeForm(ctx))
	return out
}

// importMapping decides which field each header fills. explicit maps a
// header (as written, or normalized) to a field; "" skips the column.
func importMapping(headers []string, fields []ImportField, explicit map[string]string) map[string]string {
	known := map[string]string{}
	for _, f := range fields {
		known[f.Name] = f.Name
		known[normalizeHeader(f.Name)] = f.Name
		known[normalizeHeader(f.Label)] = f.Name
	}
	out := map[string]string{}
	for _, h := range headers {
		norm := normalizeHeader(h)
		if v, ok := explicit[h]; ok {
			out[h] = v
		} else if v, ok := explicit[norm]; ok {
			out[h] = v
		} else if len(fields) == 0 {
			out[h] = norm // no form: the header names the field
		} else {
			out[h] = known[norm]
		}
		if out[h] != "" && len(fields) > 0 && known[out[h]] == "" {
			out[h] = "" // only form fields can be filled
		}
	}
	return out
}

// importRow builds the data of one row.
func importRow(headers []string, cells []string, mapping map[string]string, types map[string]string, xlsx bool) map[string]any {
	data := map[string]any{}
	for i, h := range headers {
		field := mapping[h]
		if field == "" || i >= len(cells) {
			continue
		}
		v := strings.TrimSpace(cells[i])
		if xlsx && v != "" {
			switch types[field] {
			case "date":
				if d, ok := excelSerialDate(v, false); ok {
					v = d
				}
			case "datetime":
				if d, ok := excelSerialDate(v, true); ok {
					v = d
				}
			}
		}
		if prev, ok := data[field]; ok && v == "" {
			_ = prev
			continue
		}
		data[field] = v
	}
	return data
}

func blankRow(cells []string) bool {
	for _, c := range cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// runImport previews (dryRun) or imports rows.
func (r *BaseResource[T]) runImport(ctx *Context, rows [][]string, explicit map[string]string, dryRun bool, xlsx bool) (*ImportPreview, *ImportResult) {
	headers := rows[0]
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
		if headers[i] == "" {
			headers[i] = fmt.Sprintf("Column %d", i+1)
		}
	}
	fields := r.importFields(ctx)
	mapping := importMapping(headers, fields, explicit)
	types := map[string]string{}
	for _, f := range fields {
		types[f.Name] = f.Type
	}
	preview := &ImportPreview{Headers: headers, Mapping: mapping, Fields: fields, Rows: []ImportPreviewRow{}}
	result := &ImportResult{}
	for i, cells := range rows[1:] {
		rowNum := i + 2 // the header is row 1
		if blankRow(cells) {
			continue
		}
		preview.Total++
		data := importRow(headers, cells, mapping, types, xlsx)
		var rowErr *ImportRowError
		clean, err := r.validateInput(ctx, data, true)
		if err == nil && !dryRun {
			err = r.onCreate(ctx, clean)
		}
		if err != nil {
			rowErr = &ImportRowError{Row: rowNum}
			var verr *forms.ValidationError
			if errors.As(err, &verr) {
				rowErr.Errors = verr.Errors
			} else {
				rowErr.Message = err.Error()
			}
		}
		if rowErr == nil {
			preview.Valid++
			result.SuccessCount++
		} else {
			preview.Invalid++
			result.FailedCount++
			if len(preview.Problems) < importMaxProblems {
				preview.Problems = append(preview.Problems, *rowErr)
				result.Rows = append(result.Rows, *rowErr)
				result.Errors = append(result.Errors, fmt.Sprintf("Row %d: %s", rowNum, rowErr.describe()))
			}
		}
		if len(preview.Rows) < importPreviewRows {
			values := map[string]string{}
			for k, v := range data {
				values[k] = fmt.Sprint(v)
			}
			pr := ImportPreviewRow{Row: rowNum, Values: values}
			if rowErr != nil {
				pr.Errors = rowErr.Errors
				if rowErr.Message != "" {
					pr.Errors = map[string]string{"": rowErr.Message}
				}
			}
			preview.Rows = append(preview.Rows, pr)
		}
	}
	return preview, result
}

func (e ImportRowError) describe() string {
	if e.Message != "" {
		return e.Message
	}
	keys := make([]string, 0, len(e.Errors))
	for k := range e.Errors {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+": "+e.Errors[k])
	}
	return strings.Join(parts, "; ")
}

// PreviewImport is a dry run: it maps the file's columns and validates every
// row without saving anything.
func (r *BaseResource[T]) PreviewImport(ctx *Context, rdr io.Reader, mapping map[string]string) (*ImportPreview, error) {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return nil, ErrForbidden
	}
	raw, err := io.ReadAll(io.LimitReader(rdr, MaxImportBytes+1))
	if err != nil {
		return nil, err
	}
	rows, err := readImportTable(bytes.NewReader(raw))
	if err != nil {
		return nil, &forms.ValidationError{Errors: map[string]string{"file": err.Error()}}
	}
	preview, _ := r.runImport(ctx, rows, mapping, true, bytes.HasPrefix(raw, []byte("PK\x03\x04")))
	return preview, nil
}
