package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/golaravel/gofilament/pkg/activitylog"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/widgets"
)

// counted reports whether an order counts as revenue.
func counted(o *Order) bool {
	return o.Status != "cancelled" && o.Status != "refunded" && o.Status != "pending"
}

// dayTotals sums revenue and order counts per day for the last n days.
func dayTotals(n int, now time.Time) (labels []string, revenue, count []float64) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(n - 1))
	revenue, count = make([]float64, n), make([]float64, n)
	for i := 0; i < n; i++ {
		labels = append(labels, start.AddDate(0, 0, i).Format("Jan 2"))
	}
	mu.RLock()
	for _, o := range orders.rows {
		d := int(o.CreatedAt.Sub(start).Hours() / 24)
		if o.CreatedAt.Before(start) || d < 0 || d >= n {
			continue
		}
		count[d]++
		if counted(o) {
			revenue[d] = money(revenue[d] + o.Total)
		}
	}
	mu.RUnlock()
	return labels, revenue, count
}

func sum(v []float64) float64 {
	t := 0.0
	for _, x := range v {
		t += x
	}
	return t
}

// change is the % difference between the last half of v and the half before.
func change(cur, prev float64) string {
	if prev == 0 {
		return "new"
	}
	d := (cur - prev) / prev * 100
	sign := "+"
	if d < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.1f%% vs previous 30 days", sign, d)
}

func spark(v []float64, buckets int) []int {
	out := make([]int, buckets)
	per := (len(v) + buckets - 1) / buckets
	for i, x := range v {
		out[min(i/per, buckets-1)] += int(x)
	}
	return out
}

func dashboardWidgets() []filament.WidgetInterface {
	stats := widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
		now := time.Now()
		_, rev, cnt := dayTotals(60, now)
		rev30, revPrev := sum(rev[30:]), sum(rev[:30])
		cnt30, cntPrev := sum(cnt[30:]), sum(cnt[:30])
		aov, aovPrev := 0.0, 0.0
		if cnt30 > 0 {
			aov = rev30 / cnt30
		}
		if cntPrev > 0 {
			aovPrev = revPrev / cntPrev
		}
		mu.RLock()
		low := 0
		for _, p := range products.rows {
			if p.DeletedAt == nil && p.Status == "active" && p.Stock < 10 {
				low++
			}
		}
		mu.RUnlock()
		color := func(cur, prev float64) string {
			if cur >= prev {
				return "success"
			}
			return "danger"
		}
		return []widgets.Stat{
			*widgets.NewStat("Revenue (30 days)", fmt.Sprintf("$%s", thousands(rev30))).
				Description(change(rev30, revPrev)).Color(color(rev30, revPrev)).Chart(spark(rev[30:], 10)),
			*widgets.NewStat("Orders (30 days)", fmt.Sprintf("%.0f", cnt30)).
				Description(change(cnt30, cntPrev)).Color(color(cnt30, cntPrev)).Chart(spark(cnt[30:], 10)).URL("orders"),
			*widgets.NewStat("Average order", fmt.Sprintf("$%.2f", aov)).
				Description(change(aov, aovPrev)).Color(color(aov, aovPrev)),
			*widgets.NewStat("Low stock", fmt.Sprint(low)).
				Description("active products under 10 units").Color(map[bool]string{true: "warning", false: "success"}[low > 0]).URL("products?filters[status]=active"),
		}
	})

	revenue := widgets.NewChartWidget("Revenue", widgets.ChartTypeLine).WidgetID("revenue").
		Description("Paid, shipped and delivered orders per day").
		Filters(map[string]string{"7": "Last 7 days", "30": "Last 30 days", "90": "Last 90 days"}).
		DefaultFilter("30").
		FilterData(func(ctx *filament.Context, filter string) widgets.ChartData {
			n := map[string]int{"7": 7, "30": 30, "90": 90}[filter]
			if n == 0 {
				n = 30
			}
			labels, rev, _ := dayTotals(n, time.Now())
			return widgets.ChartData{Labels: labels, Datasets: []widgets.ChartDataset{
				{Label: "Revenue ($)", Data: rev, BorderColor: "#10b981", BackgroundColor: "rgba(16,185,129,0.15)", Fill: true, Tension: 0.35},
			}}
		})

	byStatus := widgets.NewChartWidget("Orders by status", widgets.ChartTypeDoughnut).WidgetID("orders-by-status").
		Description("All orders in the store").
		Data(func(ctx *filament.Context) widgets.ChartData {
			order := []string{"pending", "paid", "shipped", "delivered", "cancelled", "refunded"}
			counts := map[string]float64{}
			mu.RLock()
			for _, o := range orders.rows {
				counts[o.Status]++
			}
			mu.RUnlock()
			var labels []string
			var data []float64
			for _, s := range order {
				labels = append(labels, orderStatus[s])
				data = append(data, counts[s])
			}
			return widgets.ChartData{Labels: labels, Datasets: []widgets.ChartDataset{{Label: "Orders", Data: data}}}
		})

	byCategory := widgets.NewChartWidget("Sales by category", widgets.ChartTypeBar).WidgetID("sales-by-category").
		Description("Units sold in the last 90 days").
		Data(func(ctx *filament.Context) widgets.ChartData {
			cat := map[string]string{}
			mu.RLock()
			for _, p := range products.rows {
				cat[p.SKU] = p.Category
			}
			units := map[string]float64{}
			for _, it := range items.rows {
				if o, ok := orders.rows[it.OrderID]; ok && counted(o) {
					units[cat[it.SKU]] += float64(it.Quantity)
				}
			}
			mu.RUnlock()
			type kv struct {
				k string
				v float64
			}
			var rows []kv
			for k, v := range units {
				rows = append(rows, kv{k, v})
			}
			sort.Slice(rows, func(i, j int) bool { return rows[i].v > rows[j].v })
			var labels []string
			var data []float64
			for _, r := range rows[:min(8, len(rows))] {
				labels = append(labels, r.k)
				data = append(data, r.v)
			}
			return widgets.ChartData{Labels: labels, Datasets: []widgets.ChartDataset{{Label: "Units", Data: data, BackgroundColor: "#10b981"}}}
		})

	recent := widgets.NewTableWidget("Latest orders").
		Columns(
			widgets.TableWidgetColumn{Name: "number", Label: "Order"},
			widgets.TableWidgetColumn{Name: "customer", Label: "Customer"},
			widgets.TableWidgetColumn{Name: "status", Label: "Status"},
			widgets.TableWidgetColumn{Name: "total", Label: "Total"},
			widgets.TableWidgetColumn{Name: "placed", Label: "Placed"},
		).
		Rows(func(ctx *filament.Context) []map[string]any {
			mu.RLock()
			rows, _ := list(orders.all(), listQuery{defaults: "created_at", perPage: 6})
			mu.RUnlock()
			out := make([]map[string]any, 0, len(rows))
			for _, o := range rows {
				out = append(out, map[string]any{
					"number": o.Number, "customer": o.CustomerName, "status": orderStatus[o.Status],
					"total": fmt.Sprintf("$%.2f", o.Total), "placed": activitylog.FormatTimeAgo(o.CreatedAt),
				})
			}
			return out
		})

	return []filament.WidgetInterface{stats, revenue, byStatus, byCategory, recent}
}

func thousands(f float64) string {
	s := fmt.Sprintf("%.0f", f)
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return string(out)
}
