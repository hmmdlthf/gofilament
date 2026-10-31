package globalsearch

import (
	"fmt"
	"html"
	"math"
	"sort"
	"strings"
	"unicode"
)

// Score calculates a relevance score between 0.0 and 1.0 for a query against target text.
// Returns (score, true) if matched, or (0.0, false) if not matched.
func Score(query, target string) (float64, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	t := strings.ToLower(strings.TrimSpace(target))

	if q == "" || t == "" {
		return 0.0, false
	}

	// 1. Exact match
	if q == t {
		return 1.0, true
	}

	// 2. Prefix match
	if strings.HasPrefix(t, q) {
		return 0.95, true
	}

	// 3. Substring match
	if strings.Contains(t, q) {
		// Bonus if match occurs at word boundary
		bonus := 0.0
		words := strings.FieldsFunc(t, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		for _, w := range words {
			if strings.HasPrefix(w, q) {
				bonus = 0.05
				break
			}
		}
		return 0.85 + bonus, true
	}

	// 4. Subsequence / Acronym match
	qr := []rune(q)
	tr := []rune(t)
	qi := 0
	boundaryMatches := 0

	for ti := 0; ti < len(tr) && qi < len(qr); ti++ {
		if tr[ti] == qr[qi] {
			// Check if match is at word boundary
			if ti == 0 || !unicode.IsLetter(tr[ti-1]) && !unicode.IsDigit(tr[ti-1]) {
				boundaryMatches++
			}
			qi++
		}
	}

	if qi == len(qr) {
		ratio := float64(boundaryMatches) / float64(len(qr))
		score := 0.70 + 0.15*ratio
		return math.Min(0.84, score), true
	}

	// 5. Typo tolerance (Levenshtein distance)
	if len(q) >= 3 {
		words := strings.FieldsFunc(t, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})

		for _, w := range words {
			dist := Levenshtein(q, w)
			if len(q) <= 4 && dist <= 1 {
				return 0.65, true
			} else if len(q) > 4 && dist <= 2 {
				return 0.65 - float64(dist)*0.10, true
			}
		}

		// Also check against full target if single phrase
		dist := Levenshtein(q, t)
		if len(q) <= 4 && dist <= 1 {
			return 0.60, true
		} else if len(q) > 4 && dist <= 2 {
			return 0.60 - float64(dist)*0.10, true
		}
	}

	return 0.0, false
}

// Match returns true if query matches target with a score > 0.40.
func Match(query, target string) bool {
	score, matched := Score(query, target)
	return matched && score >= 0.40
}

// Levenshtein computes the standard edit distance between s1 and s2.
func Levenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n, m := len(r1), len(r2)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}

	dp := make([]int, m+1)
	for j := 0; j <= m; j++ {
		dp[j] = j
	}

	for i := 1; i <= n; i++ {
		prev := dp[0]
		dp[0] = i
		for j := 1; j <= m; j++ {
			temp := dp[j]
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			dp[j] = min(dp[j]+1, min(dp[j-1]+1, prev+cost))
			prev = temp
		}
	}
	return dp[m]
}

// Highlight wraps matching characters or substrings in <mark class="spotlight-highlight">.
func Highlight(text, query string) string {
	if query == "" || text == "" {
		return html.EscapeString(text)
	}

	qLower := strings.ToLower(query)
	tLower := strings.ToLower(text)

	// 1. Direct substring match
	if idx := strings.Index(tLower, qLower); idx != -1 {
		prefix := html.EscapeString(text[:idx])
		matched := html.EscapeString(text[idx : idx+len(query)])
		suffix := html.EscapeString(text[idx+len(query):])
		return fmt.Sprintf(`%s<mark class="spotlight-highlight">%s</mark>%s`, prefix, matched, suffix)
	}

	// 2. Subsequence highlighting
	qr := []rune(qLower)
	tr := []rune(text)
	var sb strings.Builder
	qi := 0

	for _, r := range tr {
		rLower := unicode.ToLower(r)
		if qi < len(qr) && rLower == qr[qi] {
			sb.WriteString(fmt.Sprintf(`<mark class="spotlight-highlight">%s</mark>`, html.EscapeString(string(r))))
			qi++
		} else {
			sb.WriteString(html.EscapeString(string(r)))
		}
	}

	if qi == len(qr) {
		return sb.String()
	}

	// 3. Fallback: return escaped text
	return html.EscapeString(text)
}

// ScoredItem wraps an item with its relevance score for sorting.
type ScoredItem[T any] struct {
	Item  T
	Score float64
}

// Rank filters and sorts items by relevance score descending.
func Rank[T any](items []T, query string, textFunc func(T) string) []T {
	if query == "" {
		return items
	}

	var scored []ScoredItem[T]
	for _, it := range items {
		targetText := textFunc(it)
		if s, matched := Score(query, targetText); matched {
			scored = append(scored, ScoredItem[T]{Item: it, Score: s})
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	out := make([]T, len(scored))
	for i, sc := range scored {
		out[i] = sc.Item
	}
	return out
}
