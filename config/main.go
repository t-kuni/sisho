// Package config provides configuration-related functionality for the project.
// The actual implementation is split between domain and infrastructure layers.
package config

import (
	"os"
	"path/filepath"
)

// This file is kept as a placeholder to maintain the package structure.
// For actual configuration handling, refer to:
// - domain/repository/config
// - infrastructure/repository/config

// ContextScan scans the directory structure from the target path up to the root directory
// and collects relevant files (README.md and [TARGET_CODE].md).
func ContextScan(rootDir string, targetPath string) ([]string, error) {
	var collectedFiles []string
	currentDir, err := filepath.Abs(filepath.Dir(targetPath))
	if err != nil {
		return nil, err
	}
	rootDir, err = filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	for {
		// Collect README.md
		readmePath := filepath.Join(currentDir, "README.md")
		if _, err := os.Stat(readmePath); err == nil {
			collectedFiles = append(collectedFiles, readmePath)
		}

		// Collect [TARGET_CODE].md
		targetCodeMdPath := filepath.Join(currentDir, filepath.Base(targetPath)+".md")
		if _, err := os.Stat(targetCodeMdPath); err == nil {
			collectedFiles = append(collectedFiles, targetCodeMdPath)
		}

		if currentDir == rootDir {
			break
		}
		currentDir = filepath.Dir(currentDir)
	}

	return collectedFiles, nil
}

// CollectAutoCollectFiles filters the files collected by ContextScan based on the configuration.
func CollectAutoCollectFiles(config Config, rootDir string, targetPath string) ([]string, error) {
	files, err := ContextScan(rootDir, targetPath)
	if err != nil {
		return nil, err
	}

	var collectedFiles []string
	for _, file := range files {
		baseName := filepath.Base(file)
		if config.AutoCollect.ReadmeMd && baseName == "README.md" {
			collectedFiles = append(collectedFiles, file)
		}
		if config.AutoCollect.TargetCodeMd && baseName == filepath.Base(targetPath)+".md" {
			collectedFiles = append(collectedFiles, file)
		}
	}

	return collectedFiles, nil
}
