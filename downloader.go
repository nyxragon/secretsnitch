/*
*
Worker-optimized downloading
Stress tested with 100k URLs from GitHub
*
*/

package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/exp/rand"
)

var (
	timeoutSeconds = 30
	userAgentList  = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; Trident/7.0; AS; rv:11.0) like Gecko",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:92.0) Gecko/20100101 Firefox/92.0",
	}
)

func scrapeURL(url string) {
	var retryCount int

	for {
		client := &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Failed to create request for URL %s: %s\n", url, err)
			break
		}

		req.Header.Set("User-Agent", userAgentList[rand.Intn(len(userAgentList))])
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Set("Connection", "keep-alive")

		resp, err := client.Do(req)

		if err != nil {
			log.Printf("ERR Failed to send request to URL %s: %s\n", url, err)
		} else {
			defer resp.Body.Close() // Ensure the response body is closed after reading
			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("Failed to read response body from %s: %s\n", url, err)
				} else {
					responseString := url + "\n---\n" + string(body)

					err = os.WriteFile(makeCacheFilename(url), []byte(responseString), 0644)
					if err != nil {
						log.Printf("Failed to write response body to file: %s\n", err)
					} else {
						log.Printf("Content from %s saved to %s\n", url, cacheDir)
					}
				}
				break
			} else {
				log.Printf("Received non-OK HTTP status from %s: %s\n", url, resp.Status)
			}
		}

		retryCount++

		if retryCount >= *maxRetries {
			log.Printf("Maximum retries reached for URL %s\n", url)
			break
		}

		waitTime := time.Duration(1+rand.Intn(timeoutSeconds)) * time.Second
		log.Printf("No data received from %s, retrying in %v... (%d/%d)\n", url, waitTime, retryCount, *maxRetries)
		time.Sleep(waitTime)
	}
}

func fetchFromUrlList(urls []string) []string {
	var wg sync.WaitGroup
	urlChan := make(chan string)
	var processedUrls []string

	var successfulDownloads []string

	for _, url := range urls {
		if !fileExists(makeCacheFilename(url)) {
			processedUrls = append(processedUrls, url)
		} else {
			log.Printf("Skipping %s as it is already cached at %s", url, makeCacheFilename(url))
			successfulDownloads = append(successfulDownloads, makeCacheFilename(url))
		}
	}

	for i := 0; i < *maxWorkers; i++ {
		go func() {
			for url := range urlChan {
				wg.Add(1)
				defer wg.Done()
				scrapeURL(url)
			}
		}()
	}

	for _, url := range processedUrls {
		urlChan <- url
	}

	close(urlChan)
	wg.Wait()

	cachedFiles, _ := listCachedFiles()
	for _, url := range processedUrls {
		cachedFileName := makeCacheFilename(url)
		if sliceContainsString(cachedFiles, cachedFileName) {
			successfulDownloads = append(successfulDownloads, cachedFileName)
		}
	}

	return successfulDownloads
}
