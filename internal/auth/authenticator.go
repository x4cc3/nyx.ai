package auth

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Identity struct {
	Subject  string
	Email    string
	TenantID string
	Claims   map[string]any
}

type Authenticator interface {
	Verify(ctx context.Context, token string) (Identity, error)
}

type SupabaseAuthenticator struct {
	verifier tokenVerifier
}

type tokenVerifier interface {
	Verify(context.Context, string) (verifiedToken, error)
}

type verifiedToken interface {
	Claims(any) error
	Subject() string
}

type oidcTokenVerifier struct {
	inner *oidc.IDTokenVerifier
}

type oidcVerifiedToken struct {
	inner *oidc.IDToken
}

func NewSupabaseAuthenticator(ctx context.Context, supabaseURL, audience string) (*SupabaseAuthenticator, error) {
	issuer, err := issuerFromSupabaseURL(supabaseURL)
	if err != nil {
		return nil, err
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("create oidc provider: %w", err)
	}
	return &SupabaseAuthenticator{
		verifier: oidcTokenVerifier{inner: provider.Verifier(&oidc.Config{ClientID: strings.TrimSpace(audience)})},
	}, nil
}

func (a *SupabaseAuthenticator) Verify(ctx context.Context, token string) (Identity, error) {
	idToken, err := a.verifier.Verify(ctx, strings.TrimSpace(token))
	if err != nil {
		return Identity{}, fmt.Errorf("verify bearer token: %w", err)
	}

	claims := make(map[string]any)
	if err := idToken.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("decode token claims: %w", err)
	}

	identity := Identity{
		Subject: idToken.Subject(),
		Claims:  claims,
	}
	identity.Email = claimString(claims, "email")
	identity.TenantID = firstNonEmpty(
		claimString(claims, "tenant_id"),
		nestedClaimString(claims, "app_metadata", "tenant_id"),
		nestedClaimString(claims, "user_metadata", "tenant_id"),
	)
	return identity, nil
}

func (v oidcTokenVerifier) Verify(ctx context.Context, token string) (verifiedToken, error) {
	idToken, err := v.inner.Verify(ctx, token)
	if err != nil {
		return nil, err
	}
	return oidcVerifiedToken{inner: idToken}, nil
}

func (t oidcVerifiedToken) Claims(dest any) error {
	return t.inner.Claims(dest)
}

func (t oidcVerifiedToken) Subject() string {
	return t.inner.Subject
}

func issuerFromSupabaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("supabase url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse supabase url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("supabase url must be http(s), got %q", u.Scheme)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/auth/v1"
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/"), nil
}

func claimString(claims map[string]any, key string) string {
	value, ok := claims[key]
	if !ok {
		return ""
	}
	stringValue, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringValue)
}

func nestedClaimString(claims map[string]any, parentKey, childKey string) string {
	parent, ok := claims[parentKey]
	if !ok {
		return ""
	}
	typed, ok := parent.(map[string]any)
	if !ok {
		return ""
	}
	return claimString(typed, childKey)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
