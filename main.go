package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/itbasis/go-clock"
	"github.com/joho/godotenv"
	"github.com/mww/fantasy_manager_v2/controller"
	"github.com/mww/fantasy_manager_v2/db"
	"github.com/mww/fantasy_manager_v2/sleeper"
	"github.com/mww/fantasy_manager_v2/web"
)

func main() {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading .env file: %v", err)
	}
	connString := os.Getenv("POSTGRES_CONN_STR")

	portNum := 3000 // 3000 is the default
	port := os.Getenv("PORT")
	if port != "" {
		portNum, err = strconv.Atoi(port)
		if err != nil {
			log.Fatalf("error parsing port number: %v", err)
		}
	}

	clock := clock.New()
	db, err := db.New(context.Background(), connString, clock)
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

	server, err := web.NewServer(portNum, ctrl)
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