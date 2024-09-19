package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
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
	cachedContent, found := cache.Get(url)
	if found {
		fmt.Println("Sirviendo desde cache:")
		fmt.Println(cachedContent)
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
	doc.Find("strong").Each(func(i int, s *goquery.Selection) {
		linkText := s.Text()
		lines := strings.Split(s.Parent().Text(), "\n")
		if len(lines) > 0 {
			time_of_talk = strings.TrimSpace(lines[len(lines)-1])
		}
		if strings.Contains(strings.ToLower(linkText), strings.ToLower(keyword)) && !strings.Contains(strings.ToLower(content), strings.ToLower(linkText)) {
			content += fmt.Sprintf("%dº Charla: %s\nDía: %s\n", count, linkText, time_of_talk)
			count += 1
		}

	})

	cache.Set(url, content)

	fmt.Println(content)
}

func main() {
	var keyword string

	url := "https://nerdear.la/es/agenda/"

	fmt.Print("Ingrese una palabra a buscar: ")
	_, err := fmt.Scanln(&keyword)
	if err != nil {
		log.Fatalf("Error leyendo el input: %v", err)
	}

	// Initialize the cache
	cache := NewCache()

	scrapeWithCache(url, cache, keyword)
}
