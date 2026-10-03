// Package gold reads the public 24h gold quote table and SJC history, never scripts.
package gold

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const SourceURL = "https://www.24h.com.vn/gia-vang-hom-nay-c425.html"

var vietnam = time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)

type Quote struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Buy        int64  `json:"buy"`
	Sell       int64  `json:"sell"`
	BuyChange  *int64 `json:"buy_change"`
	SellChange *int64 `json:"sell_change"`
}
type Point struct {
	Date string `json:"date"`
	Buy  int64  `json:"buy"`
	Sell int64  `json:"sell"`
}
type Snapshot struct {
	Quotes       []Quote `json:"quotes"`
	History      []Point `json:"history"`
	SourceAt     int64   `json:"source_at"`
	FetchedAt    int64   `json:"fetched_at"`
	IsToday      bool    `json:"is_today"`
	Stale        bool    `json:"stale"`
	Message      string  `json:"message"`
	ChartMessage string  `json:"chart_message"`
}
type Client struct {
	mu      sync.Mutex
	last    Snapshot
	checked time.Time
	read    func(context.Context) (Snapshot, error)
	now     func() time.Time
}

func New() *Client { return &Client{read: fetch, now: time.Now} }
func (c *Client) Read(ctx context.Context, refresh bool) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	ttl := 5 * time.Minute
	if refresh {
		ttl = 30 * time.Second
	}
	if c.checked.IsZero() || now.Sub(c.checked) >= ttl {
		next, err := c.read(ctx)
		c.checked = c.now()
		if err != nil {
			c.last.Stale = c.last.FetchedAt != 0
			c.last.Message = "Unable to load prices from 24h. Check your connection or open the source."
		} else {
			c.last = next
			c.last.FetchedAt = c.checked.Unix()
		}
	}
	// Recompute across midnight even when returning a cached quote.
	out := c.last
	out.IsToday = out.SourceAt != 0 && time.Unix(out.SourceAt, 0).In(vietnam).Format("2006-01-02") == c.now().In(vietnam).Format("2006-01-02")
	return out
}
func fetch(ctx context.Context) (Snapshot, error) {
	client := &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 || req.URL.Scheme != "https" || req.URL.Host != "www.24h.com.vn" {
			return fmt.Errorf("unexpected redirect")
		}
		return nil
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, SourceURL, nil)
	if err != nil {
		return Snapshot{}, err
	}
	req.Header.Set("User-Agent", "Pika/0.1 (personal gold price viewer)")
	req.Header.Set("Accept", "text/html")
	resp, err := client.Do(req)
	if err != nil {
		return Snapshot{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Snapshot{}, fmt.Errorf("source returned %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
	if err != nil {
		return Snapshot{}, err
	}
	if len(data) > 2*1024*1024 {
		return Snapshot{}, fmt.Errorf("source too large")
	}
	return Parse(data)
}
func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func hasClass(n *html.Node, c string) bool {
	for _, v := range strings.Fields(attr(n, "class")) {
		if v == c {
			return true
		}
	}
	return false
}
func walk(n *html.Node, fn func(*html.Node)) {
	fn(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, fn)
	}
}
func textOf(n *html.Node) string {
	var b strings.Builder
	walk(n, func(v *html.Node) {
		if v.Type == html.TextNode {
			b.WriteString(v.Data)
		}
	})
	return strings.TrimSpace(b.String())
}

var stamp = regexp.MustCompile(`Cập nhật lúc\s+(\d{2}:\d{2})\s+\((\d{2}/\d{2}/\d{4})\)`)
var priceNumber = regexp.MustCompile(`^(?:[0-9]{1,3}(?:,[0-9]{3})+|[0-9]+)$`)

func money(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if !priceNumber.MatchString(s) {
		return 0, fmt.Errorf("invalid gold price")
	}
	v, e := strconv.ParseInt(strings.ReplaceAll(s, ",", ""), 10, 64)
	if e != nil || v <= 0 || v > 1e9 {
		return 0, fmt.Errorf("gold price outside bounds")
	}
	return v * 1000, nil
}
func Parse(data []byte) (Snapshot, error) {
	var s Snapshot
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return s, err
	}
	var table, box, chartBox *html.Node
	var sourceTime string
	var script string
	walk(doc, func(n *html.Node) {
		if attr(n, "id") == "div_bieu_do_gia_vang_container" {
			chartBox = n
		}
		if attr(n, "id") == "container_tin_gia_vang" {
			box = n
		}
		if n.Data == "table" && hasClass(n, "gia-vang-search-data-table") {
			table = n
		}
		if n.Type == html.TextNode && stamp.MatchString(n.Data) {
			sourceTime = n.Data
		}
		if n.Data == "script" && strings.Contains(textOf(n), "$('#div_bieu_do_gia_vang').highcharts") {
			script = textOf(n)
		}
	})
	m := stamp.FindStringSubmatch(sourceTime)
	if table == nil || box == nil || len(m) != 3 || !strings.Contains(textOf(box), "nghìn đồng/lượng") {
		return s, fmt.Errorf("unrecognized gold table or units")
	}
	at, err := time.ParseInLocation("15:04 02/01/2006", m[1]+" "+m[2], vietnam)
	if err != nil {
		return s, err
	}
	s.SourceAt = at.Unix()
	// The table's date must agree with the update label, not an unrelated article.
	if !strings.Contains(textOf(box), "Hôm nay ("+m[2]+")") {
		return s, fmt.Errorf("gold table date mismatch")
	}
	seen := map[string]bool{}
	var parseErr error
	walk(table, func(n *html.Node) {
		if n.Data != "tr" || attr(n, "data-seach") == "" {
			return
		}
		cells := []*html.Node{}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Data == "td" {
				cells = append(cells, c)
			}
		}
		if len(cells) != 5 {
			parseErr = fmt.Errorf("gold columns changed")
			return
		}
		q := Quote{Code: attr(n, "data-seach"), Name: textOf(cells[0])}
		current := func(cell *html.Node) (int64, error) {
			var value string
			walk(cell, func(v *html.Node) {
				if hasClass(v, "fixW") {
					value = textOf(v)
				}
			})
			return money(value)
		}
		buy, e1 := current(cells[1])
		sell, e2 := current(cells[2])
		if e1 != nil || e2 != nil || sell < buy || q.Name == "" || seen[q.Code] {
			parseErr = fmt.Errorf("invalid gold quote")
			return
		}
		q.Buy, q.Sell = buy, sell
		if previous, e := money(textOf(cells[3])); e == nil {
			d := buy - previous
			q.BuyChange = &d
		}
		if previous, e := money(textOf(cells[4])); e == nil {
			d := sell - previous
			q.SellChange = &d
		}
		seen[q.Code] = true
		s.Quotes = append(s.Quotes, q)
	})
	if parseErr != nil {
		return Snapshot{}, parseErr
	}
	if !seen["sjc"] || len(s.Quotes) == 0 {
		return Snapshot{}, fmt.Errorf("missing SJC quote")
	}
	var chartName string
	if chartBox != nil {
		walk(chartBox, func(n *html.Node) {
			if hasClass(n, "name-gold") {
				chartName = textOf(n)
			}
		})
	}
	if chartName != "SJC" {
		err = fmt.Errorf("history is not labelled SJC")
	} else {
		s.History, err = parseHistory(script, at)
	}
	if err == nil {
		var sjc int64
		for _, q := range s.Quotes {
			if q.Code == "sjc" {
				sjc = q.Buy
			}
		}
		for _, p := range s.History {
			if p.Buy < sjc/10 || p.Sell > sjc*10 {
				err = fmt.Errorf("history units inconsistent with quotes")
				break
			}
		}
	}
	if err != nil {
		s.ChartMessage = "SJC chart is unavailable from the source."
		s.History = nil
	}
	return s, nil
}

