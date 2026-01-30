package disk_storage

import (
	"archive/zip"
	"bytes"
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

// getAgentDirName returns the directory name for version 0 of an agent
func getAgentDirName(agentName string) string {
	return strings.ToLower(fmt.Sprintf("%s-0", agentName))
}

// UploadSkillToTemp extracts a skill zip file to the agent's temp directory
func (s *DiskStorage) UploadSkillToTemp(agentName string, zipData []byte, zipFileName string) (*agent_config.TempSkillUploadResponse, error) {
	agentDirName := getAgentDirName(agentName)
	tempDir := filepath.Join(s.Path, agentDirName, "temp")

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Extract zip file name (without extension) to use as directory name
	zipName := strings.TrimSuffix(zipFileName, filepath.Ext(zipFileName))
	extractDir := filepath.Join(tempDir, zipName)

	// Remove existing temp directory if it exists
	if err := os.RemoveAll(extractDir); err != nil {
		return nil, fmt.Errorf("failed to clean existing temp directory: %w", err)
	}

	// Create extract directory
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create extract directory: %w", err)
	}

	// Open zip reader
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}

	// Extract all files from zip
	for _, zipFile := range zipReader.File {
		if err := extractZipFile(zipFile, extractDir); err != nil {
			os.RemoveAll(extractDir) // Cleanup on error
			return nil, err
		}
	}

	// Parse SKILL.md to extract name and description
	skillMDPath := filepath.Join(extractDir, "SKILL.md")
	skillName, skillDescription, err := parseSkillMD(skillMDPath)
	if err != nil {
		os.RemoveAll(extractDir) // Cleanup on error
		return nil, fmt.Errorf("failed to parse SKILL.md: %w", err)
	}

	// Build the relative temp path for response
	relTempPath := filepath.Join(agentDirName, "temp", zipName)

	return &agent_config.TempSkillUploadResponse{
		Name:        skillName,
		Description: skillDescription,
		TempPath:    relTempPath,
		SkillFolder: zipName,
	}, nil
}

// extractZipFile extracts a single file from the zip archive
func extractZipFile(zipFile *zip.File, extractDir string) error {
	// Sanitize file path to prevent directory traversal
	cleanName := filepath.Clean(zipFile.Name)
	if strings.HasPrefix(cleanName, "..") || strings.Contains(cleanName, "..") {
		return fmt.Errorf("invalid file path in zip: %s", zipFile.Name)
	}

	filePath := filepath.Join(extractDir, cleanName)

	// Ensure the resolved path is still within extractDir
	absExtractDir, err := filepath.Abs(extractDir)
	if err != nil {
		return fmt.Errorf("failed to resolve extract directory: %w", err)
	}
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve file path: %w", err)
	}
	if !strings.HasPrefix(absFilePath, absExtractDir) {
		return fmt.Errorf("invalid file path in zip: %s", zipFile.Name)
	}

	// Create directory if needed
	if zipFile.FileInfo().IsDir() {
		return os.MkdirAll(filePath, zipFile.FileInfo().Mode())
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", zipFile.Name, err)
	}

	// Open file from zip
	rc, err := zipFile.Open()
	if err != nil {
		return fmt.Errorf("failed to open file from zip %s: %w", zipFile.Name, err)
	}
	defer rc.Close()

	// Create destination file
	outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipFile.FileInfo().Mode())
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", zipFile.Name, err)
	}
	defer outFile.Close()

	// Copy file content
	if _, err = io.Copy(outFile, rc); err != nil {
		return fmt.Errorf("failed to extract file %s: %w", zipFile.Name, err)
	}

	return nil
}

// parseSkillMD parses the SKILL.md file and extracts name and description from YAML frontmatter
func parseSkillMD(path string) (name string, description string, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("SKILL.md not found in zip")
	}

	contentStr := string(content)

	// Check if the file starts with YAML frontmatter delimiter
	if !strings.HasPrefix(contentStr, "---") {
		return "", "", fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
	}

	// Find the end of the frontmatter
	endIdx := strings.Index(contentStr[3:], "---")
	if endIdx == -1 {
		return "", "", fmt.Errorf("SKILL.md has invalid YAML frontmatter (missing closing ---)")
	}

	frontmatter := contentStr[3 : endIdx+3]

	// Parse YAML frontmatter line by line
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		} else if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}

	if name == "" {
		return "", "", fmt.Errorf("SKILL.md frontmatter must contain 'name' field")
	}
	if description == "" {
		return "", "", fmt.Errorf("SKILL.md frontmatter must contain 'description' field")
	}

	return name, description, nil
}

// DeleteTempSkill removes a skill from the agent's temp directory
func (s *DiskStorage) DeleteTempSkill(agentName string, skillFolder string) error {
	// Sanitize skill folder name to prevent directory traversal
	cleanSkillFolder := filepath.Clean(skillFolder)
	if strings.Contains(cleanSkillFolder, "..") || strings.ContainsAny(cleanSkillFolder, "/\\") {
		return fmt.Errorf("invalid skill folder name: %s", skillFolder)
	}

	agentDirName := getAgentDirName(agentName)
	tempSkillPath := filepath.Join(s.Path, agentDirName, "temp", cleanSkillFolder)

	if err := os.RemoveAll(tempSkillPath); err != nil {
		return fmt.Errorf("failed to delete temp skill: %w", err)
	}

	return nil
}

// CommitSkill moves a skill from temp to the permanent skills directory
func (s *DiskStorage) CommitSkill(agentName string, skillFolder string) (string, error) {
	// Sanitize skill folder name to prevent directory traversal
	cleanSkillFolder := filepath.Clean(skillFolder)
	if strings.Contains(cleanSkillFolder, "..") || strings.ContainsAny(cleanSkillFolder, "/\\") {
		return "", fmt.Errorf("invalid skill folder name: %s", skillFolder)
	}

	agentDirName := getAgentDirName(agentName)
	tempSkillPath := filepath.Join(s.Path, agentDirName, "temp", cleanSkillFolder)
	skillsDir := filepath.Join(s.Path, agentDirName, "skills")
	destSkillPath := filepath.Join(skillsDir, cleanSkillFolder)

	// Check if temp skill exists
	if _, err := os.Stat(tempSkillPath); os.IsNotExist(err) {
		return "", fmt.Errorf("temp skill not found: %s", skillFolder)
	}

	// Create skills directory if it doesn't exist
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Remove existing skill directory if it exists
	if err := os.RemoveAll(destSkillPath); err != nil {
		return "", fmt.Errorf("failed to clean existing skill directory: %w", err)
	}

	// Move from temp to skills directory
	if err := os.Rename(tempSkillPath, destSkillPath); err != nil {
		return "", fmt.Errorf("failed to move skill from temp: %w", err)
	}

	// Build the relative file location for SKILL.md
	fileLocation := filepath.Join(agentDirName, "skills", cleanSkillFolder, "SKILL.md")

	return fileLocation, nil
}

// DeleteSavedSkill removes a committed skill from the agent's skills directory
func (s *DiskStorage) DeleteSavedSkill(agentName string, skillFolder string) error {
	// Sanitize skill folder name to prevent directory traversal
	cleanSkillFolder := filepath.Clean(skillFolder)
	if strings.Contains(cleanSkillFolder, "..") || strings.ContainsAny(cleanSkillFolder, "/\\") {
		return fmt.Errorf("invalid skill folder name: %s", skillFolder)
	}

	agentDirName := getAgentDirName(agentName)
	skillPath := filepath.Join(s.Path, agentDirName, "skills", cleanSkillFolder)

	if err := os.RemoveAll(skillPath); err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	return nil
}
