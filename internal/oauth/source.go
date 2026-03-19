package oauth

import (
	"context"

	"golang.org/x/oauth2"
)

// DBTokenSource implements oauth2.TokenSource with database-backed storage.
// It automatically refreshes expired tokens and persists the refreshed tokens.
type DBTokenSource struct {
	store     *TokenStore
	cfg       *oauth2.Config
	accountID string
	provider  string
}

// NewDBTokenSource creates a new database-backed token source.
func NewDBTokenSource(store *TokenStore, cfg *oauth2.Config, accountID, provider string) oauth2.TokenSource {
	dts := &DBTokenSource{
		store:     store,
		cfg:       cfg,
		accountID: accountID,
		provider:  provider,
	}

	// Wrap with ReuseTokenSource for efficiency - it caches valid tokens in memory
	// and only calls our Token() method when the token is expired or not cached.
	return oauth2.ReuseTokenSource(nil, dts)
}

// Token implements oauth2.TokenSource.
// It loads the token from the database, checks expiry, refreshes if needed, and saves the new token.
func (dts *DBTokenSource) Token() (*oauth2.Token, error) {
	ctx := context.Background()

	// Load token from database
	token, err := dts.store.Load(ctx, dts.accountID, dts.provider)
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, nil
	}

	// Check if token is expired or will expire soon (within 60 seconds)
	if token.Valid() {
		return token, nil
	}

	// Token is expired - refresh it
	if token.RefreshToken == "" {
		// No refresh token available
		return token, nil
	}

	// Create a token source from the refresh token and get a new token
	tokenSource := dts.cfg.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	// Save the new token to the database
	if err := dts.store.Save(ctx, dts.accountID, dts.provider, newToken); err != nil {
		return nil, err
	}

	return newToken, nil
}
