package tenant

import (
	"fmt"
	"net/http"

	"github.com/bradtumy/credential-service/internal/domain"
)

// ErrTenantDisabled indicates the tenant is present but disabled.
var ErrTenantDisabled = fmt.Errorf("tenant disabled")

// Resolver determines the tenant for an incoming HTTP request.
type Resolver struct {
	Mode            Mode
	DefaultTenantID string
	Store           Store
}

// Resolve extracts the tenant ID from request headers or falls back to defaults.
func (r Resolver) Resolve(req *http.Request) (domain.Tenant, error) {
	if r.Store == nil {
		return domain.Tenant{}, fmt.Errorf("tenant store not configured")
	}

	tenantID := req.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		if r.Mode == ModeSingle {
			tenantID = r.DefaultTenantID
		} else {
			return domain.Tenant{}, fmt.Errorf("tenant header required")
		}
	}

	tenant, err := r.Store.GetTenant(req.Context(), tenantID)
	if err != nil {
		return domain.Tenant{}, err
	}
	if !tenant.Enabled {
		return domain.Tenant{}, ErrTenantDisabled
	}

	return tenant, nil
}
