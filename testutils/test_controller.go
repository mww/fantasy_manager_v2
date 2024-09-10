package testutils

import (
	"github.com/itbasis/go-clock"
	"golang.org/x/oauth2"
)

type TestController struct {
	Clock       clock.Clock
	YahooConfig *oauth2.Config
	fakeSleeper *FakeSleeperServer
	fakeYahoo   *FakeYahooServer
}

func (c *TestController) Close() {
	c.fakeSleeper.Close()
	c.fakeYahoo.Close()
}

func (c *TestController) SleeperURL() string {
	return c.fakeSleeper.URL()
}

func (c *TestController) YahooURL() string {
	return c.fakeYahoo.URL()
}

func NewTestController(db *TestDB) *TestController {
	return &TestController{
		Clock:       db.Clock,
		YahooConfig: nil,
		fakeSleeper: NewFakeSleeperServer(),
		fakeYahoo:   NewFakeYahooServer(),
	}
}
