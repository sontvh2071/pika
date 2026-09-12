package catalog

import (
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

type Candidate struct {
	Path        string   `json:"path"`
	VersionHint string   `json:"-"`
	FlatpakID   string   `json:"-"`
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Subtitle    string   `json:"subtitle"`
	Target      string   `json:"-"`
	Icon        string   `json:"-"`
	Aliases     []string `json:"-"`
	Terms       []string `json:"-"`
	PathTerm    string   `json:"-"`
}
type Usage struct {
	Count    int
	LastUsed int64
}
type Result struct {
	Path     string `json:"path"`
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Subtitle string `json:"subtitle"`
	Pinned   bool   `json:"pinned"`
}

func Normalize(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r == 'đ' {
			r = 'd'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
func Prepare(c Candidate) Candidate {
	c.Terms = []string{Normalize(c.Name)}
	for _, s := range c.Aliases {
		c.Terms = append(c.Terms, Normalize(s))
	}
	c.PathTerm = Normalize(c.Subtitle)
	return c
}
func Match(q, t string) (int, float64) {
	if q == t {
		return 5, 100
	}
	if strings.HasPrefix(t, q) {
		return 4, 90 - float64(len(t)-len(q))*.05
	}
	if strings.Contains(" "+t, " "+q) {
		return 3, 80
	}
	if strings.Contains(t, q) {
		return 2, 70
	}
	qr, tr := []rune(strings.ReplaceAll(q, " ", "")), []rune(t)
	if len(qr) == 0 {
		return 0, 0
	}
	j, first, last := 0, -1, 0
	for i, r := range tr {
		if r == qr[j] {
			if first < 0 {
				first = i
			}
			last = i
			j++
			if j == len(qr) {
				return 1, 50 - float64(last-first-len(qr)+1)*.5 - float64(first)*.1
			}
		}
	}
	return 0, 0
}
func Search(items []Candidate, usage map[string]Usage, pins []string, raw, kind string, limit int, now time.Time) []Result {
	if limit < 1 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}
	if strings.HasPrefix(strings.TrimSpace(raw), ">") {
		kind = "command"
		raw = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), ">"))
	}
	q := Normalize(raw)
	pinned := map[string]bool{}
	for _, id := range pins {
		pinned[id] = true
	}
	type scored struct {
		c     Candidate
		tier  int
		score float64
	}
	best := make([]scored, 0, limit+1)
	less := func(a, b scored) bool {
		if a.tier != b.tier {
			return a.tier > b.tier
		}
		if a.score != b.score {
			return a.score > b.score
		}
		if a.c.Name != b.c.Name {
			return a.c.Name < b.c.Name
		}
		return a.c.ID < b.c.ID
	}
	for _, c := range items {
		if kind != "all" && kind != "" && c.Kind != kind && !(kind == "file" && c.Kind == "directory") && !(kind == "app" && c.Kind == "system") {
			continue
		}
		tier, score := 0, 0.0
		if q == "" {
			if kind != "file" && c.Kind != "app" && c.Kind != "command" && !pinned[c.ID] && usage[c.ID].Count == 0 {
				continue
			}
			tier = 1
		} else {
			for _, term := range c.Terms {
				t, s := Match(q, term)
				if t > tier || (t == tier && s > score) {
					tier, score = t, s
				}
			}
			if strings.Contains(q, " ") {
				all := true
				for _, token := range strings.Fields(q) {
					if !strings.Contains(strings.Join(c.Terms, " ")+" "+c.PathTerm, token) {
						all = false
						break
					}
				}
				if all && tier < 2 {
					tier, score = 2, 45
				}
			}
			if tier == 0 && strings.Contains(c.PathTerm, q) {
				tier, score = 1, 5
			}
			if tier == 0 {
				continue
			}
		}
		// Explicit session keywords must resolve to the action, even when an
		// installed settings app also lists "lock" as one of its keywords.
		if c.Kind == "system" && tier == 5 {
			tier = 6
		}
		u := usage[c.ID]
		score += math.Min(20, 4*math.Log1p(float64(u.Count)))
		if u.LastUsed > 0 {
			age := math.Max(0, now.Sub(time.Unix(u.LastUsed, 0)).Hours())
			score += 10 / (1 + age/24)
		}
		if c.Kind == "app" {
			score += 3
		}
		if pinned[c.ID] {
			score += 30
		}
		s := scored{c, tier, score}
		pos := sort.Search(len(best), func(i int) bool { return less(s, best[i]) })
		if pos >= limit {
			continue
		}
		best = append(best, scored{})
		copy(best[pos+1:], best[pos:])
		best[pos] = s
		if len(best) > limit {
			best = best[:limit]
		}
	}
	out := make([]Result, 0, len(best))
	for _, s := range best {
		out = append(out, Result{ID: s.c.ID, Kind: s.c.Kind, Name: s.c.Name, Subtitle: s.c.Subtitle, Pinned: pinned[s.c.ID], Path: s.c.Path})
	}
	return out
}
