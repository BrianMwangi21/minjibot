// Package auth provides Discord OAuth 2.0 authentication and signed-cookie
// session handling for the web dashboard.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
)

const (
	discordAuthorizeEndpoint = "https://discord.com/oauth2/authorize"
	discordTokenEndpoint     = "https://discord.com/api/v10/oauth2/token"
	discordAPIBase           = "https://discord.com/api/v10"
	callbackPath             = "/api/auth/callback/discord"
)

// DiscordUser is the subset of the Discord /users/@me payload we care about.
type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Email         string `json:"email"`
	Verified      bool   `json:"verified"`
}

// DiscordOAuth wraps the oauth2 configuration for the Discord OAuth 2.0 flow.
type DiscordOAuth struct {
	Config      *oauth2.Config
	ClientID    string
	ClientToken *http.Client
}

// Permissions is a Discord permission bitfield. Discord serializes permission
// bitsets as decimal strings in some endpoints and as numbers in others, so we
// accept both when unmarshalling.
type Permissions int64

// UnmarshalJSON accepts both a JSON number and a quoted decimal string.
func (p *Permissions) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("parse permission bits %q: %w", s, err)
	}
	*p = Permissions(v)
	return nil
}

// Guild is one entry of the /users/@me/guilds response: a guild the authorized
// user belongs to, with their guild-level permission bits.
type Guild struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Owner       bool        `json:"owner"`
	Permissions Permissions `json:"permissions"`
}

var discordEndpoint = oauth2.Endpoint{
	AuthURL:  discordAuthorizeEndpoint,
	TokenURL: discordTokenEndpoint,
}

// NewDiscordOAuth builds a DiscordOAuth using the given client id/secret and
// the base URL of the app (used to derive the callback redirect URI).
func NewDiscordOAuth(clientID, clientSecret, appURL string) *DiscordOAuth {
	redirectURI := strings.TrimRight(appURL, "/") + callbackPath
	return &DiscordOAuth{
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     discordEndpoint,
			RedirectURL:  redirectURI,
			Scopes:       []string{"identify", "email", "guilds"},
		},
		ClientID:    clientID,
		ClientToken: &http.Client{},
	}
}

// AuthURL returns the Discord authorization URL for a fresh login, carrying the
// given anti-CSRF state token (echoed back to the callback by Discord).
func (d *DiscordOAuth) AuthURL(state string) string {
	return d.Config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// Exchange trades the authorization code from the callback for an access
// token and fetches the authenticated user's profile. The token is returned so
// callers can persist it for later Discord API calls (e.g. per-guild dashboard
// authorization via /users/@me/guilds).
func (d *DiscordOAuth) Exchange(ctx context.Context, code string) (*DiscordUser, *oauth2.Token, error) {
	tok, err := d.Config.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("exchange authorization code: %w", err)
	}
	u, err := d.FetchUser(ctx, tok)
	if err != nil {
		return nil, nil, err
	}
	return u, tok, nil
}

// UserGuilds returns the guilds the authorized user belongs to, each with the
// user's guild-level permission bits. Requires the "guilds" OAuth scope.
func (d *DiscordOAuth) UserGuilds(ctx context.Context, accessToken string) ([]Guild, error) {
	if accessToken == "" {
		return nil, errors.New("missing access token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discordAPIBase+"/users/@me/guilds", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := d.ClientToken.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("discord /users/@me/guilds returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var guilds []Guild
	if err := json.NewDecoder(resp.Body).Decode(&guilds); err != nil {
		return nil, err
	}
	return guilds, nil
}

// FetchUser retrieves the /users/@me profile for the given token.
func (d *DiscordOAuth) FetchUser(ctx context.Context, tok *oauth2.Token) (*DiscordUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discordAPIBase+"/users/@me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := d.ClientToken.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("discord /users/@me returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var u DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	if u.ID == "" {
		return nil, errors.New("discord /users/@me returned no user id")
	}
	return &u, nil
}

// RevokeToken invalidates the given token on Discord. Errors are returned as
// an *oauth2.RetrieveError when networking permits, otherwise lenient.
func (d *DiscordOAuth) RevokeToken(ctx context.Context, tok *oauth2.Token) error {
	if tok == nil || tok.AccessToken == "" {
		return nil
	}
	form := url.Values{}
	form.Set("token", tok.AccessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://discord.com/api/v10/oauth2/token/revoke",
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := d.ClientToken.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("discord token revoke returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
