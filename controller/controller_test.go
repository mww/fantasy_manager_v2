package controller

import (
	"fmt"
	"os"
	"testing"

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
