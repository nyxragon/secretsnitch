package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/0x4f53/textsubs"
	"mvdan.cc/xurls/v2"
)

type SecretData struct {
	Provider       string
	ServiceName    string
	Variable       string
	Secret         string
	TsallisEntropy float64
	Position       string
	Tags           []string
}

type ToolData struct {
	Tool            string
	ScanTimestamp   string
	Secret          SecretData
	CacheFile       string
	SourceUrl       string
	CapturedDomains []string
	CapturedURLs    []string
}

func grabURLs(text string) []string {

	var captured []string
	sourceUrl := grabSourceUrl(text)

	baseUrl, _ := baseURL(sourceUrl)

	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl += "/"
	}

	scanner := bufio.NewScanner(strings.NewReader(text))

	rx := xurls.Relaxed()
	rxUrls := rx.FindAllString(text, -1)

	captured = append(captured, rxUrls...)

	// split JS files for single quotes
	for _, url := range rxUrls {
		if strings.Count(url, "'") > 3 {
			splitUrls := strings.Split(url, "'")
			captured = append(captured, splitUrls...)
		}
	}

	// split JS files for double quotes
	for _, url := range rxUrls {
		if strings.Count(url, "\"") > 3 {
			splitUrls := strings.Split(url, "\"")
			captured = append(captured, splitUrls...)
		}
	}

	var splitText []string
	splitText = append(splitText, strings.Split(text, "{")...)
	splitText = append(splitText, strings.Split(text, ",")...)
	splitText = append(splitText, strings.Split(text, "\n")...)
	splitText = removeDuplicates(splitText)

	for _, line := range splitText {

		re := regexp.MustCompile(`(?:href|src|action|cite|data|formaction|poster)\s*=\s*["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(line, -1)

		for _, matchGroups := range matches {

			resource := matchGroups[1]

			if !strings.Contains(resource, "://") && !strings.HasPrefix(resource, "//") {
				resource = strings.TrimPrefix(resource, "/")
				resource = baseUrl + resource
			} else if !strings.Contains(resource, "://") && strings.HasPrefix(resource, "//") {
				resource = "https:" + resource
				if strings.Contains(resource, "http://") {
					resource = "http:" + resource
				}
			}

			captured = append(captured, resource)
		}

	}

	if err := scanner.Err(); err != nil {
		log.Printf("error reading string: %s\n", err)
	}

	var urls []string
	for _, url := range captured {
		if strings.Contains(url, "://") && strings.Contains(url, ".") && !strings.Contains(url, "'") && url != sourceUrl {
			urls = append(urls, url)
		}
	}

	return removeDuplicates(urls)

}

// to prevent duplicates
var capturedSecrets []SecretData

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

	sourceUrl := grabSourceUrl(text)
	cacheFileName := makeCacheFilename(sourceUrl)

	// Secret collection

	text = strings.Replace(text, truncateGitBinaryData(text), "", -1) // Remove Git Binary data

	splitText := splitText(text) // Break walls of text

	// log.Println("Scanning " + strconv.Itoa(len(splitText)) + " tokens for secrets")

	domains, _ := textsubs.DomainsOnly(text, false)
	// domains = textsubs.Resolve(domains)

	capturedURLs := grabURLs(text)

	for _, line := range splitText {

		secretFound := false

		data, _ := extractKeyValuePairs(strings.Replace(line, sourceUrl, "", 1))

		for _, variable := range data {

			if containsBlacklisted(variable.Value) || containsBlacklisted(variable.Name) {
				continue
			}

			for _, provider := range signatures {
				for service, regex := range provider.Items {

					variableNameMatch := regex.FindAllString(variable.Name, 1)
					variableValueMatch := regex.FindAllString(variable.Value, 1)

					var tags []string

					match := variableValueMatch

					if strings.Contains(strings.ToLower(service), "block") {
						/* files like SSH and PGP keys do not have "variables". As such
						 * matching needs to be performed on the entire file.
						 */
						match = regex.FindAllString(text, -1)
						if len(match) > 0 {
							variable.Name = strings.Split(match[0], "\n")[0]
							variable.Value = match[0]
							tags = append(tags, "textBlockMatched")
						}
					}

					if strings.Contains(strings.ToLower(service), "variable") {
						/* this is to match the names of variables where the actual secret
						 * has no pattern and the name of the variable is a better indicator
						 * this approach yields more results, but also has a lot of false positives
						 * and negatives.
						 */
						tags = append(tags, "variableNameMatched")
						match = variableNameMatch
						/* sometimes, password fields are set to "password",
						 * so checking these is excessive and may lead to a false negative.
						 */
					}

					if len(match) > 0 {

						entropy := tsallisEntropy(match[0], 2)

						providerString := strings.ToLower(strings.Split(provider.Name, ".")[0])
						if strings.Contains(strings.ToLower(text), strings.ToLower(providerString)) && !strings.EqualFold(provider.Name, "Generic") {
							tags = append(tags, "providerDetected")
						}

						if len(variable.Value) > 16 {
							tags = append(tags, "longString")
						}

						if len(variableValueMatch) > 0 {
							tags = append(tags, "variableValueMatched")
						}

						row, column := findPosition(text, variable.Value, line)
						position := strconv.Itoa(row) + ":" + strconv.Itoa(column)

						if len(variable.Value) >= 8 {

							secret := SecretData{
								Provider:       provider.Name,
								ServiceName:    service,
								Variable:       variable.Name,
								Secret:         variable.Value,
								Position:       position,
								TsallisEntropy: entropy,
								Tags:           tags,
							}

							output = ToolData{
								Tool:            "secretsnitch",
								ScanTimestamp:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"),
								SourceUrl:       sourceUrl,
								Secret:          secret,
								CacheFile:       makeCacheFilename(sourceUrl),
								CapturedDomains: domains,
								CapturedURLs:    capturedURLs,
							}

							if !containsSecret(capturedSecrets, secret) {
								logSecret(output, outputFile)
								capturedSecrets = append(capturedSecrets, secret)
								printSecret(secret, sourceUrl, cacheFileName)
								secretFound = true
							}

						}
					}

				}
				if secretFound {
					break
				}
			}
			if secretFound {
				break
			}

		}

	}

	if len(domains) > 0 {
		fmt.Printf("\nDOMAINS FOUND:\n")
		for index, item := range domains {
			fmt.Print(item)
			if index != len(domains)-1 {
				fmt.Printf(", ")
			}
		}
		fmt.Println()
	}

	if len(capturedURLs) > 0 {
		fmt.Printf("\nURLs FOUND:\n")
		for index, item := range capturedURLs {
			fmt.Printf("\t- %d. %s\n", index+1, item)
		}
	}

	return output
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

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading file %s: %v\n", filePath, err)
		return
	}

	text := string(data)

	if len(text) >= maxFileSize {
		log.Printf("Skipping this file as it is >= %d MB (%d MB)", maxFileSize/1024/1024, len(text)/1024/1024)
		return
	}

	sourceUrl := grabSourceUrl(text)

	if sourceUrl != "" {
		log.Printf("Searching for secrets in: %s (cached at: %s)", sourceUrl, makeCacheFilename(sourceUrl))
	} else {
		log.Printf("Searching for secrets in: %s", filePath)
	}

	FindSecrets(text)

	if *maxRecursions > 0 {
		recursionCount++
		urls := grabURLs(string(data))
		successfulUrls := fetchFromUrlList(urls, true)

		if recursionCount <= *maxRecursions {
			ScanFiles(successfulUrls)
		}
	}

	defer wg.Done()

}
