package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/js"
	"gopkg.in/yaml.v3"
	"mvdan.cc/xurls"
)

var (
	namesBlacklistFile  = "blacklist/names.yaml"
	valuesBlacklistFile = "blacklist/values.yaml"
)

func containsBlacklistedNames(text string) bool {

	data, err := os.ReadFile(namesBlacklistFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	var blacklist []string
	err = yaml.Unmarshal(data, &blacklist)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	for _, item := range blacklist {
		re := regexp.MustCompile(item)
		if len(re.FindAllString(text, 1)) > 0 {
			return true
		}
	}

	return false

}

func containsBlacklistedValues(text string) bool {

	data, err := os.ReadFile(valuesBlacklistFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	var blacklist []string
	err = yaml.Unmarshal(data, &blacklist)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	for _, item := range blacklist {
		re := regexp.MustCompile(item)
		if len(re.FindAllString(text, 1)) > 0 {
			return true
		}
	}

	return false

}

type VariableData struct {
	Name     string
	Operator string
	Value    string
}

func extractKeyValuePairs(text string) ([]VariableData, error) {

	// Initialize a map to hold the key-value pairs
	var assignmentPairs []VariableData

	// Scan the file line by line
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()

		jsonAndDicts := parseJsonAndDict(line)
		assignmentPairs = append(assignmentPairs, jsonAndDicts...)

		txtVariables := parseTxt(line)
		assignmentPairs = append(assignmentPairs, txtVariables...)

		xmlTags := parseXmlTags(line)
		assignmentPairs = append(assignmentPairs, xmlTags)

		equals := parseEquals(line)
		assignmentPairs = append(assignmentPairs, equals...)

		golangEquals := parseGolangEquals(line)
		assignmentPairs = append(assignmentPairs, golangEquals...)

		phpArrowEquals := parsePhpArrow(line)
		assignmentPairs = append(assignmentPairs, phpArrowEquals...)

		phpClosureArrowEquals := parsePhpClosureArrow(line)
		assignmentPairs = append(assignmentPairs, phpClosureArrowEquals...)

		colons := parseColons(line)
		assignmentPairs = append(assignmentPairs, colons...)

	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	assignmentPairs = removeEmptyAndDuplicateData(assignmentPairs)

	return assignmentPairs, nil
}

// Match static colon key-value pairs
func parseColons(line string) []VariableData {
	var parsedData []VariableData

	reColons := regexp.MustCompile(`["']?(\w+)["']?\s*:\s*([^,\n]+)`)

	matches := reColons.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		varData := VariableData{
			Name:     match[1],
			Operator: ":",
			Value:    match[2],
		}

		parsedData = append(parsedData, varData)
	}

	return parsedData
}

// Match static equals key-value pairs from places like CLI commands
func parseEquals(line string) []VariableData {
	var parsedData []VariableData

	reEquals := regexp.MustCompile(`(\S+)\s*=\s*(\S+)`)

	matches := reEquals.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		varData := VariableData{
			Name:     match[1],
			Operator: "=",
			Value:    match[2],
		}
		parsedData = append(parsedData, varData)
	}

	return parsedData
}

// Match static arrow key-value pairs for languages like php
func parsePhpArrow(line string) []VariableData {
	var parsedData []VariableData

	reArrow := regexp.MustCompile(`(\S+)\s*->\s*(\S+)`)

	matches := reArrow.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		varData := VariableData{
			Name:     match[1],
			Operator: "->",
			Value:    match[2],
		}
		parsedData = append(parsedData, varData)
	}

	return parsedData
}

// Match static closure arrow key-value pairs for languages like php
func parsePhpClosureArrow(line string) []VariableData {
	var parsedData []VariableData

	reArrow := regexp.MustCompile(`(\S+)\s*=>\s*(\S+)`)

	matches := reArrow.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		varData := VariableData{
			Name:     match[1],
			Operator: "=>",
			Value:    match[2],
		}
		parsedData = append(parsedData, varData)
	}

	return parsedData
}

// Match static equals key-value pairs from places like CLI commands
func parseGolangEquals(line string) []VariableData {
	var parsedData []VariableData

	reEquals := regexp.MustCompile(`(\w+)\s*:=\s*(\S+)`)

	matches := reEquals.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		varData := VariableData{
			Name:     match[1],
			Operator: ":=",
			Value:    match[2],
		}
		parsedData = append(parsedData, varData)
	}

	return parsedData
}

