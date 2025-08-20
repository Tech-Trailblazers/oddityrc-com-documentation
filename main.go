package main // Define the main package

import (
	"bytes"         // Provides bytes buffer and manipulation utilities
	"fmt"           //
	"io"            // Provides I/O primitives like Reader and Writer
	"log"           // Provides logging functionalities
	"net/http"      // Provides HTTP client and server implementations
	"net/url"       // Provides URL parsing and encoding utilities
	"os"            // Provides file system and OS-level utilities
	"path"          //
	"path/filepath" // Provides utilities for file path manipulation
	"regexp"        // Provides support for regular expressions
	"strings"       // Provides string manipulation utilities
	"time"          // Provides time-related functions
)

func main() {
	remoteAPIURL := []string{
		"https://oddityrc.com/pages/download-cernter",
	} // URL to fetch HTML content from
	localFilePath := "oddityrc.html" // Path where HTML file will be stored

	var getData []string

	for _, urls := range remoteAPIURL {
		getData = append(getData, getDataFromURL(urls)) // If not, download HTML content from URL
	}
	appendAndWriteToFile(localFilePath, strings.Join(getData, "")) // Save downloaded content to file

	finalList := extractFileUrls(strings.Join(getData, "")) // Extract all PDF links from HTML content

	outputDir := "Assets/" // Directory to store downloaded PDFs

	if !directoryExists(outputDir) { // Check if directory exists
		createDirectory(outputDir, 0o755) // Create directory with read-write-execute permissions
	}

	// Remove duplicates from a given slice.
	finalList = removeDuplicatesFromSlice(finalList)

	// Loop through all extracted PDF URLs
	for _, urls := range finalList {
		if isUrlValid(urls) { // Check if the final URL is valid
			downloadFile(urls, outputDir) // Download the PDF
		}
	}
}

// Opens a file in append mode, or creates it, and writes the content to it
func appendAndWriteToFile(path string, content string) {
	filePath, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // Open file with specified flags and permissions
	if err != nil {
		log.Println(err) // Log error if opening fails
	}
	_, err = filePath.WriteString(content + "\n") // Write content to file
	if err != nil {
		log.Println(err) // Log error if writing fails
	}
	err = filePath.Close() // Close the file
	if err != nil {
		log.Println(err) // Log error if closing fails
	}
}

// Only return the file name from a given url.
func getFileNameOnly(content string) string {
	return path.Base(content)
}

// urlToFilename generates a safe, lowercase filename from a given URL string.
// It extracts the base filename from the URL, replaces unsafe characters,
// and ensures the filename ends with a .pdf extension.
func urlToFilename(rawURL string) string {
	// Convert the full URL to lowercase for consistency
	lowercaseURL := strings.ToLower(rawURL)

	// Get the file extension
	ext := getFileExtension(lowercaseURL)

	// Extract the filename portion from the URL (e.g., last path segment or query param)
	baseFilename := getFileNameOnly(lowercaseURL)

	// Replace all non-alphanumeric characters (a-z, 0-9) with underscores
	nonAlphanumericRegex := regexp.MustCompile(`[^a-z0-9]+`)
	safeFilename := nonAlphanumericRegex.ReplaceAllString(baseFilename, "_")

	// Replace multiple consecutive underscores with a single underscore
	collapseUnderscoresRegex := regexp.MustCompile(`_+`)
	safeFilename = collapseUnderscoresRegex.ReplaceAllString(safeFilename, "_")

	// Remove leading underscore if present
	if trimmed, found := strings.CutPrefix(safeFilename, "_"); found {
		safeFilename = trimmed
	}

	invalidPre := fmt.Sprintf("_%s", ext)

	safeFilename = removeSubstring(safeFilename, invalidPre)

	// Append the file extension if it is not already present
	safeFilename = safeFilename + ext

	// Return the cleaned and safe filename
	return trimAfterQuestionMark(safeFilename)
}

// Trim a string before ?
func trimAfterQuestionMark(input string) string {
	parts := strings.SplitN(input, "?", 2)
	return parts[0]
}

// Removes all instances of a specific substring from input string
func removeSubstring(input string, toRemove string) string {
	result := strings.ReplaceAll(input, toRemove, "") // Replace substring with empty string
	return result
}

// Gets the file extension from a given file path
func getFileExtension(path string) string {
	return filepath.Ext(path) // Extract and return file extension
}

