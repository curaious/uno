package disk_storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/curaious/uno/internal/services/agent_config"
)

type DiskStorage struct {
	Path string
}

func NewDiskStorage(path string) *DiskStorage {
	return &DiskStorage{
		Path: path,
	}
}

func (s *DiskStorage) CreateAgentDataDirectory(config *agent_config.AgentConfig) error {
	agentDirName := config.GetName()
	agentDir := filepath.Join(s.Path, agentDirName)

	// Create skills directory under workspace
	skillsDir := filepath.Join(agentDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	// If this is a new version (version > 0), copy contents from version 0
	if config.Version > 0 {
		// Use same naming as GetName(): "{name}-{version}" (e.g. "my-agent-0")
		version0DirName := strings.ToLower(fmt.Sprintf("%s-0", config.Name))
		version0Dir := filepath.Join(s.Path, version0DirName)

		// Copy skills directory from version 0
		version0SkillsDir := filepath.Join(version0Dir, "skills")
		if _, err := os.Stat(version0SkillsDir); err == nil {
			// Copy skills content from version 0 to new version
			if err := copyDirectory(version0SkillsDir, skillsDir); err != nil {
				return fmt.Errorf("failed to copy workspace from version 0: %w", err)
			}
		}
	}

	return nil
}

// copyDirectory recursively copies a directory from source to destination
func copyDirectory(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory with same permissions
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy file content
	_, err = io.Copy(dstFile, srcFile)
	return err
}
