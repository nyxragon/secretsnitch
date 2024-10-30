package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	blacklistFile = "blacklist.yaml"
)

func containsBlacklisted(text string) bool {

	data, err := os.ReadFile(blacklistFile)
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

		xmlTags := parseXmlTags(line)
		assignmentPairs = append(assignmentPairs, xmlTags)

		colonsAndEquals := parseColonsAndEquals(line)
		assignmentPairs = append(assignmentPairs, colonsAndEquals)
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	assignmentPairs = removeEmptyAndDuplicateData(assignmentPairs)

	return assignmentPairs, nil
}

// Match static colon/equals key-value pairs
func parseColonsAndEquals(line string) VariableData {
	var parsedData VariableData
	var reStaticColon = regexp.MustCompile(`(\w+\){0,1}\]{0,1}) {0,1}(:=|=|:) {0,1}(\S+)`)

	if matches := reStaticColon.FindStringSubmatch(line); matches != nil {

		value := matches[3]
		value = strings.Trim(value, "\"")
		value = strings.Trim(value, "'")
		value = strings.Trim(value, "`")

		parsedData = VariableData{
			Name:     matches[1],
			Operator: matches[2],
			Value:    value,
		}
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

// Match static JSON and Dict key-value pairs
func parsePrivateKeys(text string) []VariableData {
	pattern := `(-----BEGIN (?:[DR]SA|EC|PGP|OPENSSH)?\s?PRIVATE KEY(?: BLOCK)?-----)[A-Za-z0-9+\/=\s]{128,}(-----END (?:[DR]SA|EC|PGP|OPENSSH)?\s?PRIVATE KEY(?: BLOCK)?-----)`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(text, -1)

	var keyValuePairs []VariableData
	for _, match := range matches {
		if len(match) == 3 {
			parsedData := VariableData{
				Name:     match[1],
				Operator: "-----",
				Value:    match[0],
			}
			if parsedData.Value != "" {
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
