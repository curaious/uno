package migrations

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"text/template"
	"time"

	"github.com/curaious/uno/internal/config"
	"github.com/curaious/uno/internal/db"

	"github.com/jmoiron/sqlx"
)

// migration ..
type migration struct {
	version string
	done    bool
	up      func(*sqlx.Tx) error
	down    func(*sqlx.Tx) error
}

// Migrator ..
type Migrator struct {
	db         *sqlx.DB
	versions   []string
	migrations map[string]*migration
}

var m = &Migrator{
	versions:   []string{},
	migrations: map[string]*migration{},
}

// NewMigrator ..
func NewMigrator() (*Migrator, error) {
	conf := config.ReadConfig()

	// Get the database instance
	m.db = db.NewConn(conf)

	_, err := m.db.Exec(`CREATE SCHEMA IF NOT EXISTS metadata`)
	if err != nil {
		slog.Error("Unable to create metadata schema", slog.Any("error", err))
		return nil, err
	}

	_, err = m.db.Exec(`CREATE TABLE IF NOT EXISTS metadata.schema_migrations (
		version varchar(255)
	);`)
	if err != nil {
		slog.Error("Unable to create `schema_migrations` table", slog.Any("error", err))
		return nil, err
	}

	rows, err := m.db.Query("SELECT version FROM metadata.schema_migrations;")
	if err != nil {
		slog.Error("Unable to fetch completed migrations", slog.Any("error", err))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		err := rows.Scan(&version)
		if err != nil {
			slog.Error("Unable to read row", slog.Any("error", err))
			return nil, err
		}

		if m.migrations[version] != nil {
			m.migrations[version].done = true
		}
	}

	return m, nil
}

// addMigration ..
func (m *Migrator) addMigration(mg *migration) {
	m.migrations[mg.version] = mg

	index := 0

	for index < len(m.versions) {
		if m.versions[index] > mg.version {
			break
		}

		index++
	}

	m.versions = append(m.versions, mg.version)
	copy(m.versions[index+1:], m.versions[index:])
	m.versions[index] = mg.version
}

// MigrationStatus ..
func (m *Migrator) MigrationStatus() error {
	for _, v := range m.versions {
		mg := m.migrations[v]

		if mg.done {
			slog.Info(fmt.Sprintf("Migration %s... completed", v))
		} else {
			slog.Info(fmt.Sprintf("Migration %s... pending", v))
		}
	}

	return nil
}

// CreateMigration ..
func (m *Migrator) CreateMigration(title string) error {
	var out bytes.Buffer

	version := time.Now().Format("20060102030405")

	in := struct {
		Version string
		Title   string
	}{
		Version: version,
		Title:   title,
	}

	t := template.Must(template.ParseFiles("./internal/migrations/template.txt"))
	err := t.Execute(&out, in)
	if err != nil {
		slog.Error("Unable to execute migration template", slog.Any("error", err))
		return err
	}

	f, err := os.Create(fmt.Sprintf("./internal/migrations/%s_%s.go", version, title))
	defer f.Close()
	if err != nil {
		slog.Error("Unable to create the migration file", slog.Any("error", err))
		return err
	}

	if _, err := f.WriteString(out.String()); err != nil {
		slog.Error("Unable to write to the migration file", slog.Any("error", err))
		return err
	}

	slog.Info("Generated new migration file...", slog.String("filename", f.Name()))
	return nil
}

// Up ..
func (m *Migrator) Up(step int) error {
	tx, err := m.db.BeginTxx(context.TODO(), &sql.TxOptions{})
	if err != nil {
		slog.Info("Unable to start transaction to run migrations", slog.Any("error", err))
		return err
	}

	defer func() {
		if err := recover(); err != nil {
			slog.Info("panic", slog.Any("details", err))
			tx.Rollback()
		}
	}()

	count := 0
	for _, v := range m.versions {
		if step > 0 && count == step {
			break
		}

		mg := m.migrations[v]
		l := slog.With(slog.String("version", mg.version))

		if mg.done {
			continue
		}

		l.Info("Running up migration...")
		if err := mg.up(tx); err != nil {
			tx.Rollback()
			l.Info("Error occured while running migration", slog.Any("error", err))
			return err
		}

		if _, err := tx.Exec("INSERT INTO metadata.schema_migrations VALUES($1);", mg.version); err != nil {
			tx.Rollback()
			l.Error("Failed to insert completed migrations to `metadata.schema_migrations`", slog.Any("error", err))
			return err
		}

		count++
		l.Info("Finished up migration...")
	}

	tx.Commit()

	return nil
}

// Down ..
func (m *Migrator) Down(step int) error {
	tx, err := m.db.BeginTxx(context.TODO(), &sql.TxOptions{})
	if err != nil {
		slog.Info("Unable to start transaction to run migrations", slog.Any("error", err))
		return err
	}

	defer func() {
		if err := recover(); err != nil {
			slog.Info("panic", slog.Any("details", err))
			tx.Rollback()
		}
	}()

	count := 0
	for _, v := range reverse(m.versions) {
		if step > 0 && count == step {
			break
		}

		mg := m.migrations[v]
		l := slog.With(slog.String("version", mg.version))

		if !mg.done {
			continue
		}

		l.Info("Running down migration...")
		if err := mg.down(tx); err != nil {
			tx.Rollback()
			l.Info("Error occured while running migration", slog.Any("error", err))
			return err
		}

		if _, err := tx.Exec("DELETE FROM metadata.schema_migrations WHERE version = $1;", mg.version); err != nil {
			tx.Rollback()
			l.Info("Failed to remove reverted migrations from `metadata.schema_migrations`", slog.Any("error", err))
			return err
		}

		count++
		l.Info("Finished down migration...")
	}

	tx.Commit()

	return nil
}

func reverse(arr []string) []string {
	for i := 0; i < len(arr)/2; i++ {
		j := len(arr) - i - 1
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}
