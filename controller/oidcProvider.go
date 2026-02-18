package controller

import (
	"context"
	"errors"
	"fmt"
	"go-backend/common"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

const (
	OIDCProviderGoogle = "google"
	OIDCProviderFGF    = "fgf-idp"
)

var (
	ErrOIDCUnsupportedProvider   = errors.New("unsupported oidc provider")
	ErrOIDCProviderNotConfigured = errors.New("oidc provider not configured")
	ErrOIDCNotConfigured         = errors.New("no oidc providers configured")
)

type OIDCProvider struct {
	Name     string
	Issuer   string
	ClientID string
	Verifier *oidc.IDTokenVerifier
}

var (
	oidcOnce      sync.Once
	oidcInitErr   error
	oidcProviders map[string]*OIDCProvider
)

func normalizeOIDCProvider(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	if s == "custom" {
		return OIDCProviderFGF
	}
	return s
}

func isKnownOIDCProvider(name string) bool {
	switch name {
	case OIDCProviderGoogle, OIDCProviderFGF:
		return true
	default:
		return false
	}
}

func issuerFromDiscoveryURL(u string) string {
	const suffix = "/.well-known/openid-configuration"
	if strings.HasSuffix(u, suffix) {
		return strings.TrimSuffix(u, suffix)
	}
	return u
}

func contextForIssuer(ctx context.Context, issuer string) context.Context {
	if strings.HasPrefix(issuer, "http://") {
		return oidc.InsecureIssuerURLContext(ctx, issuer)
	}
	return ctx
}

func initOIDCProviders() error {
	providers := make(map[string]*OIDCProvider)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if common.OIDCGoogleClientID != "" {
		issuer := "https://accounts.google.com"
		p, err := oidc.NewProvider(contextForIssuer(ctx, issuer), issuer)
		if err != nil {
			return fmt.Errorf("google oidc provider init failed: %w", err)
		}
		providers[OIDCProviderGoogle] = &OIDCProvider{
			Name:     OIDCProviderGoogle,
			Issuer:   issuer,
			ClientID: common.OIDCGoogleClientID,
			Verifier: p.Verifier(&oidc.Config{ClientID: common.OIDCGoogleClientID}),
		}
	}

	if common.OIDCCustomDiscoveryURL == "" && common.OIDCCustomClientID == "" {
		// no custom provider configured
	} else if common.OIDCCustomDiscoveryURL == "" || common.OIDCCustomClientID == "" {
		common.SysError("custom OIDC config incomplete: OIDC_CUSTOM_DISCOVERY_URL and OIDC_CUSTOM_CLIENT_ID are required")
	} else {
		issuer := issuerFromDiscoveryURL(common.OIDCCustomDiscoveryURL)
		p, err := oidc.NewProvider(contextForIssuer(ctx, issuer), issuer)
		if err != nil {
			return fmt.Errorf("custom oidc provider init failed: %w", err)
		}
		providers[OIDCProviderFGF] = &OIDCProvider{
			Name:     OIDCProviderFGF,
			Issuer:   issuer,
			ClientID: common.OIDCCustomClientID,
			Verifier: p.Verifier(&oidc.Config{ClientID: common.OIDCCustomClientID}),
		}
	}

	if len(providers) == 0 {
		return ErrOIDCNotConfigured
	}
	oidcProviders = providers
	return nil
}

func GetOIDCProvider(name string) (*OIDCProvider, error) {
	providerName := normalizeOIDCProvider(name)

	oidcOnce.Do(func() {
		oidcInitErr = initOIDCProviders()
	})

	if oidcInitErr != nil {
		return nil, oidcInitErr
	}

	provider, ok := oidcProviders[providerName]
	if !ok {
		if !isKnownOIDCProvider(providerName) {
			return nil, fmt.Errorf("%w: %s", ErrOIDCUnsupportedProvider, providerName)
		}
		return nil, fmt.Errorf("%w: %s", ErrOIDCProviderNotConfigured, providerName)
	}
	return provider, nil
}
