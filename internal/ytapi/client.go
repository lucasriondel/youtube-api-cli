package ytapi

import (
	"context"
	"fmt"

	"github.com/lucasrndl/yt/internal/auth"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// New builds an authenticated YouTube service from the stored token.
// The OAuth2 client transparently refreshes the access token when needed,
// and the new token is persisted back to storage.
func New(ctx context.Context) (*youtube.Service, error) {
	cfg, err := auth.LoadConfig(auth.DefaultScopes())
	if err != nil {
		return nil, err
	}
	tok, err := auth.LoadToken()
	if err != nil {
		return nil, err
	}

	src := &persistingTokenSource{base: cfg.TokenSource(ctx, tok), prev: tok}
	svc, err := youtube.NewService(ctx, option.WithTokenSource(src))
	if err != nil {
		return nil, fmt.Errorf("youtube client: %w", err)
	}
	return svc, nil
}
