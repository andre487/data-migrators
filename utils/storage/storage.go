package storage

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path"

	"github.com/loynoir/ExpandUser.go"
)

type Storage struct {
	baseDir   string
	namespace string
}

func New(namespace string) *Storage {
	var baseDir string
	if envBaseDir := os.Getenv("DM_BASE_DIR"); envBaseDir != "" {
		baseDir = envBaseDir
	} else {
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome == "" {
			xdgDataHome = path.Join("~", ".local", "share")
		}
		baseDir = path.Join(xdgDataHome, "data-migrators487")
	}

	baseDir, err := ExpandUser.ExpandUser(baseDir)
	if err != nil {
		log.Fatalf("invalid base dir: %v", err)
	}

	return &Storage{baseDir: baseDir, namespace: namespace}
}

func (s *Storage) GetDir(name string, fileMode fs.FileMode) string {
	curDir := path.Join(s.baseDir, s.namespace, name)
	if err := os.MkdirAll(curDir, fileMode); err != nil {
		log.Fatalf("error when creating a directory: %v", err)
	}
	return curDir
}

func (s *Storage) GetFile(name string, fileMode fs.FileMode) string {
	curFile := path.Join(s.baseDir, s.namespace, name)
	if err := os.MkdirAll(path.Dir(curFile), 0755); err != nil {
		log.Fatalf("error when creating a directory: %v", err)
	}

	if _, err := os.Stat(curFile); errors.Is(err, os.ErrNotExist) {
		fp, err := os.OpenFile(curFile, os.O_RDONLY|os.O_CREATE, fileMode)
		if err != nil {
			log.Fatalf("error when creating storage file: %v", err)
		}
		if err := fp.Close(); err != nil {
			log.Printf("WARN: error when closing file: %v", err)
		}
	}
	return curFile
}
