package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/0x4f53/textsubs"
	"github.com/dlclark/regexp2"
	"mvdan.cc/xurls/v2"
)

type Secret struct {
	Provider    string
	ServiceName string
	Variable    string
	Secret      string
	Entropy     float64
	Tags        []string
}

type ToolData struct {
	Tool            string
	ScanTimestamp   string
	Secrets         []Secret
	CacheFile       string
	SourceUrl       string
	CapturedDomains []string
	CapturedURLs    []string
}

func grabURLs(text string) []string {

	var captured []string
	location := substringBeforeFirst(text, "---")

	baseUrl, _ := baseURL(location)

	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl += "/"
	}

	text = strings.Replace(text, location, "", -1)

	scanner := bufio.NewScanner(strings.NewReader(text))

	rx := xurls.Relaxed()
	rxUrls := rx.FindAllString(text, -1)
	captured = append(captured, rxUrls...)

	splitText := strings.Split(text, "{")

	protocol := substringBeforeFirst(location, "://")

	for _, line := range splitText {

		re := regexp.MustCompile(`(?:href|src|action|cite|data|formaction|poster)\s*=\s*["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(line, -1)

		for _, matchGroups := range matches {

			resource := matchGroups[1]

			if !strings.Contains(resource, "://") && !strings.HasPrefix(resource, "//") {
				resource = baseUrl + resource
			} else if !strings.Contains(resource, "://") && strings.HasPrefix(resource, "//") {
				resource = protocol + ":" + resource
			}

			captured = append(captured, resource)
		}

	}

	if err := scanner.Err(); err != nil {
		log.Printf("error reading string: %s\n", err)
	}

	var urls []string
	for _, url := range captured {
		if strings.Contains(url, "://") && strings.Contains(url, ".") {
			urls = append(urls, url)
		}
	}

	return removeDuplicates(urls)

}

// The brains of secretsnitch. Runs a bunch of checks including regexes, provider matching, entropy etc. Refer to docs for more
//
// Input:
//
// text (string) - text to search secrets in
//
// Output:
//
// ToolData - proprietary data type containing scan results
func FindSecrets(text string) ToolData {

	var output ToolData
	var secrets []Secret

	var tags []string

	domains, _ := textsubs.DomainsOnly(text, false)
	domains = textsubs.Resolve(domains)

	splitText := strings.Split(text, "{")

	var mu sync.Mutex
	var wg sync.WaitGroup

	lineChan := make(chan string, *maxWorkers)

	for i := 0; i < *maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range lineChan {
				data, _ := extractKeyValuePairs(line)

				for key, value := range data {
					for _, provider := range signatures {
						for service, regex := range provider.Keys {
							re := regexp2.MustCompile(regex, 0)
							match, _ := re.MatchString(value)

							if match && !(containsBlacklisted(key) || containsBlacklisted(value)) {
								mu.Lock()
								tags = append(tags, "regexMatched")
								mu.Unlock()

								entropy := EntropyPercentage(value)
								if entropy > 66.6 {
									mu.Lock()
									tags = append(tags, "highEntropy")
									mu.Unlock()
								}

								providerString := strings.ToLower(strings.Split(provider.Name, ".")[0])

								if strings.Contains(strings.ToLower(text), providerString) {
									mu.Lock()
									tags = append(tags, "providerDetected")
									mu.Unlock()
								}

								secret := Secret{
									Provider:    provider.Name,
									ServiceName: service,
									Variable:    key,
									Secret:      value,
									Entropy:     entropy,
									Tags:        removeDuplicates(tags),
								}

								mu.Lock()
								secrets = append(secrets, secret)
								mu.Unlock()
							}
						}
					}
				}
			}
		}()
	}

	for _, line := range splitText {
		lineChan <- line
	}

	close(lineChan)

	wg.Wait()

	sourceUrl := grabSourceUrl(text)
	capturedUrls := grabURLs(text)

	output = ToolData{
		Tool:            "secretsnitch",
		ScanTimestamp:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		SourceUrl:       sourceUrl,
		Secrets:         secrets,
		CapturedDomains: domains,
		CapturedURLs:    removeDuplicates(capturedUrls),
	}

	return output
}

func logSecrets(secrets ToolData, outputFile *string) {
	unindented, _ := json.Marshal(secrets)
	appendToFile(*outputFile, string(unindented))
	indented, _ := json.MarshalIndent(secrets, "", "	")
	fmt.Println(string(indented))
}

func ScanFiles(files []string) {
	var wg sync.WaitGroup
	fileChan := make(chan string)

	for i := 0; i < *maxWorkers; i++ {
		go func() {
			for file := range fileChan {
				wg.Add(1)
				scanFile(file, &wg)
			}
		}()
	}

	for _, file := range files {
		fileChan <- file
	}

	close(fileChan)
	wg.Wait()
}

var recursionCount = 0

func scanFile(filePath string, wg *sync.WaitGroup) {
	defer wg.Done()

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading file %s: %v\n", filePath, err)
		return
	}

	text := string(data)

	sourceUrl := grabSourceUrl(string(data))
	if sourceUrl != "" {
		log.Println("Searching for secrets in: " + sourceUrl)
	} else {
		log.Println("Searching for secrets in: " + filePath)
	}

	secrets := FindSecrets(text)
	secrets.CacheFile = filePath

	if (*secretsOptional && len(secrets.Secrets) == 0) || len(secrets.Secrets) > 0 {
		logSecrets(secrets, outputFile)
	}

	if *maxRecursions > 0 {
		recursionCount++
		urls := grabURLs(string(data))
		successfulUrls := fetchFromUrlList(urls)

		if recursionCount < *maxRecursions {
			ScanFiles(successfulUrls)
		}
	}
}
