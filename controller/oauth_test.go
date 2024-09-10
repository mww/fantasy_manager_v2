package controller

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
	"github.com/mww/fantasy_manager_v2/testutils"
)

func TestOAuthFlow(t *testing.T) {
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	l, err := ctrl.AddLeague(ctx, model.PlatformSleeper, testutils.SleeperLeagueID, "2024", "" /* state */)
	if err != nil {
		t.Fatalf("unexpected error adding league: %v", err)
	}

	authURL, err := ctrl.OAuthStart(model.PlatformYahoo)
	state := validateOAuthStart(t, authURL, err)

	if err := ctrl.OAuthExchange(ctx, state, "code"); err != nil {
		t.Fatalf("unexpected error in OAuthExchange: %v", err)
	}

	token, err := ctrl.OAuthRetrieve(state)
	if err != nil {
		t.Fatalf("unexpected error retrieving oauth token: %v", err)
	}
	if token.AccessToken != "access_token" {
		t.Errorf("access token value not as expected, got: %s", token.AccessToken)
	}
	if token.RefreshToken != "refresh_token" {
		t.Errorf("refresh token value not as expected, got: %s", token.RefreshToken)
	}
	if token.Expiry.IsZero() || token.Expiry.Before(time.Now()) {
		t.Error("token expiry time is not in the future!")
	}

	if err := ctrl.OAuthSave(ctx, state, l.ID); err != nil {
		t.Fatalf("error saving oauth token: %v", err)
	}

	t2, err := ctrl.GetToken(ctx, l.ID)
	if err != nil {
		t.Fatalf("error getting token: %v", err)
	}
	if t2.AccessToken != "access_token" {
		t.Errorf("t2 access token value not as expected, got: %s", t2.AccessToken)
	}
	if t2.RefreshToken != "refresh_token" {
		t.Errorf("t2 refresh token value not as expected, got: %s", t2.RefreshToken)
	}
	if t2.Expiry.IsZero() || t2.Expiry.Before(time.Now()) {
		t.Error("t2 token expiry time is not in the future!")
	}
}

func TestOAuthServerStart_unsupportedPlatform(t *testing.T) {
	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	_, err := ctrl.OAuthStart("ESPN")
	if err == nil {
		t.Fatal("expected an error but did not get one")
	}
}

func TestOAuth_stateExpired(t *testing.T) {
	ctx := context.Background()

	ctrl, testCtrl := controllerForTest()
	defer testCtrl.Close()

	authURL, err := ctrl.OAuthStart(model.PlatformYahoo)
	state := validateOAuthStart(t, authURL, err)

	testCtrl.Clock.Add(6 * time.Minute)
	err = ctrl.OAuthExchange(ctx, state, "code")
	if err == nil || err.Error() != "state is not valid" {
		t.Errorf("expected error but got wrong value: %v", err)
	}
}

func validateOAuthStart(t *testing.T, auth string, err error) string {
	if err != nil {
		t.Fatalf("unexpected error in OAuthStart: %v", err)
	}
	if !strings.Contains(auth, "/auth") {
		t.Errorf("expected url to have a specific prefix, got: %s", auth)
	}

	u, err := url.Parse(auth)
	if err != nil {
		t.Fatalf("error parsing authURL: %v", err)
	}
	state := u.Query().Get("state")
	if state == "" {
		t.Fatalf("no state encoded in authURL: %s", auth)
	}

	return state
}
