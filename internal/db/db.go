package db

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/praveen001/uno/internal/config"
)

func NewConn(conf *config.Config) *sqlx.DB {
	str := fmt.Sprintf("postgresql://%v:%v@%v:%v/%v", conf.DB_USERNAME, conf.DB_PASSWORD, conf.DB_HOST, conf.DB_PORT, conf.DB_NAME)
	if conf.DISABLE_TLS == "true" {
		str = str + "?sslmode=disable"
	}
	slog.Info("Connecting to database")

	// Connect to database
	db, err := sqlx.Open("postgres", str)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalln("Unable to connect to database", err.Error())
	}

	slog.Info("Connected to database")

	return db
}
