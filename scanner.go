package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/0x4f53/textsubs"
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
	CapturedEmails  []string
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

	text = strings.Replace(text, truncateGitBinaryData(text), "", -1)
	splitText := splitText(text)

	domains, _ := textsubs.DomainsOnly(text, false)
	capturedURLs := extractURLs(text)

	capturedEmails := extractEmails(text)
	capturedEmails = removeDuplicates(capturedEmails)

	lineChan := make(chan string, len(splitText))
	resultChan := make(chan ToolData)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for line := range lineChan {
			var secretFound bool
			var result ToolData

			data, _ := extractKeyValuePairs(strings.Replace(line, sourceUrl, "", 1))

			for _, variable := range data {

				if containsBlacklistedNames(variable.Name) || containsBlacklistedValues(variable.Value) {
					continue
				}

				for _, provider := range signatures {
					for service, regex := range provider.Items {
						variableNameMatch := regex.FindAllString(variable.Name, 1)
						variableValueMatch := regex.FindAllString(variable.Value, 1)

						var tags []string
						match := variableValueMatch

						if strings.Contains(strings.ToLower(service), "block") {
							match = regex.FindAllString(text, -1)
							if len(match) > 0 {
								variable.Name = strings.Split(match[0], "\n")[0]
								variable.Value = match[0]
								tags = append(tags, "textBlockMatched")
							}
						}

						if strings.Contains(strings.ToLower(service), "variable") {
							tags = append(tags, "variableNameMatched")
							match = variableNameMatch
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

								result = ToolData{
									Tool:            "secretsnitch",
									ScanTimestamp:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"),
									SourceUrl:       sourceUrl,
									Secret:          secret,
									CacheFile:       cacheFileName,
									CapturedDomains: domains,
									CapturedURLs:    capturedURLs,
									CapturedEmails:  capturedEmails,
								}

								if !containsSecret(capturedSecrets, secret) {
									logSecret(result, outputFile)
									capturedSecrets = append(capturedSecrets, secret)
									printSecret(secret, sourceUrl, cacheFileName)
									secretFound = true
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

			if secretFound {
				resultChan <- result
			}
		}
	}

	for i := 0; i < *maxWorkers; i++ {
		wg.Add(1)
		go worker()
	}

	for _, line := range splitText {
		lineChan <- line
	}
	close(lineChan)

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for res := range resultChan {
		output = res
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
		//log.Printf("Searching for secrets in: %s (cached at: %s)", sourceUrl, makeCacheFilename(sourceUrl))
	} else {
		//log.Printf("Searching for secrets in: %s", filePath)
	}

	FindSecrets(text)

	if *maxRecursions > 0 {
		recursionCount++
		urls := extractURLs(string(data))
		successfulUrls := fetchFromUrlList(urls, true)

		if recursionCount <= *maxRecursions {
			ScanFiles(successfulUrls)
		}
	}

	defer wg.Done()

}
