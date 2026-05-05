package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type Config struct {
	Host        string
	Port        string
	User        string
	Password    string
	Database    string
	SSLMode     string
	Driver      string
	Environment string
}

func NewConnection(config *Config) *sql.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s application_name=Chatbot-%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode, config.Environment,
	)

	db, err := sql.Open(config.Driver, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

func runMigrations(conn *sql.DB) error {
	driver, err := postgres.WithInstance(conn, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create migration driver: %v", err)
		return err
	}

	pwd, _ := os.Getwd()
	// ATENÇÃO: Ajuste este caminho se suas migrations estiverem em outra pasta (ex: "sql/schema")
	migrationsPath := filepath.Join(pwd, "db/migration")
	migrationsPath = filepath.ToSlash(migrationsPath)

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres", driver)
	if err != nil {
		panic(err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal("Failed to run migrations:", err)
		return err
	}

	return nil
}
