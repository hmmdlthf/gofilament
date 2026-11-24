// Package payments puts golaravel's payment gateways (pkg/payment) in a
// panel:
//
//   - "Payment Gateways": configure any installed driver (Stripe, PayHere,
//     OnePay, Tabby, Tamara, the test gateway or your own plugin). The form
//     shows the selected driver's own settings, keeps secrets write-only, and
//     each gateway has a "Test connection" action and shows its webhook URL.
//
//   - "Payments": every payment, with filters by status, gateway and mode, its
//     history, and Refund / Capture / Void / Sync actions where the gateway
//     supports them.
//
//     payments.Register(panel, mgr)
//
// The resources are returned too, so access can be narrowed like any other
// resource (CanViewAny, AuthorizeWithGate, Policy, ...).
package payments

import (
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/payment"
)

// Group is the navigation group both resources go in.
var Group = "Payments"

// Resources are the two payment resources.
type Resources struct {
	Gateways *filament.BaseResource[*GatewayRow]
	Payments *filament.BaseResource[*PaymentRow]
}

// Register adds the gateway and payment resources to a panel.
func Register(p *filament.Panel, m *payment.Manager) Resources {
	r := Resources{Gateways: GatewaysResource(m), Payments: PaymentsResource(m)}
	p.Resource(r.Gateways).Resource(r.Payments)
	return r
}
