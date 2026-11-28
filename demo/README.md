# Live demo: demo.gofilament.elridhma.com

**Apex Store**, a sample shop's back office built with GoFilament and shown in
the Flutter admin app ([`flutter/gofilament_admin`](../flutter/gofilament_admin)).
One Go binary serves both the app and the panel API it reads, at `/admin`.

## What it shows

- **Products**, with **Variants** and **Barcodes** as relation-manager tabs, plus
  filters, a trash, import/export and bulk edits (status, category, featured)
- **Orders**, with line items, a **Ship** action, and an audit-trail History
  timeline on every record
- **Categories**: the product form's category select can create a missing
  category on the spot (`CreateOptionFrom("categories")`)
- **Customers** and the **Audit log**
- **Uploads**: product photo galleries (shrunk to 1200px), round customer
  avatars cropped square in the app, and private PDF spec sheets shown only
  through signed links that expire
- **Dashboard**: stats with trends, a revenue chart (7/30/90 days), orders by
  status, and sales by category
- **Roles**: *Admin* can do everything; *Warehouse staff* see only products and
  orders, may update them but not delete, and can't open customers or the log
- Global search, and **Settings** (theme, language with Arabic right to left, sidebar)
- **Printable invoices** from each order: a GoLaravel + Templ page rendered in
  about 20 µs, behind a signed link that expires after an hour (`invoice.templ`;
  run `templ generate` after editing it)

## Sandbox rules

| | |
| --- | --- |
| Accounts | `admin@example.com` / `demo`, `warehouse@example.com` / `demo`. The login screen has a one-click button for each; there is no sign-up |
| Data | In memory. Every hour it goes back to the same generated sample (`seed.go`): 40 products, 120 customers and about 90 days of orders. Each table is capped at 2,000 rows |
| Abuse limits | Per IP: 300 requests a minute, of which 40 may be writes. Bodies are capped at 1 MB, and logins are throttled |
| Side effects | No mail (no sign-up or password reset) and no payments. Uploads (product photos, customer avatars, PDF spec sheets) accept images and PDFs under 1 MB and are kept in a temporary folder that is wiped at every reset |
| Search engines | `X-Robots-Tag: noindex` on everything |

## Running it locally

```bash
cd demo/client && flutter pub get && flutter build web --release && cd ../..
cp -r demo/client/build/web/. demo/flutterweb/   # the Go binary embeds this
go run ./demo                                     # http://localhost:8080/admin
```

Without the Flutter build, `go run ./demo` still starts. It serves the start
page and the API, and `/admin` explains what is missing. `go test ./demo`
needs no Flutter build.

## Deploying

[`.github/workflows/demo.yml`](../.github/workflows/demo.yml) runs when a release
is published, or by hand (*Demo → Run workflow*). It does three things:

1. Builds the Flutter web app, embeds it, and compiles a static Linux binary.
2. Uploads the binary to goplesk with `POST /api/v1/apps/<app>/release`. goplesk
   swaps it in blue/green after `/healthz` answers.
3. On the first run, creates the app instead (`POST /api/v1/apps/deploy`), with
   port 8095, 200 MB of memory and `GOMEMLIMIT=170MiB`.

Before the first run:

1. **DNS**: add an `A` record for `demo.gofilament` → `217.154.252.130` at the
   `elridhma.com` DNS host (Namecheap).
2. **Secrets**: `GOPLESK_CLUSTER_KEY` and `GOLARAVEL_READ_TOKEN`, the same ones
   the Website and CI workflows use. Optional variables: `DEMO_DOMAIN`,
   `DEMO_APP` (default `gofilament-demo`), `DEMO_PORT` (default `8095`) and
   `GOPLESK_URL`.
