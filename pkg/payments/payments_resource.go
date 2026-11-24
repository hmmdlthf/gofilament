package payments

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/payment"
)

// PaymentRow is a payment as the panel shows it.
type PaymentRow struct {
	*payment.Payment
	GatewayName   string `json:"gateway_name"`
	Money         string `json:"money"`
	RefundedMoney string `json:"refunded_money"`
	StatusLabel   string `json:"status_label"`
	Created       string `json:"created"`
	Paid          string `json:"paid"`
	features      payment.Features
}

// LogRow is one history entry of a payment.
type LogRow struct {
	*payment.Log
	Change string `json:"change"`
	At     string `json:"at"`
}

var statusColors = map[string]string{
	"pending": "warning", "authorized": "info", "paid": "success", "partially_refunded": "info",
	"refunded": "gray", "failed": "danger", "canceled": "gray", "expired": "gray",
}

func statusLabels() map[string]string {
	out := map[string]string{}
	for _, s := range payment.Statuses() {
		out[string(s)] = s.Label()
	}
	return out
}

const timeLayout = "2006-01-02 15:04"

func paymentRow(m *payment.Manager, ctx context.Context, p *payment.Payment, names map[string]string) *PaymentRow {
	row := &PaymentRow{Payment: p, GatewayName: payment.Pick(names[p.GatewayID], p.GatewayID), Money: p.Money(),
		StatusLabel: p.Status.Label(), Created: p.CreatedAt.Local().Format(timeLayout)}
	if p.Refunded > 0 {
		row.RefundedMoney = payment.FormatMoney(p.Refunded, p.Currency)
	}
	if p.PaidAt != nil {
		row.Paid = p.PaidAt.Local().Format(timeLayout)
	}
	if d, ok := m.Registry().Driver(p.Driver); ok {
		row.features = d.Features
	}
	return row
}

func gatewayNames(ctx context.Context, m *payment.Manager) map[string]string {
	names := map[string]string{}
	if gs, err := m.GatewayConfigs(ctx); err == nil {
		for _, g := range gs {
			names[g.ID] = g.Name
		}
	}
	return names
}

func trimErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, payment.ErrNotSupported) {
		return errors.New("this gateway does not support that")
	}
	return errors.New(strings.TrimPrefix(err.Error(), "payment: "))
}

