package tables

import (
	"net/url"
	"strings"
)

// FilterValues collects the filter_* query parameters of a list request:
// repeated values (checkbox filters) are joined with ",", and the
// filter_<name>_from / filter_<name>_to inputs of range filters become
// "<from>,<to>" under <name> (the _from / _to keys are kept too, so a
// filter that is really named "valid_from" still gets its value).
func FilterValues(q url.Values) map[string]string {
	out := map[string]string{}
	ranges := map[string][2]string{}
	for k, vs := range q {
		if !strings.HasPrefix(k, "filter_") || len(vs) == 0 {
			continue
		}
		name := strings.TrimPrefix(k, "filter_")
		switch {
		case strings.HasSuffix(name, "_from"):
			r := ranges[strings.TrimSuffix(name, "_from")]
			r[0] = strings.TrimSpace(vs[0])
			ranges[strings.TrimSuffix(name, "_from")] = r
		case strings.HasSuffix(name, "_to"):
			r := ranges[strings.TrimSuffix(name, "_to")]
			r[1] = strings.TrimSpace(vs[0])
			ranges[strings.TrimSuffix(name, "_to")] = r
		}
		var vals []string
		for _, v := range vs {
			if v = strings.TrimSpace(v); v != "" {
				vals = append(vals, v)
			}
		}
		if len(vals) > 0 {
			out[name] = strings.Join(vals, ",")
		}
	}
	for name, r := range ranges {
		if r[0] != "" || r[1] != "" {
			if _, set := out[name]; !set {
				out[name] = r[0] + "," + r[1]
			}
		}
	}
	return out
}