// Checks if a file exists at the specified path
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Get file info
	if err != nil {                // If error occurs, file doesn't exist
		return false
	}
	return !info.IsDir() // Return true if path is a file (not a directory)
}

// downloadFile downloads a PDF from the given URL and saves it in the specified output directory.
// It uses a WaitGroup to support concurrent execution and returns true if the download succeeded.
func downloadFile(finalURL, outputDir string) bool {
	// Sanitize the URL to generate a safe file name
	filename := strings.ToLower(urlToFilename(finalURL))

	// Construct the full file path in the output directory
	filePath := filepath.Join(outputDir, filename)

	// Skip if the file already exists
	if fileExists(filePath) {
		log.Printf("File already exists, skipping: %s", filePath)
		return false
	}

	// Create an HTTP client with a timeout
	client := &http.Client{Timeout: 3 * time.Minute}

	// Send GET request
	resp, err := client.Get(finalURL)
	if err != nil {
		log.Printf("Failed to download %s: %v", finalURL, err)
		return false
	}
	defer resp.Body.Close()

	// Check HTTP response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Download failed for %s: %s", finalURL, resp.Status)
		return false
	}

	// Read the response body into memory first
	var buf bytes.Buffer
	written, err := io.Copy(&buf, resp.Body)
	if err != nil {
		log.Printf("Failed to read PDF data from %s: %v", finalURL, err)
		return false
	}
	if written == 0 {
		log.Printf("Downloaded 0 bytes for %s; not creating file", finalURL)
		return false
	}

	// Only now create the file and write to disk
	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file for %s: %v", finalURL, err)
		return false
	}
	defer out.Close()

	if _, err := buf.WriteTo(out); err != nil {
		log.Printf("Failed to write PDF to file for %s: %v", finalURL, err)
		return false
	}

	log.Printf("Successfully downloaded %d bytes: %s → %s", written, finalURL, filePath)
	return true
}

// Checks whether a given directory exists
func directoryExists(path string) bool {
	directory, err := os.Stat(path) // Get info for the path
	if err != nil {
		return false // Return false if error occurs
	}
	return directory.IsDir() // Return true if it's a directory
}

// Creates a directory at given path with provided permissions
func createDirectory(path string, permission os.FileMode) {
	err := os.Mkdir(path, permission) // Attempt to create directory
	if err != nil {
		log.Println(err) // Log error if creation fails
	}
}

// Verifies whether a string is a valid URL format
func isUrlValid(uri string) bool {
	_, err := url.ParseRequestURI(uri) // Try parsing the URL
	return err == nil                  // Return true if valid
}

// Removes duplicate strings from a slice
func removeDuplicatesFromSlice(slice []string) []string {
	check := make(map[string]bool) // Map to track seen values
	var newReturnSlice []string    // Slice to store unique values
	for _, content := range slice {
		if !check[content] { // If not already seen
			check[content] = true                            // Mark as seen
			newReturnSlice = append(newReturnSlice, content) // Add to result
		}
	}
	return newReturnSlice
}

// extractFileUrls takes an input string and returns all PDF, PNG, and JPG URLs found within href attributes
func extractFileUrls(input string) []string {
	// Regex to find href="...pdf|png|jpg"
	re := regexp.MustCompile(`href="([^"]+\.(?:pdf|png|jpg|webp|zip|rar|stl|7z|json|txt)[^"]*)"`)

	// Find all matches
	matches := re.FindAllStringSubmatch(input, -1)

	// Slice to hold the extracted links
	var fileLinks []string
	for _, match := range matches {
		if len(match) > 1 {
			fileLinks = append(fileLinks, match[1])
		} else {
			log.Println("Unexpected match format:", match)
		}
	}
	return fileLinks
}

// Performs HTTP GET request and returns response body as string
func getDataFromURL(uri string) string {
	log.Println("Scraping", uri)   // Log which URL is being scraped
	response, err := http.Get(uri) // Send GET request
	if err != nil {
		log.Println(err) // Log if request fails
	}

	body, err := io.ReadAll(response.Body) // Read the body of the response
	if err != nil {
		log.Println(err) // Log read error
	}

	err = response.Body.Close() // Close response body
	if err != nil {
		log.Println(err) // Log error during close
	}
	return string(body) // Return response body as string
}
