package testutils

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/itbasis/go-clock"
	"golang.org/x/oauth2"
)

type TestController struct {
	Clock       *clock.Mock
	YahooConfig *oauth2.Config
	fakeSleeper *FakeSleeperServer
	fakeYahoo   *FakeYahooServer
	fakeOAuth   *httptest.Server
}

func (c *TestController) Close() {
	c.fakeSleeper.Close()
	c.fakeYahoo.Close()
	c.fakeOAuth.Close()
}

func (c *TestController) SleeperURL() string {
	return c.fakeSleeper.URL()
}

func (c *TestController) YahooURL() string {
	return c.fakeYahoo.URL()
}

func NewTestController(db *TestDB) *TestController {
	fakeOAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"access_token": "access_token",
			"refresh_token": "refresh_token",
			"token_type": "bearer",
			"expires_in": 3600
		}`))
	}))

	fakeYahooConfig := &oauth2.Config{
		ClientID:     "fakeClientID",
		ClientSecret: "fakeClientSecret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/auth", fakeOAuthServer.URL),
			TokenURL: fmt.Sprintf("%s/token", fakeOAuthServer.URL),
		},
		RedirectURL: fmt.Sprintf("%s/redirect", fakeOAuthServer.URL),
	}
	return &TestController{
		Clock:       db.Clock,
		YahooConfig: fakeYahooConfig,
		fakeSleeper: NewFakeSleeperServer(),
		fakeYahoo:   NewFakeYahooServer(),
		fakeOAuth:   fakeOAuthServer,
	}
}
