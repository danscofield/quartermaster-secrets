package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const BilletsContextKey contextKey = "billets"

// Claims holds JWT claims from the IdP.
type Claims struct {
	jwt.RegisteredClaims
	Billets []string `json:"billets"`
}

func extractBearerToken(header string) (string, error) {
	if header == "" {
		return "", fmt.Errorf("missing authorization header")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", fmt.Errorf("invalid authorization header")
	}
	return parts[1], nil
}

// BilletsFromContext returns the caller's billets from the request context.
func BilletsFromContext(ctx context.Context) []string {
	billets, _ := ctx.Value(BilletsContextKey).([]string)
	return billets
}
