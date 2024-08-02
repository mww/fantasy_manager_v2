package web

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/mww/fantasy_manager_v2/controller"
	"github.com/mww/fantasy_manager_v2/controller/mockcontroller"
	"github.com/stretchr/testify/mock"
)

func TestRankingsUploadHandler_success(t *testing.T) {
	mockCtrl := &mockcontroller.C{}

	d, _ := time.Parse(time.DateOnly, "2024-07-29")
	mockCtrl.On("AddRanking", mock.Anything, mock.Anything, d).Return(123, nil)

	resp := runRankingsUploadHandlerTest(t, mockCtrl, "text/csv", "2024-07-29")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("unexpected status code. Got: %d", resp.StatusCode)
	}
	if resp.Header.Get("Location") != "/players/rankings/123" {
		t.Errorf("redirect location not expected: %s", resp.Header.Get("Location"))
	}

	mockCtrl.AssertExpectations(t)
}

func TestRankingsUploadHandler_badFileContentType(t *testing.T) {
	mockCtrl := &mockcontroller.C{}

	resp := runRankingsUploadHandlerTest(t, mockCtrl, "application/json", "2024-07-29")
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

	mockCtrl.AssertNotCalled(t, "AddRanking", mock.Anything, mock.Anything, mock.Anything)
}

func TestRankingsUploadHandler_badDate(t *testing.T) {
	mockCtrl := &mockcontroller.C{}

	resp := runRankingsUploadHandlerTest(t, mockCtrl, "text/csv", "June 20th")
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

	mockCtrl.AssertNotCalled(t, "AddRanking", mock.Anything, mock.Anything, mock.Anything)
}

func TestRankingsUploadHandler_errorCallingAddRankings(t *testing.T) {
	mockCtrl := &mockcontroller.C{}

	d, _ := time.Parse(time.DateOnly, "2024-02-14")
	mockCtrl.On("AddRanking", mock.Anything, mock.Anything, d).Return(0, errors.New("Error with AddRankings"))

	resp := runRankingsUploadHandlerTest(t, mockCtrl, "text/csv", "2024-02-14")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("unexpected status code. Got: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error response body: %v", err)
	}

	if !strings.Contains(string(b), "Error with AddRanking") {
		t.Errorf("response body does not contain expected string")
	}

	mockCtrl.AssertExpectations(t)
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
	part.Write([]byte(`"RK",TIERS,"PLAYER NAME",TEAM,"POS","BYE WEEK","SOS SEASON","ECR VS. ADP"\n`))
	part.Write([]byte(`"1",1,"Justin Jefferson",MIN,"WR1","13","3 out of 5 stars","+1"\n`))

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
