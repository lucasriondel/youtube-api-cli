package ytapi

import (
	"github.com/lucasrndl/yt/internal/auth"
	"golang.org/x/oauth2"
)

// persistingTokenSource wraps an oauth2.TokenSource and writes the token back
// to disk whenever it changes — typically after a refresh.
type persistingTokenSource struct {
	base oauth2.TokenSource
	prev *oauth2.Token
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := p.base.Token()
	if err != nil {
		return nil, err
	}
	if p.prev == nil || tok.AccessToken != p.prev.AccessToken {
		_ = auth.SaveToken(tok)
		p.prev = tok
	}
	return tok, nil
}
