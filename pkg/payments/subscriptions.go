package payments

import (
	"context"
	"fmt"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/payment"
)

// SubscriptionRow is a subscription as the panel shows it.
type SubscriptionRow struct {
	*payment.Subscription
	GatewayName string `json:"gateway_name"`
	Money       string `json:"money"`
	ScheduleTxt string `json:"schedule"`
	StatusLabel string `json:"status_label"`
	BillingTxt  string `json:"billing_label"`
	NextBilling string `json:"next_billing"`
	Created     string `json:"created"`
	Progress    string `json:"progress"`
	features    payment.Features
}

// MethodRow is a saved payment method as the panel shows it (never its token).
type MethodRow struct {
	*payment.PaymentMethod
	GatewayName string `json:"gateway_name"`
	DisplayTxt  string `json:"display"`
	StatusLabel string `json:"status_label"`
	Created     string `json:"created"`
}

var subStatusColors = map[string]string{"pending": "warning", "active": "success", "past_due": "danger",
	"canceled": "gray", "completed": "info", "failed": "danger"}

func subStatusLabels() map[string]string {
	out := map[string]string{}
	for _, s := range payment.SubscriptionStatuses() {
		out[string(s)] = s.Label()
	}
	return out
}

var methodStatusColors = map[string]string{"pending": "warning", "active": "success", "failed": "danger", "revoked": "gray"}

func methodStatusLabels() map[string]string {
	out := map[string]string{}
	for _, s := range []payment.MethodStatus{payment.MethodPending, payment.MethodActive, payment.MethodFailed, payment.MethodRevoked} {
		out[string(s)] = s.Label()
	}
	return out
}

func subscriptionRow(m *payment.Manager, s *payment.Subscription, names map[string]string) *SubscriptionRow {
	row := &SubscriptionRow{Subscription: s, GatewayName: payment.Pick(names[s.GatewayID], s.GatewayID), Money: s.Money(),
		ScheduleTxt: s.Schedule(), StatusLabel: s.Status.Label(), Created: s.CreatedAt.Local().Format(timeLayout),
		BillingTxt: map[payment.Billing]string{payment.BillingGateway: "By the gateway", payment.BillingManager: "Saved card"}[s.Billing]}
	if s.NextBillingAt != nil && !s.Status.Final() {
		row.NextBilling = s.NextBillingAt.Local().Format(timeLayout)
	}
	row.Progress = fmt.Sprintf("%d paid", s.CyclesPaid)
	if s.Cycles > 0 {
		row.Progress = fmt.Sprintf("%d of %d paid", s.CyclesPaid, s.Cycles)
	}
	if d, ok := m.Registry().Driver(s.Driver); ok {
		row.features = d.Features
	}
	return row
}

func methodRow(pm *payment.PaymentMethod, names map[string]string) *MethodRow {
	return &MethodRow{PaymentMethod: pm, GatewayName: payment.Pick(names[pm.GatewayID], pm.GatewayID),
		DisplayTxt: pm.Display(), StatusLabel: pm.Status.Label(), Created: pm.CreatedAt.Local().Format(timeLayout)}
}

func historyRelation(m *payment.Manager) filament.RelationManager {
	return filament.NewRelationManager[*LogRow]("history", "History").
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
					change = label(string(l.To))
					if l.From != "" {
						change = label(string(l.From)) + " → " + change
					}
				}
				rows = append(rows, &LogRow{Log: l, Change: change, At: l.CreatedAt.Local().Format("2006-01-02 15:04:05")})
			}
			return pageOf(rows, page, perPage), len(rows), nil
		})
}

// label words any status (payments, subscriptions and methods share logs).
func label(s string) string {
	return payment.SubscriptionStatus(s).Label() // "past_due" → "Past due"
}

