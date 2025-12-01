package httpx

import (
	"context"
	"net/http"

	"github.com/bradtumy/credential-service/internal/tenant"
)

// TenantMiddleware enforces tenant resolution rules before reaching handlers.
func TenantMiddleware(resolver tenant.Resolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tnt, err := resolver.Resolve(r)
		if err != nil {
			status := http.StatusForbidden
			code := "tenant_error"
			switch err {
			case tenant.ErrTenantNotFound:
				status = http.StatusNotFound
				code = "tenant_not_found"
			case tenant.ErrTenantDisabled:
				code = "tenant_disabled"
			default:
				code = "tenant_required"
				if resolver.Mode == tenant.ModeSingle {
					status = http.StatusInternalServerError
				} else {
					status = http.StatusBadRequest
				}
			}
			WriteAPIError(w, status, code, err.Error())
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, tenantIDKey, tnt.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
