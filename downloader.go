/*
*
Worker-optimized downloading
Stress tested with 100k URLs from GitHub
*
*/

package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"golang.org/x/exp/rand"
)

var (
	timeoutSeconds  = 30
	maxRetries      = 5
	userAgentString = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"
)

func scrapeURL(url string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Create the collector once at the beginning of the function
	c := colly.NewCollector()
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", userAgentString)
	})
	c.SetRequestTimeout(time.Duration(timeoutSeconds) * time.Second)

	var dataReceived bool
	// Handle response
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode != http.StatusOK {
			log.Printf("Received non-OK response: %s (status: %d)\n", url, r.StatusCode)
			return
		}
		dataReceived = true
		responseString := url + "\n---\n" + string(r.Body)

		err := os.WriteFile(makeCacheFilename(url), []byte(responseString), 0644)
		if err != nil {
			log.Printf("Failed to write response body to file: %s\n", err)
		} else {
			log.Printf("Content from %s saved to %s\n", url, makeCacheFilename(url))
		}
	})

	var retryCount int
	for {
		err := c.Visit(url)
		log.Printf("Visiting %s", url)

		if err != nil {
			log.Printf("Failed to visit URL %s: %s\n", url, err)
		}

		if dataReceived {
			break // Exit if data has been successfully received
		}

		retryCount++
		if retryCount >= maxRetries {
			log.Printf("Maximum retries reached for URL %s\n", url)
			break
		}

		waitTime := time.Duration(1+rand.Intn(timeoutSeconds)) * time.Second
		log.Printf("No data received from %s, retrying in %v... (%d/%d)\n", url, waitTime, retryCount, maxRetries)
		time.Sleep(waitTime)
	}
}

func fetchFromUrlList(urls []string) []string {

	var wg sync.WaitGroup

	urlChan := make(chan string)

	for i := 0; i < *maxWorkers; i++ {
		go func() {
			for url := range urlChan {
				if fileExists(makeCacheFilename(url)) {
					log.Printf("Skipping %s as it is already cached", url)
					continue
				}
				wg.Add(1)
				scrapeURL(url, &wg)
			}
		}()
	}

	for _, url := range urls {
		urlChan <- url
	}

	close(urlChan)
	wg.Wait()

	var successfulDownloads []string

	cachedFiles, _ := listCachedFiles()
	for _, url := range urls {
		cachedFileName := makeCacheFilename(url)
		if sliceContainsString(cachedFiles, cachedFileName) {
			successfulDownloads = append(successfulDownloads, cachedFileName)
		}
	}

	return successfulDownloads

}
