package main

import (
	"embed"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

//go:embed content/*
var notesFS embed.FS

// Link represents a link with label and URL
type Link struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

// Note represents a single note from YAML
type Note struct {
	Slug    string   `yaml:"slug"`
	Title   string   `yaml:"title"`
	Thesis  string   `yaml:"thesis"`
	Quote   string   `yaml:"quote"`
	Bullets []string `yaml:"bullets"`
	Example string   `yaml:"example"`
	Diagram string   `yaml:"diagram"`
	Links   []Link   `yaml:"links"`
	Tags    []string `yaml:"tags"`
	Theme   string   `yaml:"theme"`
}

// IndexData holds data for the index template
type IndexData struct {
	Notes []Note
}

// Sitemap represents the root sitemap element
type Sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []SitemapURL `xml:"url"`
}

// SitemapURL represents a URL in the sitemap
type SitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Read all notes
	notes, err := readNotes()
	if err != nil {
		return fmt.Errorf("reading notes: %w", err)
	}

	// Sort notes by slug for consistent ordering
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Slug < notes[j].Slug
	})

	// Clean and recreate output directory
	if err := os.RemoveAll("output"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing output directory: %w", err)
	}
	if err := os.MkdirAll("output", 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Parse templates
	indexTmpl, err := template.ParseFS(templatesFS, "templates/index.html", "templates/footer.html")
	if err != nil {
		return fmt.Errorf("parsing index template: %w", err)
	}

	noteTmpl, err := template.ParseFS(templatesFS, "templates/note.html", "templates/footer.html")
	if err != nil {
		return fmt.Errorf("parsing note template: %w", err)
	}

	// Generate index page
	if err := generateIndex(indexTmpl, notes); err != nil {
		return fmt.Errorf("generating index: %w", err)
	}

	// Generate individual note pages
	for _, note := range notes {
		if err := generateNotePage(noteTmpl, note); err != nil {
			return fmt.Errorf("generating note page for %s: %w", note.Slug, err)
		}
	}

	// Copy static files
	if err := copyStaticFiles(); err != nil {
		return fmt.Errorf("copying static files: %w", err)
	}

	// Generate sitemap
	if err := generateSitemap(notes); err != nil {
		return fmt.Errorf("generating sitemap: %w", err)
	}

	fmt.Printf("✓ Generated %d note pages\n", len(notes))
	fmt.Println("✓ Generated index page")
	fmt.Println("✓ Copied static files")
	fmt.Println("✓ Generated sitemap.xml")
	fmt.Println("\nBuild complete! Output is in the 'output' directory.")

	return nil
}

func readNotes() ([]Note, error) {
	var notes []Note

	entries, err := notesFS.ReadDir("content")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join("content", entry.Name())
		data, err := notesFS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}

		var note Note
		if err := yaml.Unmarshal(data, &note); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}

		// Set default theme if not specified
		if note.Theme == "" {
			note.Theme = "default"
		}

		notes = append(notes, note)
	}

	return notes, nil
}

func generateIndex(tmpl *template.Template, notes []Note) error {
	f, err := os.Create("output/index.html")
	if err != nil {
		return err
	}
	defer f.Close()

	data := IndexData{Notes: notes}
	return tmpl.Execute(f, data)
}

func generateNotePage(tmpl *template.Template, note Note) error {
	// Generate /slug.html
	htmlFile := filepath.Join("output", note.Slug+".html")
	if err := writeNoteHTML(tmpl, htmlFile, note); err != nil {
		return err
	}

	// Generate /slug/index.html
	slugDir := filepath.Join("output", note.Slug)
	if err := os.MkdirAll(slugDir, 0755); err != nil {
		return err
	}
	indexFile := filepath.Join(slugDir, "index.html")
	return writeNoteHTML(tmpl, indexFile, note)
}

func writeNoteHTML(tmpl *template.Template, path string, note Note) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, note)
}

func copyStaticFiles() error {
	entries, err := staticFS.ReadDir("static")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		src, err := staticFS.Open(filepath.Join("static", entry.Name()))
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(filepath.Join("output", entry.Name()))
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
	}

	return nil
}

func generateSitemap(notes []Note) error {
	f, err := os.Create("output/sitemap.xml")
	if err != nil {
		return err
	}
	defer f.Close()

	// Get base URL from environment variable with fallback
	baseURL := os.Getenv("BASEURL")
	if baseURL == "" {
		return fmt.Errorf("BASEURL environment variable must be set")
	}
	lastMod := time.Now().Format("2006-01-02")

	// Build sitemap structure
	sitemap := Sitemap{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  make([]SitemapURL, 0, len(notes)+1),
	}

	// Add homepage
	sitemap.URLs = append(sitemap.URLs, SitemapURL{
		Loc:        baseURL + "/",
		LastMod:    lastMod,
		ChangeFreq: "weekly",
		Priority:   "1.0",
	})

	// Add individual notes
	for _, note := range notes {
		sitemap.URLs = append(sitemap.URLs, SitemapURL{
			Loc:        fmt.Sprintf("%s/%s/", baseURL, note.Slug),
			LastMod:    lastMod,
			ChangeFreq: "monthly",
			Priority:   "0.8",
		})
	}

	// Write XML with proper encoding
	encoder := xml.NewEncoder(f)
	encoder.Indent("", "  ")

	// Write XML header
	if _, err := f.WriteString(xml.Header); err != nil {
		return err
	}

	// Encode sitemap
	return encoder.Encode(sitemap)
}