// PaymentsResource is the "Payments" resource: read-only records with
// gateway actions.
func PaymentsResource(m *payment.Manager) *filament.BaseResource[*PaymentRow] {
	res := filament.NewResource[*PaymentRow]("payments").
		ModelName("Payment").
		PluralModelName("Payments").
		NavigationGroup(Group).
		NavigationIcon("heroicon-banknotes").
		NavigationSort(91).
		RecordID(func(p *PaymentRow) string { return p.ID }).
		CanCreate(func(*filament.Context) bool { return false }).
		CanEdit(func(*filament.Context, *PaymentRow) bool { return false }).
		CanDelete(func(*filament.Context, *PaymentRow) bool { return false }).
		CanDeleteAny(func(*filament.Context) bool { return false })

	res.Table(func(t *tables.Table[*PaymentRow]) *tables.Table[*PaymentRow] {
		gatewayOpts := map[string]string{}
		if gs, err := m.GatewayConfigs(context.Background()); err == nil {
			for _, g := range gs {
				gatewayOpts[g.ID] = g.Name
			}
		}
		return t.Columns(
			tables.TextColumn[*PaymentRow]("reference").Label("Order").Weight("bold").Searchable().Copyable(),
			tables.TextColumn[*PaymentRow]("money").Label("Amount").Sortable(),
			tables.BadgeColumn[*PaymentRow]("status").Label("Status").Sortable().
				Colors(statusColors).Labels(statusLabels()),
			tables.TextColumn[*PaymentRow]("gateway_name").Label("Gateway"),
			tables.TextColumn[*PaymentRow]("customer_email").Label("Customer").Searchable(),
			tables.TextColumn[*PaymentRow]("created").Label("Created").Sortable(),
			tables.BadgeColumn[*PaymentRow]("mode").Label("Mode").
				Colors(map[string]string{"live": "success", "test": "warning"}).
				Labels(map[string]string{"live": "Live", "test": "Test"}).ToggledHiddenByDefault(true),
			tables.TextColumn[*PaymentRow]("refunded_money").Label("Refunded").ToggledHiddenByDefault(true),
			tables.TextColumn[*PaymentRow]("id").Label("Payment ID").Copyable().ToggledHiddenByDefault(true),
			tables.TextColumn[*PaymentRow]("gateway_ref").Label("Gateway ref").Copyable().ToggledHiddenByDefault(true),
		).Filters(
			tables.SelectFilter("status").Label("Status").Options(statusLabels()),
			tables.SelectFilter("gateway_id").Label("Gateway").Options(gatewayOpts),
			tables.SelectFilter("mode").Label("Mode").Options(map[string]string{"live": "Live", "test": "Test"}),
		).Actions(
			tables.ViewAction[*PaymentRow](),
			tables.Action[*PaymentRow]("refund").Label("Refund").Icon("heroicon-arrow-uturn-left").Color("danger").
				Visible(func(p *PaymentRow) bool { return p.features.Refunds && p.Refundable() > 0 }).
				ModalHeading("Refund payment").
				ModalDescription("Money goes back to the customer through the gateway. Leave the amount blank to refund everything left.").
				Form(func(f *forms.Form) *forms.Form {
					return f.Schema(
						forms.TextInput("amount").Label("Amount").Numeric().
							HelperText("In the payment's currency, for example 12.50. Blank refunds everything left."),
						forms.TextInput("reason").Label("Reason"),
					)
				}).
				Handle(func(ctx context.Context, p *PaymentRow, data map[string]any) error {
					var amount int64
					if s := strOf(data["amount"]); s != "" {
						n, err := payment.ParseDecimal(s, p.Currency)
						if err != nil || n <= 0 {
							return fieldError("amount", errors.New("enter a positive amount"))
						}
						amount = n
					}
					_, err := m.Refund(ctx, p.ID, amount, strOf(data["reason"]))
					return trimErr(err)
				}).
				SuccessMessage("Refunded"),
			tables.Action[*PaymentRow]("capture").Label("Capture").Icon("heroicon-check-circle").Color("primary").
				Visible(func(p *PaymentRow) bool { return p.Status == payment.StatusAuthorized && p.features.Capture }).
				RequiresConfirmation().ModalDescription("Take the authorized amount from the customer now.").
				Handle(func(ctx context.Context, p *PaymentRow, _ map[string]any) error {
					_, err := m.Capture(ctx, p.ID, 0)
					return trimErr(err)
				}).
				SuccessMessage("Captured"),
			tables.Action[*PaymentRow]("void").Label("Void").Icon("heroicon-x-circle").Color("warning").
				Visible(func(p *PaymentRow) bool {
					return p.Status == payment.StatusPending || p.Status == payment.StatusAuthorized && p.features.Cancel
				}).
				RequiresConfirmation().ModalDescription("Release the authorization (or abandon the pending payment). The customer is not charged.").
				Handle(func(ctx context.Context, p *PaymentRow, _ map[string]any) error {
					_, err := m.Cancel(ctx, p.ID)
					return trimErr(err)
				}).
				SuccessMessage("Voided"),
			tables.Action[*PaymentRow]("sync").Label("Sync with gateway").Icon("heroicon-arrow-path").
				Visible(func(p *PaymentRow) bool { return p.features.StatusLookup && !p.Status.Final() }).
				Handle(func(ctx context.Context, p *PaymentRow, _ map[string]any) error {
					_, err := m.Sync(ctx, p.ID)
					return trimErr(err)
				}).
				SuccessMessage("Synced"),
		)
	})

	// The form is the record view in the Flutter admin (opened read-only).
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Section("Payment").Columns(2).Schema(
				forms.TextInput("reference").Label("Order").Disabled(),
				forms.TextInput("money").Label("Amount").Disabled(),
				forms.TextInput("status_label").Label("Status").Disabled(),
				forms.TextInput("gateway_name").Label("Gateway").Disabled(),
				forms.TextInput("refunded_money").Label("Refunded").Disabled(),
				forms.TextInput("failure_reason").Label("Failure").Disabled(),
				forms.TextInput("created").Label("Created").Disabled(),
				forms.TextInput("paid").Label("Paid").Disabled(),
			),
			forms.Section("Customer").Columns(3).Schema(
				forms.TextInput("customer_name").Label("Name").Disabled(),
				forms.TextInput("customer_email").Label("Email").Disabled(),
				forms.TextInput("customer_phone").Label("Phone").Disabled(),
			),
			forms.Section("Reference").Columns(2).Schema(
				forms.TextInput("id").Label("Payment ID").Disabled(),
				forms.TextInput("gateway_ref").Label("Gateway reference").Disabled(),
				forms.TextInput("mode").Label("Mode").Disabled(),
				forms.TextInput("description").Label("Description").Disabled(),
			),
		)
	})

	res.Infolist(func(i *infolists.Infolist) *infolists.Infolist {
		return i.Schema(
			infolists.Section("Payment").Columns(3).Schema(
				infolists.TextEntry("reference").Label("Order").Weight("bold").Copyable(),
				infolists.TextEntry("money").Label("Amount").Weight("bold"),
				infolists.BadgeEntry("status").Colors(statusColors),
				infolists.TextEntry("gateway_name").Label("Gateway"),
				infolists.TextEntry("refunded_money").Label("Refunded"),
				infolists.TextEntry("failure_reason").Label("Failure"),
				infolists.TextEntry("created").Label("Created"),
				infolists.TextEntry("paid").Label("Paid"),
				infolists.TextEntry("mode").Label("Mode"),
			),
			infolists.Section("Customer").Columns(3).Schema(
				infolists.TextEntry("customer_name").Label("Name"),
				infolists.TextEntry("customer_email").Label("Email").Copyable(),
				infolists.TextEntry("customer_phone").Label("Phone"),
			),
			infolists.Section("Reference").Columns(2).Schema(
				infolists.TextEntry("id").Label("Payment ID").Copyable(),
				infolists.TextEntry("gateway_ref").Label("Gateway reference").Copyable(),
			),
		)
	})

	history := filament.NewRelationManager[*LogRow]("history", "History").
		Icon("heroicon-clock").
		RecordID(func(l *LogRow) string { return l.ID }).
		Table(func(t *tables.Table[*LogRow]) *tables.Table[*LogRow] {
			return t.Columns(
				tables.TextColumn[*LogRow]("at").Label("When"),
				tables.TextColumn[*LogRow]("event").Label("Event"),
				tables.TextColumn[*LogRow]("change").Label("Status"),
				tables.TextColumn[*LogRow]("message").Label("Details"),
			)
		}).
		OnList(func(ctx *filament.Context, parentID, _, _, _ string, page, perPage int) ([]*LogRow, int, error) {
			logs, err := m.Logs(ctx.GoContext(), parentID)
			if err != nil {
				return nil, 0, err
			}
			rows := make([]*LogRow, 0, len(logs))
			for i := len(logs) - 1; i >= 0; i-- { // newest first
				l := logs[i]
				change := ""
				if l.To != "" {
					change = l.To.Label()
					if l.From != "" {
						change = l.From.Label() + " → " + change
					}
				}
				rows = append(rows, &LogRow{Log: l, Change: change, At: l.CreatedAt.Local().Format("2006-01-02 15:04:05")})
			}
			return pageOf(rows, page, perPage), len(rows), nil
		})
	res.Relations(history)

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*PaymentRow, int, error) {
		q := payment.PaymentQuery{Search: search, Status: payment.Status(filters["status"]),
			GatewayID: filters["gateway_id"], Mode: payment.Mode(filters["mode"]), Page: page, PerPage: perPage}
		switch sortCol {
		case "money":
			q.Sort = "amount"
		case "status":
			q.Sort = "status"
		case "created":
			q.Sort = "created_at"
		}
		q.Desc = sortDir == "desc"
		list, total, err := m.Payments(ctx.GoContext(), q)
		if err != nil {
			return nil, 0, err
		}
		names := gatewayNames(ctx.GoContext(), m)
		rows := make([]*PaymentRow, len(list))
		for i, p := range list {
			rows[i] = paymentRow(m, ctx.GoContext(), p, names)
		}
		return rows, total, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*PaymentRow, error) {
		p, err := m.Payment(ctx.GoContext(), id)
		if err != nil {
			return nil, fmt.Errorf("record not found")
		}
		return paymentRow(m, ctx.GoContext(), p, gatewayNames(ctx.GoContext(), m)), nil
	})
	res.NavigationBadge(func(ctx *filament.Context) string {
		_, n, err := m.Payments(ctx.GoContext(), payment.PaymentQuery{Status: payment.StatusAuthorized, PerPage: 1})
		if err != nil || n == 0 {
			return ""
		}
		return fmt.Sprint(n) // authorized payments waiting for capture
	})
	return res
}
