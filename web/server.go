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

func NewServer(port int, ctrl controller.C) (*Server, error) {
	render := newRender()
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

func newRender() *render.Render {
	return render.New(render.Options{
		Directory: "templates",
		Layout:    "layout",
		FileSystem: &render.EmbedFileSystem{
			FS: templates,
		},
		Funcs: []template.FuncMap{
			{
				"age":    ageFormatter,
				"date":   dateFormatter,
				"height": heightFormatter,
				"year":   yearFormatter,
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
