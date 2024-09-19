package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type CacheItem struct {
	content   string
	timestamp time.Time
}

type Cache struct {
	data map[string]CacheItem
	mu   sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		data: make(map[string]CacheItem),
	}
}

func (c *Cache) Get(url string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.data[url]
	if found {
		if time.Since(item.timestamp) > 10*time.Minute {
			return "", false
		}
		return item.content, true
	}
	return "", false
}

func (c *Cache) Set(url, content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[url] = CacheItem{
		content:   content,
		timestamp: time.Now(),
	}
}

func scrapeWithCache(url string, cache *Cache, keyword string) {
	cachedContentReturned, found := cache.Get(url)
	if found {
		lines := strings.Split(cachedContentReturned, "\n")
		var matchedTalks string
		addedLines := make(map[string]bool) // Track lines that have been added to matchedTalks

		for i := 0; i < len(lines); i++ {
			line := strings.ToLower(lines[i])
			if strings.Contains(line, strings.ToLower(keyword)) && !addedLines[line] {
				// Append current line and next line (if it exists)
				if i+1 < len(lines) {
					matchedTalks += fmt.Sprintf("%s\n%s\n", lines[i], lines[i+1])
					addedLines[line] = true       // Mark line as added
					addedLines[lines[i+1]] = true // Mark the next line as added
				} else {
					matchedTalks += fmt.Sprintf("%s\n", lines[i])
					addedLines[line] = true
				}
			}
		}
		fmt.Println("Sirviendo desde cache:")
		fmt.Println(matchedTalks)
		return
	}

	fmt.Println("Trayendo nuevo contenido...")
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("Fallo al traer la página: %s, Status code: %d", url, resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var content, time_of_talk string
	count := 1
	cachedContent := ""
	doc.Find("strong").Each(func(i int, s *goquery.Selection) {
		linkText := s.Text()
		lines := strings.Split(s.Parent().Text(), "\n")

		if len(lines) > 0 {
			time_of_talk = strings.TrimSpace(lines[len(lines)-1])
		}
		cachedContent += fmt.Sprintf("%s\nDía: %s\n", linkText, time_of_talk)
		if strings.Contains(strings.ToLower(linkText), strings.ToLower(keyword)) && !strings.Contains(strings.ToLower(content), strings.ToLower(linkText)) {
			content += fmt.Sprintf("%s\nDía: %s\n", linkText, time_of_talk)
			count += 1
		}

	})

	cache.Set(url, cachedContent)

	fmt.Println(content)
}

func main() {
	url := "https://nerdear.la/es/agenda/"

	// Initialize the cache
	cache := NewCache()
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Press Ctrl + C to stop...")

	for {
		select {
		case <-sigint:
			fmt.Println("\nReceived interrupt signal. Exiting...")
			return
		default:
			var keyword string

			// Prompt for a new keyword
			fmt.Print("Ingrese una palabra a buscar: ")
			_, err := fmt.Scanln(&keyword)
			if err != nil {
				log.Fatalf("Error reading input: %v", err)
			}

			// Scrape the webpage with cache enabled
			scrapeWithCache(url, cache, keyword)

			// Optionally, wait for a specified interval before prompting for the next keyword (e.g., 1 second)
			time.Sleep(1 * time.Second)
		}
	}
}
