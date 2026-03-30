package auth

import (
	"context"
	"errors"
	"testing"
)

type fakeVerifier struct {
	token verifiedToken
	err   error
}

type fakeToken struct {
	subject string
	claims  map[string]any
	err     error
}

func (v fakeVerifier) Verify(context.Context, string) (verifiedToken, error) {
	if v.err != nil {
		return nil, v.err
	}
	return v.token, nil
}

func (t fakeToken) Claims(dest any) error {
	if t.err != nil {
		return t.err
	}
	out, ok := dest.(*map[string]any)
	if !ok {
		return errors.New("unexpected claims target")
	}
	clone := make(map[string]any, len(t.claims))
	for key, value := range t.claims {
		clone[key] = value
	}
	*out = clone
	return nil
}

func (t fakeToken) Subject() string {
	return t.subject
}

func TestIssuerFromSupabaseURL(t *testing.T) {
	got, err := issuerFromSupabaseURL("https://demo.supabase.co/")
	if err != nil {
		t.Fatalf("issuerFromSupabaseURL: %v", err)
	}
	if got != "https://demo.supabase.co/auth/v1" {
		t.Fatalf("unexpected issuer: %q", got)
	}
}

func TestIssuerFromSupabaseURLRejectsInvalidInput(t *testing.T) {
	for _, raw := range []string{"", "ftp://demo.supabase.co", "://bad"} {
		if _, err := issuerFromSupabaseURL(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

func TestSupabaseAuthenticatorVerifyExtractsIdentity(t *testing.T) {
	authn := &SupabaseAuthenticator{
		verifier: fakeVerifier{
			token: fakeToken{
				subject: "user-123",
				claims: map[string]any{
					"email": "operator@example.com",
					"app_metadata": map[string]any{
						"tenant_id": "tenant-app",
					},
					"user_metadata": map[string]any{
						"tenant_id": "tenant-user",
					},
				},
			},
		},
	}

	identity, err := authn.Verify(context.Background(), " bearer-token ")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if identity.Subject != "user-123" {
		t.Fatalf("unexpected subject: %q", identity.Subject)
	}
	if identity.Email != "operator@example.com" {
		t.Fatalf("unexpected email: %q", identity.Email)
	}
	if identity.TenantID != "tenant-app" {
		t.Fatalf("expected app metadata tenant, got %q", identity.TenantID)
	}
}

func TestSupabaseAuthenticatorVerifyPrefersTopLevelTenant(t *testing.T) {
	authn := &SupabaseAuthenticator{
		verifier: fakeVerifier{
			token: fakeToken{
				subject: "user-123",
				claims: map[string]any{
					"tenant_id": "tenant-top",
					"app_metadata": map[string]any{
						"tenant_id": "tenant-app",
					},
				},
			},
		},
	}

	identity, err := authn.Verify(context.Background(), "token")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if identity.TenantID != "tenant-top" {
		t.Fatalf("expected top-level tenant, got %q", identity.TenantID)
	}
}

func TestSupabaseAuthenticatorVerifyPropagatesErrors(t *testing.T) {
	authn := &SupabaseAuthenticator{
		verifier: fakeVerifier{err: errors.New("invalid token")},
	}

	if _, err := authn.Verify(context.Background(), "token"); err == nil {
		t.Fatal("expected verify error")
	}

	authn = &SupabaseAuthenticator{
		verifier: fakeVerifier{
			token: fakeToken{
				subject: "user-123",
				err:     errors.New("claims error"),
			},
		},
	}
	if _, err := authn.Verify(context.Background(), "token"); err == nil {
		t.Fatal("expected claims decode error")
	}
}

func TestClaimHelpers(t *testing.T) {
	claims := map[string]any{
		"email": " user@example.com ",
		"nested": map[string]any{
			"tenant_id": " tenant-1 ",
		},
	}

	if got := claimString(claims, "email"); got != "user@example.com" {
		t.Fatalf("unexpected claim value: %q", got)
	}
	if got := nestedClaimString(claims, "nested", "tenant_id"); got != "tenant-1" {
		t.Fatalf("unexpected nested claim value: %q", got)
	}
	if got := firstNonEmpty("", "  ", "value"); got != "value" {
		t.Fatalf("unexpected first non-empty result: %q", got)
	}
}
