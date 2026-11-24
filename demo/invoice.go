package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	gcontext "github.com/golaravel/golaravel/pkg/context"
)

// Printable invoices: a plain server-rendered page (Templ) behind a signed
// link that expires, so it opens in a new browser tab without the app's
// sign-in token, and can be printed or saved as PDF.

// invoiceLinkTTL is how long an invoice link from the admin stays valid.
const invoiceLinkTTL = time.Hour

// linkKey signs invoice links; a new one each start, so old links die with
// the process (the demo resets anyway).
var linkKey = func() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}()

func linkSig(id string, exp int64) string {
	m := hmac.New(sha256.New, linkKey)
	fmt.Fprintf(m, "invoice:%s:%d", id, exp)
	return hex.EncodeToString(m.Sum(nil))[:32]
}

// invoiceURL is a signed link to an order's invoice, valid until exp.
func invoiceURL(id string, exp time.Time) string {
	e := exp.Unix()
	return "/invoices/" + url.PathEscape(id) + "?exp=" + strconv.FormatInt(e, 10) + "&sig=" + linkSig(id, e)
}

func validInvoiceLink(id, exp, sig string, now time.Time) bool {
	e, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || now.Unix() > e {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(linkSig(id, e)))
}

func invoiceNumber(o Order) string {
	return "INV-" + strings.TrimPrefix(o.Number, "ORD-")
}

func paymentLabel(p string) string {
	if l, ok := payments[p]; ok {
		return l
	}
	return p
}

// invoiceHandler serves GET /invoices/:id?exp=…&sig=….
func invoiceHandler(c *gcontext.Context) error {
	id := c.Param("id")
	if !validInvoiceLink(id, c.Query("exp"), c.Query("sig"), time.Now()) {
		return c.String(http.StatusForbidden, "This invoice link is invalid or has expired. Open the invoice again from the admin.")
	}
	mu.RLock()
	o, err := orders.get(id)
	var v invoiceView
	if err == nil {
		v.Order = *o
		if cust, err := customers.get(o.CustomerID); err == nil {
			v.Customer = *cust
		}
		for _, it := range items.rows {
			if it.OrderID == id {
				v.Items = append(v.Items, *it)
			}
		}
	}
	mu.RUnlock()
	if err != nil {
		return c.String(http.StatusNotFound, "Order not found (the demo data resets every hour).")
	}
	sortItems(v.Items)

	var buf bytes.Buffer
	start := time.Now()
	if err := invoicePage(v).Render(c.Request.Context(), &buf); err != nil {
		return err
	}
	took := time.Since(start)
	page := bytes.Replace(buf.Bytes(), []byte(renderTimeMark), []byte(formatDuration(took)), 1)

	c.SetHeader("Cache-Control", "private, no-store")
	c.SetHeader("Server-Timing", fmt.Sprintf("render;desc=\"Templ\";dur=%.3f", float64(took.Microseconds())/1000))
	c.SetHeader("Referrer-Policy", "no-referrer")
	return c.HTML(http.StatusOK, string(page))
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%d µs", max(1, d.Microseconds()))
	}
	return fmt.Sprintf("%.2f ms", float64(d.Microseconds())/1000)
}

func sortItems(v []OrderItem) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && lessValue(v[j].ID, v[j-1].ID); j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}
