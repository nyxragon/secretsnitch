package main

import (
	"bufio"
	"encoding/json"
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

type Secret struct {
	Provider       string
	ServiceName    string
	Variable       string
	Secret         string
	TsallisEntropy float64
	Tags           []string
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

	// remove metadata URL
	text = strings.Replace(text, location, "", -1)

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
		if strings.Contains(url, "://") && strings.Contains(url, ".") && !strings.Contains(url, "'") {
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

	// Secret collection

	// 1. Secret file detection
	privateKeys := parsePrivateKeys(text)
	if len(privateKeys) > 0 {
		for _, variable := range privateKeys {
			serviceName := "Secure Shell"
			if strings.Contains(strings.ToLower(variable.Name), "pgp") {
				serviceName = "PGP"
			}
			secret := Secret{
				Provider:       serviceName,
				ServiceName:    "Private Key",
				Variable:       variable.Name,
				Secret:         variable.Value,
				TsallisEntropy: 1.0,
				Tags:           []string{"longString", "providerDetected", "regexMatched"},
			}
			secrets = append(secrets, secret)
		}
	}

	// 2. Variable detection
	text = strings.ReplaceAll(text, `\"`, `"`)
	splitText := strings.Split(text, "{")
	splitText = append(splitText, strings.Split(text, ",")...)
	splitText = append(splitText, strings.Split(text, ";")...)
	splitText = append(splitText, strings.Split(text, "\n")...)
	splitText = removeDuplicates(splitText)
	log.Println("Scanning " + strconv.Itoa(len(splitText)) + "tokens")

	var mu sync.Mutex
	var wg sync.WaitGroup

	lineChan := make(chan string, *maxWorkers)

	for i := 0; i < *maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range lineChan {
				data, _ := extractKeyValuePairs(line)

				for _, variable := range data {

					if containsBlacklisted(variable.Value) {
						continue
					}

					for _, provider := range signatures {
						for service, regex := range provider.Keys {
							re := regexp.MustCompile(regex)
							variableNameMatch := re.FindAllString(variable.Name, 1)
							variableValueMatch := re.FindAllString(variable.Value, 1)

							match := variableValueMatch
							if strings.Contains(strings.ToLower(service), "variable") {
								match = variableNameMatch
							}

							if len(match) > 0 {

								mu.Lock()

								var tags []string

								tags = append(tags, "regexMatched")

								entropy := tsallisEntropy(match[0], 2)

								providerString := strings.ToLower(strings.Split(provider.Name, ".")[0])
								if strings.Contains(strings.ToLower(text), providerString) && !strings.EqualFold(provider.Name, "Generic") {
									tags = append(tags, "providerDetected")
								}

								if len(variable.Value) > 16 {
									tags = append(tags, "longString")
								}

								if len(variableValueMatch) > 0 {
									variable.Value = variableValueMatch[0]
								}

								if len(variable.Value) > 8 {
									secret := Secret{
										Provider:       provider.Name,
										ServiceName:    service,
										Variable:       variable.Name,
										Secret:         variable.Value,
										TsallisEntropy: entropy,
										Tags:           removeDuplicates(tags),
									}

									if !containsSecret(secrets, secret) {
										secrets = append(secrets, secret)
									}
								}

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

	output = ToolData{
		Tool:            "secretsnitch",
		ScanTimestamp:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		SourceUrl:       "",
		Secrets:         secrets,
		CapturedDomains: []string{},
		CapturedURLs:    []string{},
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

	sourceUrl := grabSourceUrl(text)
	if sourceUrl != "" {
		log.Println("Searching for secrets in: " + sourceUrl)
	} else {
		log.Println("Searching for secrets in: " + filePath)
	}

	text = strings.Replace(text, sourceUrl+"\n---\n", "", -1)
	secrets := FindSecrets(text)
	secrets.CacheFile = filePath

	// Metadata collection
	domains, _ := textsubs.DomainsOnly(text, false)
	domains = textsubs.Resolve(domains)
	secrets.CapturedDomains = domains

	secrets.CapturedURLs = grabURLs(text)
	secrets.SourceUrl = sourceUrl

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
