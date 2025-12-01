package domain

import (
	"errors"
	"time"
)

const (
	// AdminVCTTL is the time-to-live for admin credentials (6 months)
	AdminVCTTL = 6 * 30 * 24 * time.Hour

	// AdminVCType identifies admin credentials
	AdminVCType = "admin"

	// UserVCType identifies regular user credentials  
	UserVCType = "user"

	// SuperAdminRole grants full system access
	SuperAdminRole = "super-admin"
)

var (
	// ErrNotAdminVC indicates the credential is not an admin credential
	ErrNotAdminVC = errors.New("credential is not an admin VC")

	// ErrInsufficientAdminPrivileges indicates the admin lacks required role
	ErrInsufficientAdminPrivileges = errors.New("insufficient admin privileges")
)

// AdminClaims represents the structure of admin credential claims
type AdminClaims struct {
	VCType     string   `json:"vc_type"`
	AdminRoles []string `json:"admin_roles"`
}

// IsAdminVC checks if a credential has admin claims
func IsAdminVC(claims map[string]interface{}) bool {
	vcType, ok := claims["vc_type"].(string)
	return ok && vcType == AdminVCType
}

// HasAdminRole checks if the claims contain a specific admin role
func HasAdminRole(claims map[string]interface{}, requiredRole string) bool {
	adminRoles, ok := claims["admin_roles"]
	if !ok {
		return false
	}

	// Handle both []string and []interface{} from JSON unmarshaling
	switch roles := adminRoles.(type) {
	case []string:
		for _, role := range roles {
			if role == requiredRole {
				return true
			}
		}
	case []interface{}:
		for _, role := range roles {
			if str, ok := role.(string); ok && str == requiredRole {
				return true
			}
		}
	}
	return false
}

// ValidateAdminVC validates that a credential is a valid admin VC with required role
func ValidateAdminVC(claims map[string]interface{}, requiredRole string) error {
	if !IsAdminVC(claims) {
		return ErrNotAdminVC
	}

	if !HasAdminRole(claims, requiredRole) {
		return ErrInsufficientAdminPrivileges
	}

	return nil
}

// CreateAdminClaims creates a claims map for an admin credential
func CreateAdminClaims(roles []string, additionalClaims map[string]interface{}) map[string]interface{} {
	claims := map[string]interface{}{
		"vc_type":     AdminVCType,
		"admin_roles": roles,
	}

	// Add any additional claims
	for key, value := range additionalClaims {
		claims[key] = value
	}

	return claims
}