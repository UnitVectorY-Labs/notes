package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestContentFilesStructure validates that all YAML files in the content directory
// adhere to the required structure and constraints
func TestContentFilesStructure(t *testing.T) {
	// Read all files in the content directory
	entries, err := os.ReadDir("content")
	if err != nil {
		t.Fatalf("Failed to read content directory: %v", err)
	}

	// Filter to only YAML files
	var yamlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			yamlFiles = append(yamlFiles, entry.Name())
		}
	}

	if len(yamlFiles) == 0 {
		t.Fatal("No YAML files found in content directory")
	}

	// Run parametric test for each YAML file
	for _, filename := range yamlFiles {
		t.Run(filename, func(t *testing.T) {
			validateContentFile(t, filepath.Join("content", filename))
		})
	}
}

func validateContentFile(t *testing.T, path string) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Parse YAML
	var note Note
	if err := yaml.Unmarshal(data, &note); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Validate required fields
	if note.Slug == "" {
		t.Error("slug field is required but missing or empty")
	}

	if note.Title == "" {
		t.Error("title field is required but missing or empty")
	}

	if note.Thesis == "" {
		t.Error("thesis field is required but missing or empty")
	}

	if len(note.Bullets) == 0 {
		t.Error("bullets field is required but missing or empty")
	}

	if len(note.Tags) == 0 {
		t.Error("tags field is required but missing or empty")
	}

	// Validate slug format (should be lowercase with hyphens, no spaces or special chars)
	if note.Slug != "" {
		if strings.Contains(note.Slug, " ") {
			t.Error("slug should not contain spaces")
		}
		if strings.ToLower(note.Slug) != note.Slug {
			t.Error("slug should be lowercase")
		}
		// Check for only alphanumeric and hyphens
		for _, ch := range note.Slug {
			if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
				t.Errorf("slug contains invalid character '%c', should only contain lowercase letters, numbers, and hyphens", ch)
				break
			}
		}
	}

	// Validate bullets are non-empty strings
	for i, bullet := range note.Bullets {
		if strings.TrimSpace(bullet) == "" {
			t.Errorf("bullet at index %d is empty or whitespace only", i)
		}
	}

	// Validate tags are non-empty strings
	for i, tag := range note.Tags {
		if strings.TrimSpace(tag) == "" {
			t.Errorf("tag at index %d is empty or whitespace only", i)
		}
	}

	// Validate links (if present) have both label and url
	for i, link := range note.Links {
		if link.Label == "" {
			t.Errorf("link at index %d is missing label", i)
		}
		if link.URL == "" {
			t.Errorf("link at index %d is missing url", i)
		}
		// Basic URL validation - should start with http:// or https://
		if link.URL != "" && !strings.HasPrefix(link.URL, "http://") && !strings.HasPrefix(link.URL, "https://") {
			t.Errorf("link at index %d has invalid URL '%s', should start with http:// or https://", i, link.URL)
		}
	}

	// Validate theme (if present) is non-empty
	// Note: theme is optional (default is "default" as per main.go)
	if note.Theme != "" && strings.TrimSpace(note.Theme) == "" {
		t.Error("theme field should not be whitespace only if present")
	}

	// Validate that filename matches slug
	expectedFilename := note.Slug + ".yaml"
	actualFilename := filepath.Base(path)
	if expectedFilename != actualFilename {
		t.Errorf("filename '%s' does not match slug '%s' (expected '%s')", actualFilename, note.Slug, expectedFilename)
	}
}
