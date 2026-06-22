package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
)

// OIDCOptions configures OIDC discovery and token verification.
type OIDCOptions struct {
	// InsecureSkipTLSVerify disables TLS certificate verification when fetching
	// OIDC discovery documents and JWKS. For local/dev use only.
	InsecureSkipTLSVerify bool
}

// OIDCValidator validates bearer tokens against an OIDC issuer's JWKS.
type OIDCValidator struct {
	verifier *oidc.IDTokenVerifier
}

// NewOIDCValidator discovers the issuer's JWKS endpoint and builds a verifier.
// audience is optional; when set it is checked against the token's aud claim.
func NewOIDCValidator(ctx context.Context, issuer, audience string, opts OIDCOptions) (*OIDCValidator, error) {
	if opts.InsecureSkipTLSVerify {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // dev-only opt-in
		}
		ctx = oidc.ClientContext(ctx, &http.Client{Transport: transport})
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	verifierCfg := &oidc.Config{}
	if audience != "" {
		verifierCfg.ClientID = audience
	} else {
		verifierCfg.SkipClientIDCheck = true
	}

	return &OIDCValidator{
		verifier: provider.Verifier(verifierCfg),
	}, nil
}

func (v *OIDCValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr, err := extractBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		idToken, err := v.verifier.Verify(r.Context(), tokenStr)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		claims := &Claims{}
		if err := idToken.Claims(claims); err != nil {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), BilletsContextKey, claims.Billets)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
