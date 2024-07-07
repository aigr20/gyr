package gyr

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type MigratorSettings struct {
	Directory string
	Context   context.Context
	LogWriter *os.File
}

func DefaultMigratorSettings() MigratorSettings {
	return MigratorSettings{
		Context:   context.Background(),
		Directory: "migrations",
		LogWriter: os.Stdout,
	}
}

func MigrationDirectory(dir string) func(*MigratorSettings) {
	return func(ms *MigratorSettings) {
		ms.Directory = dir
	}
}

func MigrationContext(context context.Context) func(*MigratorSettings) {
	return func(ms *MigratorSettings) {
		ms.Context = context
	}
}

func MigrationLogOutput(file *os.File) func(*MigratorSettings) {
	return func(ms *MigratorSettings) {
		ms.LogWriter = file
	}
}

type Migrator struct {
	connection  *sql.DB
	version     string
	path        string
	logger      *slog.Logger
	LastVersion string
	Settings    MigratorSettings
}

func NewMigrator(connection *sql.DB, settings ...SettingsFunc[MigratorSettings]) *Migrator {
	migratorSettings := DefaultMigratorSettings()
	for _, setting := range settings {
		setting(&migratorSettings)
	}

	logLevel := slog.LevelInfo
	if isGyrDebug() {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(migratorSettings.LogWriter, &slog.HandlerOptions{Level: logLevel}))

	logger.Info("Initializing Gyr Database Migrator", "directory", migratorSettings.Directory)
	return &Migrator{
		connection: connection,
		logger:     logger,
		Settings:   migratorSettings,
	}
}

func (mig *Migrator) Migrate() error {
	err := mig.createMigrationTable()
	if err != nil {
		return err
	}
	err = mig.getMigrationVersion()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	transaction, err := mig.connection.BeginTx(mig.Settings.Context, nil)
	if err != nil {
		return err
	}
	defer mig.rollbackTransaction(transaction)
	err = mig.executeMigrations(transaction)
	if err != nil {
		mig.logger.Error("Error in migration execution", "error", err)
		return err
	}

	err = mig.setMigrationVersion()
	if err != nil {
		return err
	}
	return transaction.Commit()
}

func (mig *Migrator) createMigrationTable() error {
	mig.logger.Debug("Creating gyr_migrator_version_history table")
	const query = "create table if not exists gyr_migrator_version_history (version varchar(10), path varchar(255));"
	_, err := mig.connection.ExecContext(mig.Settings.Context, query)
	return err
}

func (mig *Migrator) getMigrationVersion() error {
	const query = "select version from gyr_migrator_version_history order by version desc"
	row := mig.connection.QueryRowContext(mig.Settings.Context, query)
	err := row.Scan(&mig.LastVersion)

	mig.logger.Info("Detected migration version", "version", mig.LastVersion)
	return err
}

func (mig *Migrator) setMigrationVersion() error {
	if mig.path == "" || mig.version == "" {
		return nil
	}
	const query = "insert into gyr_migrator_version_history (version, path) values (?, ?)"
	_, err := mig.connection.ExecContext(mig.Settings.Context, query, mig.version, mig.path)
	if err != nil {
		return err
	}
	mig.LastVersion = mig.version
	mig.logger.Info("Migrated to version", "version", mig.LastVersion)
	return nil
}

func (mig *Migrator) executeMigrations(transaction *sql.Tx) error {
	paths := getSqlFilenames(mig.Settings.Directory)
	paths = removeAlreadyMigratedPaths(paths, mig.LastVersion)
	mig.logger.Info("Running migrations", "migrations", len(paths))

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			mig.logger.Warn("Failed to open a file", "path", path, "error", err)
			continue
		}
		defer file.Close()

		mig.logger.Info("Running SQL script", "file", path)

		fileReader := bufio.NewReader(file)
		var query string
		var readErr error = nil
		for !errors.Is(readErr, io.EOF) {
			query, readErr = fileReader.ReadString(';')
			if readErr == nil {
				query = strings.TrimSpace(query)
				mig.logger.Debug("Executing query", "query", query)
				_, err = transaction.ExecContext(mig.Settings.Context, query)
				if err != nil {
					return err
				}
			}
		}

		mig.path = path
		mig.version = migrationVersionFromFilepath(path)
	}
	return nil
}

func removeAlreadyMigratedPaths(paths []string, mostRecentVersion string) []string {
	return slices.DeleteFunc(paths, func(path string) bool {
		return strings.Compare(migrationVersionFromFilepath(path), mostRecentVersion) <= 0
	})
}

func getSqlFilenames(directory string) []string {
	sqlFiles := make([]string, 0)
	filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".sql") {
			sqlFiles = append(sqlFiles, path)
		}
		return nil
	})
	slices.SortFunc(sqlFiles, func(a, b string) int {
		fileNameA := a[strings.LastIndex(a, "/")+1:]
		fileNameB := b[strings.LastIndex(b, "/")+1:]
		return strings.Compare(fileNameA, fileNameB)
	})

	return sqlFiles
}

func (mig *Migrator) rollbackTransaction(transaction *sql.Tx) {
	if err := transaction.Rollback(); !errors.Is(err, sql.ErrTxDone) && err != nil {
		mig.logger.Error("Transaction rollback failed", "error", err)
	}
}

func migrationVersionFromFilepath(path string) string {
	filename := path[strings.LastIndex(path, "/")+1:]
	return strings.Split(filename, "_")[0]
}
