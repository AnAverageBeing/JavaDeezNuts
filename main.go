package main

import (
	"JavaDeezNuts/utils"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/russross/blackfriday/v2"
)

const (
	repoRoot    = "web"
	templateDir = "templates"
	staticDir   = "static"
)

// Data is a struct to hold the data for rendering the template.
type Data struct {
	Title   string
	Content template.HTML
}

// renderTemplate reads the template from the file and renders it with the given data.
func renderTemplate(w http.ResponseWriter, tmpl string, data Data) {
	tmplPath := filepath.Join(repoRoot, templateDir, tmpl)
	tmplContent, err := utils.FetchFileFromGitHub(tmplPath)
	if err != nil {
		http.Error(w, "Error fetching template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmplParsed, err := template.New("").Parse(tmplContent)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmplParsed.Execute(w, data)
	if err != nil {
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// markdownToHTML converts Markdown content to HTML.
func markdownToHTML(mdContent string) template.HTML {
	htmlContent := blackfriday.Run([]byte(mdContent))
	return template.HTML(htmlContent)
}

// handleStaticFile fetches and serves the static files from the GitHub repository.
func handleStaticFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path[len("/static/"):]
	content, err := utils.FetchFileFromGitHub(filepath.Join(repoRoot, staticDir, filePath))
	if err != nil {
		http.Error(w, "Error fetching static file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", http.DetectContentType([]byte(content)))

	if strings.HasSuffix(filePath, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}
	w.Write([]byte(content))
}

// handler retrieves the Markdown file from the GitHub repo, converts it to HTML, and serves it using the layout template.
func handler(w http.ResponseWriter, r *http.Request) {
	// Get the requested path from the URL and convert it to the relative file path.
	requestedPath := strings.TrimPrefix(r.URL.Path, "/")
	filePath := filepath.Join(repoRoot, requestedPath)
	if requestedPath == "" {
		filePath = "web/java/Home.md"
	}
	// Fetch the content of the Markdown file from GitHub.
	mdContent, err := utils.FetchFileFromGitHub(filePath)
	if err != nil {
		http.Error(w, "Error fetching file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert the Markdown content to HTML.
	htmlContent := markdownToHTML(mdContent)

	// Get the title of the page from the first line of the Markdown file.
	lines := strings.Split(mdContent, "\n")
	title := lines[0]

	// Create the data for rendering the template.
	data := Data{
		Title:   title,
		Content: htmlContent,
	}

	// Render the template with the data.
	renderTemplate(w, "layout.html", data)
}

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/static/", handleStaticFile)

	fmt.Println("Starting server on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
