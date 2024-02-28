package main

import (
	"embed"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

//go:embed templates/*
var content embed.FS

var templates *template.Template

// Img represents information about an image
type Img struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// Global variable to store information about all assets
var assets map[string][]Img
var mu sync.Mutex // Mutex to protect concurrent access to the assets map
var imageDir string
var port string

func init() {
	// Define a command-line flag for the image directory
	flag.StringVar(&imageDir, "dir", "./assets", "Directory containing images")
	flag.StringVar(&port, "port", "8080", "Port to listen on")
	flag.Parse()

	templateContent, err := content.ReadFile("templates/home.html")

	if err != nil {
		panic(err)
	}

	templates = template.Must(template.New("home").Parse(string(templateContent)))

	assets = make(map[string][]Img)

	// Walk through the image directory to get a list of files
	err = filepath.Walk(imageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the file is a regular file (not a directory)
		if !info.IsDir() {
			// Get the relative path of the file (relative to the image directory)
			relPath, err := filepath.Rel(imageDir, path)
			if err != nil {
				return err
			}

			// Create an Img object for the image
			img := Img{
				Path: "/assets/" + relPath,
				Name: info.Name(),
			}

			// Lock the mutex before updating the assets map
			mu.Lock()
			defer mu.Unlock()

			// Add the Img object to the assets map
			assets["Assets"] = append(assets["Assets"], img)

			// Add the Img object to a category based on the first directory name
			dir, _ := filepath.Split(relPath)
			category := filepath.Clean(dir)
			assets[category] = append(assets[category], img)
		}

		return nil
	})

	if err != nil {
		panic(err)
	}
}

func main() {

	app := http.NewServeMux()

	fileServer := http.FileServer(http.Dir(imageDir))

	app.Handle("GET /assets/", http.StripPrefix("/assets/", fileServer))

	app.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		templateContent, err := content.ReadFile("templates/home.html")

		if err != nil {
			log.Print("template parsing error: ", err)
			return
		}

		tmpl, err := template.New("home").Parse(string(templateContent))

		if err != nil {
			log.Print("template parsing error: ", err)
		}

		tmpl.Execute(w, assets)
	})

	app.HandleFunc("GET /paginate", paginateHandler)

	log.Printf("Server listening on %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, app))
}

func isImage(filename string) bool {
	supportedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		// Add more extensions as needed
	}

	ext := strings.ToLower(filepath.Ext(filename))
	return supportedExtensions[ext]
}

func getImagesByLimit(page, limit int) []Img {
	// Lock the mutex before accessing the assets map
	mu.Lock()
	defer mu.Unlock()

	return assets["Assets"][page*limit : (page*limit)+limit]
}

func paginateHandler(w http.ResponseWriter, r *http.Request) {
	// Lock the mutex before accessing the assets map
	mu.Lock()
	defer mu.Unlock()

	// Get the requested page number from the query parameters
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	// Set the default page number to 1 if not provided
	if page == "" {
		page = "1"
	}

	if limit == "" {
		limit = "20"
	}

	// Convert the page number to an integer
	pageNumber := 1
	if num, err := strconv.Atoi(page); err == nil && num > 0 {
		pageNumber = num
	}

	limitNumber := 20

	if num, err := strconv.Atoi(limit); err == nil && num > 0 {
		limitNumber = num
	}

	// Determine the start and end indices for the requested page
	startIndex := (pageNumber - 1) * limitNumber
	endIndex := startIndex + limitNumber

	// Retrieve the assets for the "all" category
	allAssets := assets["Assets"]

	// Ensure that the start and end indices are within bounds
	if startIndex >= len(allAssets) {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}
	if endIndex > len(allAssets) {
		endIndex = len(allAssets)
	}

	// Extract the assets for the requested page
	paginatedAssets := allAssets[startIndex:endIndex]

	// Convert the assets to JSON and write to the response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(paginatedAssets); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
