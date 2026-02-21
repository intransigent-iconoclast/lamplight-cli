package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// based on this rust idea https://crates.io/crates/directories ... can't bundle the db with the executable so we must create it in standard location
// this varies based on operating system.
func DefaultDataDir(appName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var base string
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			base = filepath.Join(appData, appName)
		} else {
			base = filepath.Join(home, appName)
		}
	case "darwin":
		base = filepath.Join(home, "Library", "Application Support", appName)
	default: // linux, *bsd, etc.
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			base = filepath.Join(xdg, appName)
		} else {
			base = filepath.Join(home, ".local", "share", appName)
		}
	}

	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}
	return base, nil
}

func DefaultDBPath(appName string) (string, error) {
	dir, err := DefaultDataDir(appName)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lamplight-cli.db.sqlite"), nil
}

func Open(appName string, dropSchema bool) (*gorm.DB, error) {
	dbPath, err := DefaultDBPath(appName)
	if err != nil {
		return nil, fmt.Errorf("resolve db path: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// wipes db which is no goooood if i'm releasing this
	if dropSchema {
		if err := db.Migrator().DropTable(&entity.Downloader{}); err != nil {
			return nil, fmt.Errorf("drop schema: %w", err)
		}
	}

	if err := db.AutoMigrate(&entity.Indexer{}); err != nil {
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	if err := db.AutoMigrate(&entity.Downloader{}); err != nil {
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	if err := db.AutoMigrate(&entity.SearchCache{}); err != nil {
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return db, nil
}