// Match XML tags
func parseXmlTags(line string) VariableData {
	var parsedData VariableData
	var reXML = regexp.MustCompile(`<([^\/>]+)[\/]*>.*<(/[^\/>]+)[\/]*>`)
	if matches := reXML.FindStringSubmatch(line); matches != nil {
		value := strings.Replace(matches[0], matches[1], "", -1)
		value = strings.Replace(value, "<>", "", -1)
		value = strings.Replace(value, "</>", "", -1)

		value = strings.Trim(value, "\"")
		value = strings.Trim(value, "'")
		value = strings.Trim(value, "`")

		parsedData = VariableData{
			Name:     matches[1],
			Operator: "<>...</>",
			Value:    value,
		}
	}
	return parsedData
}

// Match static JSON and Dict key-value pairs
func parseJsonAndDict(text string) []VariableData {
	pattern := `["']?([^"':\s]+)["']?\s*:\s*["']?([^"'\n]+)["']?`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(text, -1)
	var keyValuePairs []VariableData
	for _, match := range matches {
		if len(match) == 3 {
			parsedData := VariableData{
				Name:     match[1],
				Operator: ":",
				Value:    match[2],
			}
			if parsedData.Value != "" {

				parsedData.Value = strings.Trim(parsedData.Value, "\"")
				parsedData.Value = strings.Trim(parsedData.Value, "'")
				parsedData.Value = strings.Trim(parsedData.Value, "`")

				keyValuePairs = append(keyValuePairs, parsedData)
			}
		}
	}

	return keyValuePairs
}

// Match static text key-value pairs
func parseTxt(text string) []VariableData {
	pattern := `["']?([^"':\s]+)["']?\s*-\s*["']?([^"'\n ]+)["']?`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(text, -1)
	var keyValuePairs []VariableData
	for _, match := range matches {
		if len(match) == 3 {
			parsedData := VariableData{
				Name:     match[1],
				Operator: ":",
				Value:    match[2],
			}
			if parsedData.Value != "" {

				parsedData.Value = strings.Trim(parsedData.Value, "\"")
				parsedData.Value = strings.Trim(parsedData.Value, "'")
				parsedData.Value = strings.Trim(parsedData.Value, "`")

				keyValuePairs = append(keyValuePairs, parsedData)
			}
		}
	}

	return keyValuePairs
}

func removeEmptyAndDuplicateData(input []VariableData) []VariableData {
	var parsedData []VariableData
	exists := func(item VariableData) bool {
		for _, existingItem := range parsedData {
			if existingItem.Value == item.Value && existingItem.Name == item.Name {
				return true
			}
		}
		return false
	}

	for _, item := range input {
		if len(item.Value) > 3 && !exists(item) {
			parsedData = append(parsedData, item)
		}
	}

	return parsedData
}

func prettifyJS(text string) string {
	m := minify.New()
	m.AddFunc("text/javascript", js.Minify)
	var prettyJS strings.Builder
	err := m.Minify("text/javascript", &prettyJS, strings.NewReader(text))
	if err != nil {
		fmt.Println("Error minifying:", err)
		return ""
	}
	return prettyJS.String()
}

func extractEmails(text string) []string {
	emailRegex := `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`
	re := regexp.MustCompile(emailRegex)
	emails := re.FindAllString(text, -1)
	return emails
}

func extractURLs(text string) []string {
	var captured []string
	sourceUrl := grabSourceUrl(text)
	baseUrl, _ := baseURL(sourceUrl)

	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl += "/"
	}

	rxUrls := xurls.Relaxed.FindAllString(text, -1)
	captured = append(captured, rxUrls...)

	var splitText []string
	splitText = append(splitText, strings.Split(text, "{")...)
	splitText = append(splitText, strings.Split(text, ",")...)
	splitText = append(splitText, strings.Split(text, "\n")...)
	splitText = removeDuplicates(splitText)

	textChunks := make(chan string, len(splitText))
	results := make(chan []string, len(splitText))

	var wg sync.WaitGroup

	for i := 0; i < *maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var workerCaptured []string
			for line := range textChunks {
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

					workerCaptured = append(workerCaptured, resource)
				}
			}
			results <- workerCaptured
		}()
	}

	for _, line := range splitText {
		textChunks <- line
	}
	close(textChunks)

	go func() {
		wg.Wait()
		close(results)
	}()

	for workerCaptured := range results {
		captured = append(captured, workerCaptured...)
	}

	var urls []string
	for _, url := range captured {
		if strings.Contains(url, "://") && strings.Contains(url, ".") && !strings.Contains(url, "'") && url != sourceUrl {
			urls = append(urls, url)
		}
	}

	return removeDuplicates(urls)
}
