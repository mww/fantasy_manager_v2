package containers

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	image      = "postgres:16.3-alpine"
	dbName     = "fantasy_manager"
	dbUser     = "ffuser"
	dbPassword = "secret"
)

type DBContainer struct {
	container *postgres.PostgresContainer
}

func NewDBContainer() *DBContainer {
	ctx := context.Background()

	container, err := postgres.Run(ctx, image,
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.WithInitScripts(filepath.Join("..", "schema", "schema.sql")),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		log.Fatalf("error starting container: %v", err)
	}

	return &DBContainer{
		container: container,
	}
}

func (c *DBContainer) Shutdown() {
	err := c.container.Terminate(context.Background())
	if err != nil {
		log.Fatalf("error terminating container: %v", err)
	}
}

func (c *DBContainer) ConnectionString() string {
	// explicitly set sslmode=disable because the container is not configured to use TLS
	connStr, err := c.container.ConnectionString(context.Background(), "sslmode=disable")
	if err != nil {
		log.Fatalf("error getting connection string: %v", err)
	}
	return connStr
}
