package kdb

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rizesql/kerberos/internal/o11y/logging"
)

type Config struct {
	DSN    string
	Logger *logging.Logger
}

type database struct {
	*sql.DB
	logger *logging.Logger
}

var _ DBTX = (*database)(nil)

func New(cfg Config) (*database, error) {
	db, err := open(cfg.DSN, *cfg.Logger)
	if err != nil {
		return nil, err
	}

	return &database{DB: db, logger: cfg.Logger}, nil
}

func open(dsn string, log logging.Logger) (db *sql.DB, err error) {
	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Warn("failed to ping database on startup",
			"error", err.Error())
	} else {
		log.Info("database connection pool initialized successfully")
	}

	return db, nil
}

func (d *database) Migrate() error {
	if _, err := d.Exec(SchemaSQL()); err != nil {
		return err
	}
	d.logger.Info("database schema applied successfully")
	return nil
}
