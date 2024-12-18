package web

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/mww/fantasy_manager_v2/controller"
	"github.com/unrolled/render"
)

//go:embed templates
var templates embed.FS

type Server struct {
	server *http.Server
}

func NewServer(port int, ctrl controller.C, version, githash, buildDate string) (*Server, error) {
	render := newRender(version, githash, buildDate)
	router := getRouter(ctrl, render)

	s := &Server{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: router,
		},
	}
	return s, nil
}

func (s *Server) ListenAndServe(shutdown chan bool, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()

		// Wait for the shutdown signal and safely close the server.
		<-shutdown

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			log.Fatalf("fatal error shutting down server: %v", err)
		}
	}()

	log.Printf("web server is listening on %s", s.server.Addr)
	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("fatal error with server: %v", err)
	}
}

func newRender(version, githash, buildDate string) *render.Render {
	return render.New(render.Options{
		Directory: "templates",
		Layout:    "layout",
		FileSystem: &render.EmbedFileSystem{
			FS: templates,
		},
		Funcs: []template.FuncMap{
			{
				"age":       ageFormatter,
				"date":      dateFormatter,
				"height":    heightFormatter,
				"year":      yearFormatter,
				"score":     scoreFormatter,
				"dateTime":  dateTimeFormatter,
				"version":   stringOrUnset(version),
				"gitHash":   stringOrUnset(githash),
				"buildDate": stringOrUnset(buildDate),
			},
		},
	})
}

func ageFormatter(t time.Time) string {
	return ageFormatterInternal(t, time.Now())
}

func ageFormatterInternal(t, n time.Time) string {
	if n.Before(t) {
		return "error calculating negative age"
	}

	years := n.Year() - t.Year()
	days := n.YearDay() - t.YearDay()
	if days < 0 {
		days += 365
		years--
	}
	return fmt.Sprintf("%s (%d years, %d days)", t.Format("Jan 2, 2006"), years, days)
}

func dateFormatter(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02")
}

func heightFormatter(inches int) string {
	ft := inches / 12
	in := inches % 12
	return fmt.Sprintf("%d'%d\"", ft, in)
}

func yearFormatter(t time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}
	return t.Format("2006")
}

func dateTimeFormatter(t time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}
	return t.Format(time.DateTime)
}

func scoreFormatter(s int32) string {
	d := float64(s) / 1000
	return fmt.Sprintf("%0.2f", d)
}

// Returns a function that either returns the value s or "unset"
func stringOrUnset(s string) func() string {
	if s == "" {
		s = "unset"
	}

	return func() string {
		return s
	}
}
