package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"math/rand"

	"github.com/mww/fantasy_manager_v2/model"
	"golang.org/x/oauth2"
)

func (c *controller) OAuthStart(platform string) (string, error) {
	if platform != model.PlatformYahoo {
		return "", errors.New("yahoo is the only supported oauth platform")
	}

	if c.yahooConfig == nil {
		return "", errors.New("yahoo oauth client is not configured")
	}

	state := generateRandomState()
	url := c.yahooConfig.AuthCodeURL(state)
	c.oauthStates[state] = &oauthState{
		platform: platform,
		expiry:   time.Now().Add(5 * time.Minute),
	}
	return url, nil
}

func (c *controller) OAuthExchange(ctx context.Context, state, code string) error {
	s, ok := c.oauthStates[state]
	if !ok || time.Now().After(s.expiry) {
		return errors.New("state is not valid")
	}

	if c.yahooConfig == nil {
		return errors.New("yahoo oauth client is not configured")
	}

	token, err := c.yahooConfig.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("error exchanging code: %w", err)
	}

	s.token = token
	return nil
}

func (c *controller) OAuthRetrieve(state string) (*oauth2.Token, error) {
	s, ok := c.oauthStates[state]
	if !ok || time.Now().After(s.expiry) {
		return nil, errors.New("state parameter is not valid")
	}

	return s.token, nil
}

func (c *controller) OAuthSave(ctx context.Context, state string, leagueID int32) error {
	s, ok := c.oauthStates[state]
	if !ok || time.Now().After(s.expiry) {
		return errors.New("state parameters is not valid")
	}

	return c.db.SaveToken(ctx, leagueID, s.token)
}

func (c *controller) GetToken(ctx context.Context, leagueID int32) (*oauth2.Token, error) {
	t, err := c.db.GetToken(ctx, leagueID)
	if err != nil {
		return nil, err
	}

	// We must manually refresh the token in order to be able
	// to save it back. If we just use yahooOAuth.Client(ctx, t)
	// then it will refresh the token in the background, but never
	// give us access to it.
	// TODO: verify if this is still true, or if the library has been
	// changed to remove this being necessary.
	if t.Expiry.Before(time.Now()) {
		log.Printf("refreshing token for league: %d", leagueID)
		tknSrc := c.yahooConfig.TokenSource(ctx, t)

		t, err = tknSrc.Token()
		if err != nil {
			return nil, fmt.Errorf("error refreshing token for league %d: %w", leagueID, err)
		}

		if err := c.db.SaveToken(ctx, leagueID, t); err != nil {
			return nil, fmt.Errorf("error saving refreshed token for league %d: %w", leagueID, err)
		}
	}

	return t, nil
}

func generateRandomState() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 15)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
