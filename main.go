package main

import (
	"JavaDeezNuts/utils"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/russross/blackfriday/v2"
)

const (
	repoRoot      = "web"
	templateDir   = "templates"
	staticDir     = "static"
	logFilePath   = "access_logs.json" // Path to the log file
	ipAPIEndpoint = "http://ip-api.com/json/"
)

// Data is a struct to hold the data for rendering the template.
type Data struct {
	Title   string
	Content template.HTML
}

// AccessLog represents the logged data for each request.
type AccessLog struct {
	Time        string `json:"time"`
	IP          string `json:"ip"`
	Path        string `json:"path"`
	UserAgent   string `json:"user_agent"`
	GeoLocation string `json:"geo_location"`
}

// renderTemplate reads the template from the file and renders it with the given data.
func renderTemplate(w http.ResponseWriter, tmpl string, data Data, useCache bool) {
	tmplPath := filepath.Join(repoRoot, templateDir, tmpl)
	tmplContent, err := utils.FetchFileFromGitHub(tmplPath, true)
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
	content, err := utils.FetchFileFromGitHub(filepath.Join(repoRoot, staticDir, filePath), true)
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
	// Log the request data
	go logRequestData(r)

	// Get the requested path from the URL and convert it to the relative file path.
	requestedPath := strings.TrimPrefix(r.URL.Path, "/")
	filePath := filepath.Join(repoRoot, requestedPath)
	if requestedPath == "" {
		filePath = "web/java/Home.md"
	}
	// Fetch the content of the Markdown file from GitHub.
	mdContent, err := utils.FetchFileFromGitHub(filePath, false)
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

	cacheParam := r.URL.Query().Get("cache")
	useCache := cacheParam != "false"
	// Render the template with the data.
	go renderTemplate(w, "layout.html", data, useCache)
}

// logRequestData logs the relevant data of the incoming request.
func logRequestData(r *http.Request) {
	// Get the request time
	requestTime := time.Now().Format(time.ANSIC)

	// Get the user's IP address from the CF-Connecting-IP header
	ipAddress := r.Header.Get("CF-Connecting-IP")
	if ipAddress == "" {
		// If CF-Connecting-IP header is not available, fallback to RemoteAddr
		ipAddress, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	// Get the user's user agent
	userAgent := r.UserAgent()

	// Get the requested path
	requestedPath := strings.TrimPrefix(r.URL.Path, "/")

	// Get the user's geo-location from ip-api service
	geoLocation := getGeoLocation(ipAddress)

	// Prepare the log data
	logData := AccessLog{
		Time:        requestTime,
		IP:          ipAddress,
		Path:        requestedPath,
		UserAgent:   userAgent,
		GeoLocation: geoLocation,
	}

	// Log the data to console
	fmt.Printf("[%s] IP: %s Requested: %s, GeoLocation: %s\n",
		requestTime, ipAddress, requestedPath, geoLocation)

	// Log the data to a JSON file
	writeAccessLog(logData)
}

// getGeoLocation retrieves geo-location information from the ip-api service based on the given IP address.
func getGeoLocation(ipAddress string) string {
	response, err := http.Get(ipAPIEndpoint + ipAddress)
	if err != nil {
		log.Printf("Error fetching geo-location: %s", err)
		return "Unknown"
	}
	defer response.Body.Close()

	var geoInfo struct {
		Success string `json:"status"`
		City    string `json:"city"`
		Region  string `json:"regionName"`
		Country string `json:"country"`
	}

	err = json.NewDecoder(response.Body).Decode(&geoInfo)
	if err != nil {
		log.Printf("Error parsing geo-location data: %s", err)
		return "Unknown"
	}

	if strings.Compare(geoInfo.Success, "success") != 0 {
		return "UNABLE TO GET"
	}

	// Construct a string containing the relevant geo-location information
	geoLocation := fmt.Sprintf("%s, %s, %s", geoInfo.City, geoInfo.Region, geoInfo.Country)
	return geoLocation
}

// writeAccessLog writes the access log data to a JSON file.
func writeAccessLog(data AccessLog) {
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Error opening log file: %s", err)
		return
	}
	defer logFile.Close()

	logData, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		log.Printf("Error marshaling access log data: %s", err)
		return
	}

	logFile.Write(logData)
	logFile.Write([]byte("\n")) // Add a new line after each log entry
}

func main() {
	port := flag.Int("p", 8080, "Port number to run the server on")
	flag.Parse()

	http.HandleFunc("/", handler)
	http.HandleFunc("/static/", handleStaticFile)

	fmt.Printf("Starting server on http://localhost:%d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
