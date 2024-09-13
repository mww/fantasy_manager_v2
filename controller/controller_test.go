package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/mww/fantasy_manager_v2/platforms/sleeper"
	"github.com/mww/fantasy_manager_v2/platforms/yahoo"
	"github.com/mww/fantasy_manager_v2/testutils"
)

// A global testDB instance to use for all of the tests instead of setting up a new one each time.
var testDB *testutils.TestDB

// TestMain controls the main for the tests and allows for setup and shutdown of the tests
func TestMain(m *testing.M) {
	defer func() {
		// Catch all panics to make sure the shutdown is successfully run
		if r := recover(); r != nil {
			if testDB != nil {
				testDB.Shutdown()
			}
			fmt.Printf("panic - %v\n", r)
		}
	}()

	// Setup the global testDB variable
	testDB = testutils.NewTestDB()
	defer testDB.Shutdown()
	code := m.Run()
	os.Exit(code)
}

func controllerForTest() (C, *testutils.TestController) {
	tc := testutils.NewTestController(testDB)
	sleeper := sleeper.NewForTest(tc.SleeperURL())
	yahoo := yahoo.NewForTest(tc.YahooURL())
	ctrl, err := New(tc.Clock, testDB.DB, sleeper, yahoo, tc.YahooConfig)
	if err != nil {
		panic(fmt.Sprintf("error creating controller for test: %v", err))
	}
	return ctrl, tc
}

// Add some test coverage for nilPlatformAdapter to meet file level code coverage requirements
func TestNilPlatformAdapter(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("expected error")
	a := &nilPlatformAdapter{err: expectedErr}

	_, err := a.getLeagues("user", "2023")
	if !errors.Is(err, expectedErr) {
		t.Error("getLeagues did not return expected response")
	}

	_, err = a.getLeagueName(ctx, "", "")
	if !errors.Is(err, expectedErr) {
		t.Error("getLeagueName did not return expected response")
	}

	_, err = a.getManagers(ctx, nil)
	if !errors.Is(err, expectedErr) {
		t.Error("getManagers did not return expected response")
	}

	_, _, err = a.getMatchupResults(ctx, nil, 0)
	if !errors.Is(err, expectedErr) {
		t.Error("getMatchupResults did not return expected response")
	}

	_, err = a.getRosters(ctx, nil)
	if !errors.Is(err, expectedErr) {
		t.Error("getRosters did not return expected response")
	}

	_, err = a.getStarters(ctx, nil)
	if !errors.Is(err, expectedErr) {
		t.Error("getStarters did not return expected response")
	}
}
