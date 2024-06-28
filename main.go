package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/itbasis/go-clock"
	"github.com/mww/fantasy_manager_v2/controller"
	"github.com/mww/fantasy_manager_v2/db"
	"github.com/mww/fantasy_manager_v2/sleeper"
	"github.com/mww/fantasy_manager_v2/web"
)

func main() {
	clock := clock.New()
	db, err := db.New(context.Background(), "postgresql://ffuser:secret@localhost:5433/fantasy_manager", clock)
	if err != nil {
		log.Fatalf("cannot connect to DB: %v", err)
	}

	client, err := sleeper.New()
	if err != nil {
		log.Fatalf("error creating sleeper client: %v", err)
	}

	ctrl, err := controller.New(client, db)
	if err != nil {
		log.Fatalf("error creating a new controller: %v", err)
	}

	server, err := web.NewServer(3000, ctrl)
	if err != nil {
		log.Fatalf("error creating new web server: %v", err)
	}

	shutdown := make(chan bool)
	wg := &sync.WaitGroup{}

	// Setup a handler to catch ctrl-c signals and properly shutdown everything.
	intChannel := make(chan os.Signal, 2)
	signal.Notify(intChannel, os.Interrupt)
	go func() {
		<-intChannel
		close(shutdown)

		if err := waitTimeout(wg, 10*time.Second); err != nil {
			log.Printf("timed out waiting for proper shutdown")
			os.Exit(255)
		}
	}()

	// Setup a job that updates the players database from sleeper every 24-hours
	wg.Add(1)
	go ctrl.RunPeriodicPlayerUpdates(shutdown, wg)

	// Start the web server
	wg.Add(1)
	go server.ListenAndServe(shutdown, wg)

	// Wait for everything to stop.
	wg.Wait()
	log.Printf("server shutdown")
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) error {
	c := make(chan any)
	go func() {
		defer close(c)
		wg.Wait()
	}()

	select {
	case <-c:
		return nil // completed normally
	case <-time.After(timeout):
		return errors.New("timed out waiting")
	}
}