// SubscriptionsResource is the "Subscriptions" resource.
func SubscriptionsResource(m *payment.Manager) *filament.BaseResource[*SubscriptionRow] {
	res := filament.NewResource[*SubscriptionRow]("subscriptions").
		ModelName("Subscription").
		PluralModelName("Subscriptions").
		NavigationGroup(Group).
		NavigationIcon("heroicon-arrow-path-rounded-square").
		NavigationSort(92).
		RecordID(func(s *SubscriptionRow) string { return s.ID }).
		CanCreate(func(*filament.Context) bool { return false }).
		CanEdit(func(*filament.Context, *SubscriptionRow) bool { return false }).
		CanDelete(func(*filament.Context, *SubscriptionRow) bool { return false }).
		CanDeleteAny(func(*filament.Context) bool { return false })

	res.Table(func(t *tables.Table[*SubscriptionRow]) *tables.Table[*SubscriptionRow] {
		return t.Columns(
			tables.TextColumn[*SubscriptionRow]("reference").Label("Plan").Weight("bold").Searchable().Copyable(),
			tables.TextColumn[*SubscriptionRow]("money").Label("Amount"),
			tables.TextColumn[*SubscriptionRow]("schedule").Label("Schedule"),
			tables.BadgeColumn[*SubscriptionRow]("status").Label("Status").Colors(subStatusColors).Labels(subStatusLabels()),
			tables.TextColumn[*SubscriptionRow]("customer_email").Label("Customer").Searchable(),
			tables.TextColumn[*SubscriptionRow]("next_billing").Label("Next billing"),
			tables.TextColumn[*SubscriptionRow]("progress").Label("Paid"),
			tables.TextColumn[*SubscriptionRow]("gateway_name").Label("Gateway").ToggledHiddenByDefault(true),
			tables.TextColumn[*SubscriptionRow]("billing_label").Label("Billed").ToggledHiddenByDefault(true),
			tables.TextColumn[*SubscriptionRow]("created").Label("Created").ToggledHiddenByDefault(true),
		).Filters(
			tables.SelectFilter("status").Label("Status").Options(subStatusLabels()),
			tables.SelectFilter("billing").Label("Billed").Options(map[string]string{
				string(payment.BillingGateway): "By the gateway", string(payment.BillingManager): "Saved card"}),
		).Actions(
			tables.ViewAction[*SubscriptionRow](),
			tables.Action[*SubscriptionRow]("retry").Label("Retry payment").Icon("heroicon-arrow-path").Color("primary").
				Visible(func(s *SubscriptionRow) bool { return s.Status == payment.SubPastDue || s.Status == payment.SubFailed }).
				RequiresConfirmation().ModalDescription("Charge the failed renewal again now.").
				Handle(func(ctx context.Context, s *SubscriptionRow, _ map[string]any) error {
					_, err := m.RetrySubscription(ctx, s.ID)
					return trimErr(err)
				}).SuccessMessage("Retried"),
			tables.Action[*SubscriptionRow]("sync").Label("Sync with gateway").Icon("heroicon-arrow-path").
				Visible(func(s *SubscriptionRow) bool { return s.Billing == payment.BillingGateway && !s.Status.Final() }).
				Handle(func(ctx context.Context, s *SubscriptionRow, _ map[string]any) error {
					_, err := m.SyncSubscription(ctx, s.ID)
					return trimErr(err)
				}).SuccessMessage("Synced"),
			tables.Action[*SubscriptionRow]("cancel").Label("Cancel").Icon("heroicon-x-circle").Color("danger").
				Visible(func(s *SubscriptionRow) bool { return !s.Status.Final() }).
				RequiresConfirmation().ModalDescription("Stop billing this subscription. This cannot be undone.").
				Handle(func(ctx context.Context, s *SubscriptionRow, _ map[string]any) error {
					_, err := m.CancelSubscription(ctx, s.ID)
					return trimErr(err)
				}).SuccessMessage("Canceled"),
		)
	})

	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Section("Subscription").Columns(2).Schema(
				forms.TextInput("reference").Label("Plan").Disabled(),
				forms.TextInput("status_label").Label("Status").Disabled(),
				forms.TextInput("money").Label("Amount").Disabled(),
				forms.TextInput("schedule").Label("Schedule").Disabled(),
				forms.TextInput("progress").Label("Paid").Disabled(),
				forms.TextInput("next_billing").Label("Next billing").Disabled(),
				forms.TextInput("billing_label").Label("Billed").Disabled(),
				forms.TextInput("gateway_name").Label("Gateway").Disabled(),
				forms.TextInput("failure_reason").Label("Last failure").Disabled(),
				forms.TextInput("created").Label("Created").Disabled(),
			),
			forms.Section("Customer").Columns(3).Schema(
				forms.TextInput("customer_name").Label("Name").Disabled(),
				forms.TextInput("customer_email").Label("Email").Disabled(),
				forms.TextInput("customer_ref").Label("Customer ref").Disabled(),
			),
			forms.Section("Reference").Columns(2).Schema(
				forms.TextInput("id").Label("Subscription ID").Disabled(),
				forms.TextInput("gateway_ref").Label("Gateway reference").Disabled(),
			),
		)
	})
	res.Infolist(func(i *infolists.Infolist) *infolists.Infolist {
		return i.Schema(
			infolists.Section("Subscription").Columns(3).Schema(
				infolists.TextEntry("reference").Label("Plan").Weight("bold").Copyable(),
				infolists.TextEntry("money").Label("Amount").Weight("bold"),
				infolists.BadgeEntry("status").Colors(subStatusColors),
				infolists.TextEntry("schedule").Label("Schedule"),
				infolists.TextEntry("progress").Label("Paid"),
				infolists.TextEntry("next_billing").Label("Next billing"),
				infolists.TextEntry("billing_label").Label("Billed"),
				infolists.TextEntry("gateway_name").Label("Gateway"),
				infolists.TextEntry("failure_reason").Label("Last failure"),
			),
			infolists.Section("Customer").Columns(3).Schema(
				infolists.TextEntry("customer_name").Label("Name"),
				infolists.TextEntry("customer_email").Label("Email").Copyable(),
				infolists.TextEntry("customer_ref").Label("Customer ref"),
			),
		)
	})

	periods := filament.NewRelationManager[*PaymentRow]("payments", "Payments").
		Icon("heroicon-banknotes").
		RecordID(func(p *PaymentRow) string { return p.ID }).
		Table(func(t *tables.Table[*PaymentRow]) *tables.Table[*PaymentRow] {
			return t.Columns(
				tables.TextColumn[*PaymentRow]("created").Label("Date"),
				tables.TextColumn[*PaymentRow]("reference").Label("Reference"),
				tables.TextColumn[*PaymentRow]("money").Label("Amount"),
				tables.BadgeColumn[*PaymentRow]("status").Label("Status").Colors(statusColors).Labels(statusLabels()),
				tables.TextColumn[*PaymentRow]("failure_reason").Label("Failure"),
			)
		}).
		OnList(func(ctx *filament.Context, parentID, _, _, _ string, page, perPage int) ([]*PaymentRow, int, error) {
			list, n, err := m.SubscriptionPayments(ctx.GoContext(), parentID, page, perPage)
			if err != nil {
				return nil, 0, err
			}
			names := gatewayNames(ctx.GoContext(), m)
			rows := make([]*PaymentRow, len(list))
			for i, p := range list {
				rows[i] = paymentRow(m, ctx.GoContext(), p, names)
			}
			return rows, n, nil
		})
	res.Relations(periods, historyRelation(m))

	res.OnList(func(ctx *filament.Context, search, _, _ string, page, perPage int, filters map[string]string) ([]*SubscriptionRow, int, error) {
		list, n, err := m.Subscriptions(ctx.GoContext(), payment.SubscriptionQuery{Search: search,
			Status: payment.SubscriptionStatus(filters["status"]), Billing: payment.Billing(filters["billing"]),
			Page: page, PerPage: perPage})
		if err != nil {
			return nil, 0, err
		}
		names := gatewayNames(ctx.GoContext(), m)
		rows := make([]*SubscriptionRow, len(list))
		for i, s := range list {
			rows[i] = subscriptionRow(m, s, names)
		}
		return rows, n, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*SubscriptionRow, error) {
		s, err := m.Subscription(ctx.GoContext(), id)
		if err != nil {
			return nil, fmt.Errorf("record not found")
		}
		return subscriptionRow(m, s, gatewayNames(ctx.GoContext(), m)), nil
	})
	res.NavigationBadge(func(ctx *filament.Context) string {
		_, n, err := m.Subscriptions(ctx.GoContext(), payment.SubscriptionQuery{Status: payment.SubPastDue, PerPage: 1})
		if err != nil || n == 0 {
			return ""
		}
		return fmt.Sprint(n) // past due
	})
	return res
}

