package gold

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const fixture = `<p><em>Nguồn: 24h - Cập nhật lúc 17:46 (03/10/2026)</em></p>
<div id="container_tin_gia_vang"><div>Hôm nay (03/10/2026)</div>
<table class="gia-vang-search-data-table"><tbody>
<tr data-seach="sjc"><td><h2>SJC</h2></td><td><span class="fixW">140,500</span><span>600</span></td><td><span class="fixW">143,500</span><span>600</span></td><td>141,100</td><td>144,100</td></tr>
<tr data-seach="btmh"><td>BTMH</td><td><span class="fixW">139,500</span></td><td><span class="fixW">143,500</span></td><td>-</td><td>143,900</td></tr>
</tbody></table><em>Đơn vị: nghìn đồng/lượng</em></div>
<div id="div_bieu_do_gia_vang_container"><p class="name-gold">SJC</p><script>$('#div_bieu_do_gia_vang').highcharts({xAxis:{categories:['01/10','02/10','03/10']},series:[{name:'Mua vào',data:[141300000,141100000,140500000]},{name:'Bán ra',data:[144300000,144100000,143500000]}]});</script></div>`

func TestParseUnitsQuotesAndHistory(t *testing.T) {
	s, err := Parse([]byte(fixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Quotes) != 2 || s.Quotes[0].Buy != 140500000 || s.Quotes[0].Sell != 143500000 || *s.Quotes[0].BuyChange != -600000 || s.Quotes[1].BuyChange != nil {
		t.Fatal(s.Quotes)
	}
	if s.SourceAt != time.Date(2026, 10, 3, 17, 46, 0, 0, vietnam).Unix() {
		t.Fatal(s.SourceAt)
	}
	if len(s.History) != 3 || s.History[2].Date != "2026-10-03" || s.History[2].Buy != s.Quotes[0].Buy || s.ChartMessage != "" {
		t.Fatal(s)
	}
}
func TestFailClosedForChangedUnitsAndDates(t *testing.T) {
	for _, data := range []string{strings.Replace(fixture, "nghìn đồng/lượng", "USD/ounce", 1), strings.Replace(fixture, "Hôm nay (03/10/2026)", "Hôm nay (02/10/2026)", 1), strings.Replace(fixture, "140,500", "140.500", 1), strings.Replace(fixture, "data-seach=\"sjc\"", "data-seach=\"other\"", 1), "<html>maintenance</html>"} {
		if _, err := Parse([]byte(data)); err == nil {
			t.Fatal("invalid source accepted")
		}
	}
}
func TestChartFailureKeepsValidQuotes(t *testing.T) {
	s, err := Parse([]byte(strings.Replace(fixture, "data:[141300000,141100000,140500000]", "data:[141300000,null,140500000]", 1)))
	if err != nil || len(s.Quotes) != 2 || len(s.History) != 0 || s.ChartMessage == "" {
		t.Fatal(s, err)
	}
	// Unknown script format cannot execute or fabricate history.
	s, err = Parse([]byte(strings.Split(fixture, "<script>")[0]))
	if err != nil || s.History != nil || s.ChartMessage == "" {
		t.Fatal(s, err)
	}
}
func TestYearBoundary(t *testing.T) {
	script := `categories:['31/12','01/01','02/01'],series:[{name:'Mua vào',data:[100000000,101000000,102000000]},{name:'Bán ra',data:[103000000,104000000,105000000]}]`
	rows, err := parseHistory(script, time.Date(2027, 1, 2, 17, 0, 0, 0, vietnam))
	if err != nil || rows[0].Date != "2026-12-31" || rows[2].Date != "2027-01-02" {
		t.Fatal(rows, err)
	}
	if _, err := parseHistory(strings.Replace(script, "'01/01'", "'31/12'", 1), time.Date(2027, 1, 2, 17, 0, 0, 0, vietnam)); err == nil {
		t.Fatal("duplicate/out-of-window chart accepted")
	}
}
func TestCacheRefreshMidnightAndOffline(t *testing.T) {
	now := time.Date(2026, 10, 3, 23, 59, 50, 0, vietnam)
	var calls atomic.Int32
	c := &Client{now: func() time.Time { return now }, read: func(context.Context) (Snapshot, error) { calls.Add(1); return Parse([]byte(fixture)) }}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); c.Read(context.Background(), false) }()
	}
	wg.Wait()
	if calls.Load() != 1 || !c.Read(context.Background(), false).IsToday {
		t.Fatal("cache/today")
	}
	now = now.Add(20 * time.Second)
	if c.Read(context.Background(), true).IsToday || calls.Load() != 1 {
		t.Fatal("midnight/cache")
	}
	now = now.Add(11 * time.Second)
	c.Read(context.Background(), true)
	if calls.Load() != 2 {
		t.Fatal("manual refresh not allowed after 30s")
	}
	now = now.Add(time.Minute)
	c.Read(context.Background(), false)
	if calls.Load() != 2 {
		t.Fatal("auto cache shorter than 5min")
	}
	c.read = func(context.Context) (Snapshot, error) { return Snapshot{}, errors.New("offline") }
	now = now.Add(5 * time.Minute)
	s := c.Read(context.Background(), false)
	if !s.Stale || len(s.Quotes) != 2 || s.Message == "" || s.FetchedAt == now.Unix() {
		t.Fatal(s)
	}
	c.read = func(context.Context) (Snapshot, error) { return Parse([]byte(fixture)) }
	now = now.Add(31 * time.Second)
	s = c.Read(context.Background(), true)
	if s.Stale || s.Message != "" {
		t.Fatal(s)
	}
}

func TestChartBrandAndUnits(t *testing.T) {
	for _, data := range []string{strings.Replace(fixture, `class="name-gold">SJC`, `class="name-gold">DOJI`, 1), strings.ReplaceAll(strings.ReplaceAll(fixture, "[141300000,141100000,140500000]", "[141300,141100,140500]"), "[144300000,144100000,143500000]", "[144300,144100,143500]")} {
		s, err := Parse([]byte(data))
		if err != nil || len(s.Quotes) != 2 || s.History != nil || s.ChartMessage == "" {
			t.Fatal(s, err)
		}
	}
}
