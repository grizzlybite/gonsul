package exporter

import (
	"github.com/grizzlybite/gonsul/internal/entities"
	"github.com/grizzlybite/gonsul/internal/util"

	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// parseDir recursively traverses a repository directory and parses supported files.
func (e *exporter) parseDir(directory string, localData map[string]string) error {
	// Read the entire directory
	files, err := os.ReadDir(directory)
	if err != nil {
		return util.NewGonsulError(fmt.Errorf("read directory %s: %w", directory, err), util.ErrorFailedJsonDecode)
	}
	// Loop each entry
	for _, file := range files {
		if file.IsDir() {
			// We found a directory, recurse it
			newDir := directory + "/" + file.Name()
			if err := e.parseDir(newDir, localData); err != nil {
				return err
			}
		} else {
			filePath := directory + "/" + file.Name()
			ext := filepath.Ext(filePath)
			if !e.isExtensionValid(ext) {
				continue
			}
			content, err := os.ReadFile(filePath)
			if err != nil {
				return util.NewGonsulError(fmt.Errorf("read file %s: %w", filePath, err), util.ErrorFailedJsonDecode)
			}
			if err := e.parseFile(filePath, string(content), localData); err != nil {
				return err
			}
		}
	}

	return nil
}

// isExtensionValid checks if given file extensions is valid for processing
func (e *exporter) isExtensionValid(extension string) bool {
	for _, validExtension := range e.config.GetValidExtensions() {
		if strings.Trim(extension, ".") == strings.Trim(validExtension, ".") {
			return true
		}
	}

	return false
}

// parseFile ...
func (e *exporter) parseFile(filePath string, value string, localData map[string]string) error {
	// Extract our file extension and cleanup file path
	ext := filepath.Ext(filePath)
	cleanedPath := e.cleanFilePath(filePath)

	// Check if the file is JSON.
	if ext == ".json" {
		if e.config.ShouldExpandJSON() {
			if err := e.expandJSON(cleanedPath, value, localData); err != nil {
				return err
			}

			// Return here to avoid importing the original file as a blob.
			return nil
		}

		// Not expanding JSON, but the file still must be valid JSON.
		if _, err := e.validateJSON(cleanedPath, value); err != nil {
			return util.NewGonsulError(err, util.ErrorFailedJsonDecode)
		}
	}

	// Check if the file is YAML.
	if isYAMLExtension(ext) {
		if e.config.ShouldExpandYAML() {
			if err := e.expandYAML(cleanedPath, value, localData); err != nil {
				return err
			}

			// Return here to avoid importing the original file as a blob.
			return nil
		}

		// Not expanding YAML, but the file still must be valid YAML.
		if _, err := e.validateYAML(cleanedPath, value); err != nil {
			return util.NewGonsulError(err, util.ErrorFailedJsonDecode)
		}
	}

	// Store the whole file content as a single value.
	piece := e.createPiece(cleanedPath, value)
	localData[piece.KVPath] = piece.Value

	return nil
}

func isYAMLExtension(extension string) bool {
	return extension == ".yaml" || extension == ".yml"
}

// cleanFilePath ...
func (e *exporter) cleanFilePath(filePath string) string {
	// Set part of the config that should be removed from the current
	// file system path in order to build our final Consul KV path
	replace := path.Join(e.config.GetRepoRootDir(), e.config.GetRepoBasePath())
	// Remove the above from the file system path
	entryFilePath := strings.Replace(filePath, replace, "", 1)
	// Remove any left slash
	entryFilePath = strings.Replace(entryFilePath, "/", "", 1)
	// Set or not the file extension when importing to consul k/v the file
	if !e.config.KeepFileExt() {
		entryFilePath = strings.TrimSuffix(entryFilePath, filepath.Ext(entryFilePath))
	}

	return entryFilePath
}

// createPiece ...
func (e *exporter) createPiece(piecePath string, value string) entities.Entry {
	// Create our Consul base path variable
	var kvPath string

	// Check if we have a Consul KV base path
	if e.config.GetConsulBasePath() != "" {
		kvPath = e.config.GetConsulBasePath()
	}

	// Finally append the Consul KV base path to the file path, if base is not an empty string
	if kvPath != "" {
		fullPath := path.Join(kvPath, piecePath)
		return entities.Entry{KVPath: fullPath, Value: value}
	}

	return entities.Entry{KVPath: piecePath, Value: value}
}
