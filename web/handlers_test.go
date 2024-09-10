package web

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/mww/fantasy_manager_v2/controller"
	"github.com/mww/fantasy_manager_v2/platforms/yahoo"
	"github.com/mww/fantasy_manager_v2/sleeper"
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

	if err := testutils.InsertTestPlayers(testDB.DB); err != nil {
		fmt.Printf("error inserting test players: %v", err)
	}

	code := m.Run()
	os.Exit(code)
}

func TestRankingsUploadHandler_success(t *testing.T) {
	testCtrl := testutils.NewTestController(testDB)
	defer testCtrl.Close()
	sleeperClient := sleeper.NewForTest(testCtrl.SleeperURL())
	yahooClient := yahoo.NewForTest(testCtrl.YahooURL())

	ctrl, err := controller.New(testCtrl.Clock, testDB.DB, sleeperClient, yahooClient, testCtrl.YahooConfig)
	if err != nil {
		t.Fatalf("error creating controller: %v", err)
	}

	resp := runRankingsUploadHandlerTest(t, ctrl, "text/csv", "2024-07-29")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("unexpected status code. Got: %d", resp.StatusCode)
	}

	if !regexp.MustCompile(`/players/rankings/\d+`).Match([]byte(resp.Header.Get("Location"))) {
		t.Errorf("redirect location not expected: %s", resp.Header.Get("Location"))
	}
}

func TestRankingsUploadHandler_badFileContentType(t *testing.T) {
	testCtrl := testutils.NewTestController(testDB)
	defer testCtrl.Close()
	sleeperClient := sleeper.NewForTest(testCtrl.SleeperURL())
	yahooClient := yahoo.NewForTest(testCtrl.YahooURL())

	ctrl, err := controller.New(testCtrl.Clock, testDB.DB, sleeperClient, yahooClient, testCtrl.YahooConfig)
	if err != nil {
		t.Fatalf("error creating controller: %v", err)
	}

	resp := runRankingsUploadHandlerTest(t, ctrl, "application/json", "2024-07-29")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unexpected status code. Got: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error response body: %v", err)
	}

	if !strings.Contains(string(b), "Only CSV files are supported. Got application/json") {
		t.Errorf("response body does not contain expected string")
	}
}

func TestRankingsUploadHandler_badDate(t *testing.T) {
	testCtrl := testutils.NewTestController(testDB)
	defer testCtrl.Close()
	sleeperClient := sleeper.NewForTest(testCtrl.SleeperURL())
	yahooClient := yahoo.NewForTest(testCtrl.YahooURL())

	ctrl, err := controller.New(testCtrl.Clock, testDB.DB, sleeperClient, yahooClient, testCtrl.YahooConfig)
	if err != nil {
		t.Fatalf("error creating controller: %v", err)
	}

	resp := runRankingsUploadHandlerTest(t, ctrl, "text/csv", "June 20th")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unexpected status code. Got: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error response body: %v", err)
	}

	if !strings.Contains(string(b), "Unable to parse rankings date. Expected format is YYYY-MM-DD:") {
		t.Errorf("response body does not contain expected string")
	}
}

func runRankingsUploadHandlerTest(t *testing.T, ctrl controller.C, contentType, date string) *http.Response {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	defer writer.Close()

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="rankings-file"; filename="file.csv"`)
	header.Set("Content-Type", contentType)

	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("error creating form filed 'rankings-file': %v", err)
	}
	part.Write([]byte(`"RK",TIERS,"PLAYER NAME",TEAM,"POS","BYE WEEK","SOS SEASON","ECR VS. ADP"`))
	part.Write([]byte("\n"))
	part.Write([]byte(`"1",1,"Justin Jefferson",MIN,"WR1","13","3 out of 5 stars","+1"`))
	part.Write([]byte("\n"))

	fieldWriter, err := writer.CreateFormField("rankings-date")
	if err != nil {
		t.Fatalf("error creating form field 'rankings-date': %v", err)
	}
	fieldWriter.Write([]byte(date))
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, "/", &buf)
	if err != nil {
		t.Fatalf("error creating http request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(rankingsUploadHandler(ctrl, newRender()))
	handler.ServeHTTP(rr, req)
	return rr.Result()
}