var categories = regexp.MustCompile(`categories:\s*\[([^\]]*)\]`)
var dates = regexp.MustCompile(`'([0-9]{2}/[0-9]{2})'`)

func parseHistory(script string, at time.Time) ([]Point, error) {
	bad := fmt.Errorf("invalid SJC history")
	c := categories.FindStringSubmatch(script)
	if len(c) != 2 {
		return nil, bad
	}
	labels := dates.FindAllStringSubmatch(c[1], -1)
	rest := dates.ReplaceAllString(c[1], "")
	if strings.TrimSpace(strings.ReplaceAll(rest, ",", "")) != "" || len(labels) < 2 || len(labels) > 40 {
		return nil, bad
	}
	values := make([][]int64, 2)
	for i, name := range []string{"Mua vào", "Bán ra"} {
		re := regexp.MustCompile(`(?s)name:\s*'` + name + `'.*?data:\s*(\[[^\]]*\])`)
		m := re.FindStringSubmatch(script)
		if len(m) != 2 || json.Unmarshal([]byte(m[1]), &values[i]) != nil || len(values[i]) != len(labels) {
			return nil, bad
		}
	}
	out := make([]Point, len(labels))
	next := at.AddDate(0, 0, 1)
	for i := len(labels) - 1; i >= 0; i-- {
		day, e := time.ParseInLocation("02/01/2006", labels[i][1]+fmt.Sprintf("/%d", next.Year()), vietnam)
		if e != nil {
			return nil, bad
		}
		if !day.Before(next) {
			day = day.AddDate(-1, 0, 0)
		}
		if at.Sub(day) > 40*24*time.Hour || values[0][i] <= 0 || values[1][i] < values[0][i] || values[1][i] > 1e12 {
			return nil, bad
		}
		out[i] = Point{Date: day.Format("2006-01-02"), Buy: values[0][i], Sell: values[1][i]}
		next = day
	}
	return out, nil
}