// MethodsResource is the "Saved Payment Methods" resource.
func MethodsResource(m *payment.Manager) *filament.BaseResource[*MethodRow] {
	res := filament.NewResource[*MethodRow]("payment-methods").
		ModelName("Saved Payment Method").
		PluralModelName("Saved Payment Methods").
		NavigationGroup(Group).
		NavigationIcon("heroicon-wallet").
		NavigationSort(93).
		RecordID(func(r *MethodRow) string { return r.ID }).
		CanCreate(func(*filament.Context) bool { return false }).
		CanEdit(func(*filament.Context, *MethodRow) bool { return false }).
		CanDelete(func(*filament.Context, *MethodRow) bool { return false }).
		CanDeleteAny(func(*filament.Context) bool { return false })

	res.Table(func(t *tables.Table[*MethodRow]) *tables.Table[*MethodRow] {
		return t.Columns(
			tables.TextColumn[*MethodRow]("display").Label("Method").Weight("bold"),
			tables.TextColumn[*MethodRow]("customer_email").Label("Customer").Searchable(),
			tables.TextColumn[*MethodRow]("customer_ref").Label("Customer ref").Searchable(),
			tables.BadgeColumn[*MethodRow]("status").Label("Status").Colors(methodStatusColors).Labels(methodStatusLabels()),
			tables.TextColumn[*MethodRow]("gateway_name").Label("Gateway"),
			tables.TextColumn[*MethodRow]("created").Label("Saved"),
		).Filters(
			tables.SelectFilter("status").Label("Status").Options(methodStatusLabels()),
		).Actions(
			tables.ViewAction[*MethodRow](),
			tables.Action[*MethodRow]("revoke").Label("Revoke").Icon("heroicon-no-symbol").Color("danger").
				Visible(func(r *MethodRow) bool { return r.Status == payment.MethodActive }).
				RequiresConfirmation().ModalDescription("The card can no longer be charged. Subscriptions billed to it will fail their next renewal.").
				Handle(func(ctx context.Context, r *MethodRow, _ map[string]any) error {
					_, err := m.RevokePaymentMethod(ctx, r.ID)
					return trimErr(err)
				}).SuccessMessage("Revoked"),
		)
	})
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(forms.Section("Saved payment method").Columns(2).Schema(
			forms.TextInput("display").Label("Method").Disabled(),
			forms.TextInput("status_label").Label("Status").Disabled(),
			forms.TextInput("customer_name").Label("Customer").Disabled(),
			forms.TextInput("customer_email").Label("Email").Disabled(),
			forms.TextInput("customer_ref").Label("Customer ref").Disabled(),
			forms.TextInput("gateway_name").Label("Gateway").Disabled(),
			forms.TextInput("failure_reason").Label("Failure").Disabled(),
			forms.TextInput("created").Label("Saved").Disabled(),
		))
	})
	res.Infolist(func(i *infolists.Infolist) *infolists.Infolist {
		return i.Schema(infolists.Section("Saved payment method").Columns(3).Schema(
			infolists.TextEntry("display").Label("Method").Weight("bold"),
			infolists.BadgeEntry("status").Colors(methodStatusColors),
			infolists.TextEntry("gateway_name").Label("Gateway"),
			infolists.TextEntry("customer_name").Label("Customer"),
			infolists.TextEntry("customer_email").Label("Email").Copyable(),
			infolists.TextEntry("customer_ref").Label("Customer ref"),
			infolists.TextEntry("failure_reason").Label("Failure"),
			infolists.TextEntry("created").Label("Saved"),
		))
	})
	res.Relations(historyRelation(m))
	res.OnList(func(ctx *filament.Context, search, _, _ string, page, perPage int, filters map[string]string) ([]*MethodRow, int, error) {
		list, n, err := m.PaymentMethods(ctx.GoContext(), payment.MethodQuery{Search: search,
			Status: payment.MethodStatus(filters["status"]), Page: page, PerPage: perPage})
		if err != nil {
			return nil, 0, err
		}
		names := gatewayNames(ctx.GoContext(), m)
		rows := make([]*MethodRow, len(list))
		for i, pm := range list {
			rows[i] = methodRow(pm, names)
		}
		return rows, n, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*MethodRow, error) {
		pm, err := m.PaymentMethod(ctx.GoContext(), id)
		if err != nil {
			return nil, fmt.Errorf("record not found")
		}
		return methodRow(pm, gatewayNames(ctx.GoContext(), m)), nil
	})
	return res
}
