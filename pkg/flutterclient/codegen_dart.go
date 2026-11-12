package flutterclient

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------- literals

// dartString quotes s as a single-quoted Dart string.
func dartString(s string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\'':
			b.WriteString(`\'`)
		case '$':
			b.WriteString(`\$`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f || r == 0x2028 || r == 0x2029 {
				fmt.Fprintf(&b, `\u{%x}`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// dartLiteral writes v (decoded JSON) as a const-compatible Dart literal with
// sorted map keys, one entry per line.
func dartLiteral(b *strings.Builder, v any, indent string) {
	in := indent + "  "
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case json.Number:
		b.WriteString(x.String())
	case float64:
		b.WriteString(fmt.Sprint(x))
	case string:
		b.WriteString(dartString(x))
	case []any:
		if len(x) == 0 {
			b.WriteString("<dynamic>[]")
			return
		}
		b.WriteString("<dynamic>[\n")
		for _, e := range x {
			b.WriteString(in)
			dartLiteral(b, e, in)
			b.WriteString(",\n")
		}
		b.WriteString(indent + "]")
	case map[string]any:
		if len(x) == 0 {
			b.WriteString("<String, dynamic>{}")
			return
		}
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString("<String, dynamic>{\n")
		for _, k := range keys {
			b.WriteString(in + dartString(k) + ": ")
			dartLiteral(b, x[k], in)
			b.WriteString(",\n")
		}
		b.WriteString(indent + "}")
	default:
		dartLiteral(b, toGeneric(x), indent)
	}
}

// ---------------------------------------------------------------- names

// words splits an identifier-ish string into words: "line_items",
// "lineItems", "LineItems" and "line-items" all give [line items]. Runs of
// capitals are kept together ("SKUCode" -> [SKU Code]).
func words(s string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			out = append(out, string(cur))
			cur = nil
		}
	}
	rs := []rune(s)
	for i, r := range rs {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if unicode.IsUpper(r) && len(cur) > 0 {
			prev := cur[len(cur)-1]
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				flush()
			}
		}
		cur = append(cur, r)
	}
	flush()
	return out
}

func capitalize(w string) string {
	rs := []rune(strings.ToLower(w))
	if len(rs) == 0 {
		return ""
	}
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
}

// pascal makes a Dart type name.
func pascal(s string) string {
	var b strings.Builder
	for _, w := range words(s) {
		b.WriteString(capitalize(w))
	}
	out := b.String()
	if out == "" {
		return "Value"
	}
	if unicode.IsDigit([]rune(out)[0]) {
		out = "N" + out
	}
	return out
}

// dartReserved are words that cannot be used as member names.
var dartReserved = map[string]bool{
	"assert": true, "break": true, "case": true, "catch": true, "class": true, "const": true,
	"continue": true, "default": true, "do": true, "else": true, "enum": true, "extends": true,
	"false": true, "final": true, "finally": true, "for": true, "if": true, "in": true, "is": true,
	"new": true, "null": true, "rethrow": true, "return": true, "super": true, "switch": true,
	"this": true, "throw": true, "true": true, "try": true, "var": true, "void": true, "while": true,
	"with": true, "await": true, "yield": true,
	// members every generated class has or inherits
	"raw": true, "extra": true, "toJson": true, "fromJson": true, "hashCode": true,
	"runtimeType": true, "toString": true, "noSuchMethod": true, "api": true,
}

// camel makes a Dart member name.
func camel(s string) string {
	ws := words(s)
	if len(ws) == 0 {
		return "value"
	}
	var b strings.Builder
	b.WriteString(strings.ToLower(ws[0]))
	for _, w := range ws[1:] {
		b.WriteString(capitalize(w))
	}
	out := b.String()
	if unicode.IsDigit([]rune(out)[0]) {
		out = "n" + out
	}
	if dartReserved[out] {
		out += "Value"
	}
	return out
}

// nameSet hands out unique names.
type nameSet map[string]bool

func (n nameSet) unique(name string) string {
	out := name
	for i := 2; n[out]; i++ {
		out = fmt.Sprintf("%s%d", name, i)
	}
	n[out] = true
	return out
}

// singular is a light English singularizer for class names ("Docs" -> "Doc").
func singular(s string) string {
	switch {
	case strings.HasSuffix(s, "ies") && len(s) > 3:
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(s, "sses"), strings.HasSuffix(s, "shes"), strings.HasSuffix(s, "ches"), strings.HasSuffix(s, "xes"):
		return s[:len(s)-2]
	case strings.HasSuffix(s, "ss"), strings.HasSuffix(s, "us"), strings.HasSuffix(s, "is"):
		return s
	case strings.HasSuffix(s, "s") && len(s) > 1:
		return s[:len(s)-1]
	}
	return s
}

// docComment renders text as /// lines.
func docComment(indent, text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		line = strings.TrimRight(line, " ")
		if line == "" {
			b.WriteString(indent + "///\n")
		} else {
			b.WriteString(indent + "/// " + line + "\n")
		}
	}
	return b.String()
}
