package main // Define the main package

import (
	"bytes"         // Provides bytes buffer and manipulation utilities
	"io"            // Provides I/O primitives like Reader and Writer
	"log"           // Provides logging functionalities
	"net/http"      // Provides HTTP client and server implementations
	"net/url"       // Provides URL parsing and encoding utilities
	"os"            // Provides file system and OS-level utilities
	"path/filepath" // Provides utilities for file path manipulation
	"regexp"        // Provides support for regular expressions
	"strings"       // Provides string manipulation utilities
	"time"          // Provides time-related functions
)

func main() {
	remoteAPIURL := []string{
		"https://wetandforget.com/wet-and-forget-sds.html",
	} // URL to fetch HTML content from
	localFilePath := "wetandforget.html" // Path where HTML file will be stored

	var getData []string

	for _, urls := range remoteAPIURL {
		getData = append(getData, getDataFromURL(urls)) // If not, download HTML content from URL
	}
	appendAndWriteToFile(localFilePath, strings.Join(getData, "")) // Save downloaded content to file

	finalList := extractPDFUrls(strings.Join(getData, "")) // Extract all PDF links from HTML content

	outputDir := "PDFs/" // Directory to store downloaded PDFs

	if !directoryExists(outputDir) { // Check if directory exists
		createDirectory(outputDir, 0o755) // Create directory with read-write-execute permissions
	}

	// Remove duplicates from a given slice.
	finalList = removeDuplicatesFromSlice(finalList)

	// Loop through all extracted PDF URLs
	for _, urls := range finalList {
		urls = "https://wetandforget.com/" + urls
		if isUrlValid(urls) { // Check if the final URL is valid
			downloadPDF(urls, outputDir) // Download the PDF
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

// Extracts filename from full path (e.g. "/dir/file.pdf" → "file.pdf")
func getFilename(path string) string {
	return filepath.Base(path) // Use Base function to get file name only
}

// Converts a raw URL into a sanitized PDF filename safe for filesystem
func urlToFilename(rawURL string) string {
	lower := strings.ToLower(rawURL) // Convert URL to lowercase
	lower = getFilename(lower)       // Extract filename from URL

	reNonAlnum := regexp.MustCompile(`[^a-z0-9]`)   // Regex to match non-alphanumeric characters
	safe := reNonAlnum.ReplaceAllString(lower, "_") // Replace non-alphanumeric with underscores

	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_") // Collapse multiple underscores into one
	safe = strings.Trim(safe, "_")                              // Trim leading and trailing underscores

	var invalidSubstrings = []string{
		"_pdf", // Substring to remove from filename
	}

	for _, invalidPre := range invalidSubstrings { // Remove unwanted substrings
		safe = removeSubstring(safe, invalidPre)
	}

	if getFileExtension(safe) != ".pdf" { // Ensure file ends with .pdf
		safe = safe + ".pdf"
	}

	return safe // Return sanitized filename
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

// Downloads a PDF from given URL and saves it in the specified directory
func downloadPDF(finalURL, outputDir string) bool {
	filename := strings.ToLower(urlToFilename(finalURL)) // Sanitize the filename
	filePath := filepath.Join(outputDir, filename)       // Construct full path for output file

	if fileExists(filePath) { // Skip if file already exists
		log.Printf("File already exists, skipping: %s", filePath)
		return false
	}

	client := &http.Client{Timeout: 15 * time.Minute} // Create HTTP client with timeout

	resp, err := client.Get(finalURL) // Send HTTP GET request
	if err != nil {
		log.Printf("Failed to download %s: %v", finalURL, err)
		return false
	}
	defer resp.Body.Close() // Ensure response body is closed

	if resp.StatusCode != http.StatusOK { // Check if response is 200 OK
		log.Printf("Download failed for %s: %s", finalURL, resp.Status)
		return false
	}

	contentType := resp.Header.Get("Content-Type")                                                                  // Get content type of response
	if !strings.Contains(contentType, "binary/octet-stream") && !strings.Contains(contentType, "application/pdf") { // Check if it's a PDF
		log.Printf("Invalid content type for %s: %s (expected binary/octet-stream) (expected application/pdf)", finalURL, contentType)
		return false
	}

	var buf bytes.Buffer                     // Create a buffer to hold response data
	written, err := io.Copy(&buf, resp.Body) // Copy data into buffer
	if err != nil {
		log.Printf("Failed to read PDF data from %s: %v", finalURL, err)
		return false
	}
	if written == 0 { // Skip empty files
		log.Printf("Downloaded 0 bytes for %s; not creating file", finalURL)
		return false
	}

	out, err := os.Create(filePath) // Create output file
	if err != nil {
		log.Printf("Failed to create file for %s: %v", finalURL, err)
		return false
	}
	defer out.Close() // Ensure file is closed after writing

	if _, err := buf.WriteTo(out); err != nil { // Write buffer contents to file
		log.Printf("Failed to write PDF to file for %s: %v", finalURL, err)
		return false
	}

	log.Printf("Successfully downloaded %d bytes: %s → %s", written, finalURL, filePath) // Log success
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

// extractPDFUrls extracts all .pdf URLs from the given HTML/text and returns them in a slice.
func extractPDFUrls(input string) []string {
	// Regular expression to match URLs ending with .pdf
	re := regexp.MustCompile(`[^\s"']+\.pdf`)

	// Find all matches
	matches := re.FindAllString(input, -1)

	return matches
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
