package main

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

func substringBeforeFirst(input string, delimiter string) string {
	index := strings.Index(input, delimiter)
	if index == -1 {
		return ""
	}
	return strings.TrimSpace(input[:index])
}

func md5Hash(text string) string {
	data := []byte(text)
	return fmt.Sprintf("%x", md5.Sum(data))
}

func removeDuplicates(elements []string) []string {
	seen := make(map[string]struct{})
	result := []string{}

	for _, element := range elements {
		if _, found := seen[element]; !found {
			seen[element] = struct{}{}
			result = append(result, element)
		}
	}

	return result
}

func sliceContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func baseURL(inputURL string) (string, error) {
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return "", err
	}

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
	return baseURL, nil
}

func removeFromSlice(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func grabSourceUrl(text string) string {
	lines := strings.Split(text, "\n")

	if len(lines) < 2 {
		return ""
	}

	if strings.TrimSpace(lines[1]) == "---" {
		return lines[0]
	}

	return ""
}

func containsSecret(secrets []SecretData, target SecretData) bool {
	for _, s := range secrets {
		if s.Provider == target.Provider &&
			s.ServiceName == target.ServiceName &&
			s.Variable == target.Variable &&
			s.Secret == target.Secret &&
			s.TsallisEntropy == target.TsallisEntropy &&
			equalTags(s.Tags, target.Tags) {
			return true
		}
	}
	return false
}

func findPosition(text, substring, lineString string) (line, column int) {
	lines := strings.Split(text, "\n")

	// If lineString is not empty, find its position in the text
	if lineString != "" {
		for i, lineText := range lines {
			lineStart := strings.Index(lineText, lineString)
			if lineStart != -1 {
				// Now find the position of the substring within this line
				substringStart := strings.Index(lineText[lineStart:], substring)
				if substringStart != -1 {
					return i + 1, lineStart + substringStart + 1 // Return 1-based index
				}
				return i + 1, -1 // If substring is not found in this line
			}
		}
	}

	// If lineString is empty or not found, fall back to finding the substring
	for i, lineText := range lines {
		col := strings.Index(lineText, substring)
		if col != -1 {
			return i + 1, col + 1 // Return 1-based index of substring
		}
	}

	return -1, -1 // Return -1 if neither lineString nor substring is found
}

func equalTags(tags1, tags2 []string) bool {
	if len(tags1) != len(tags2) {
		return false
	}
	for i, tag := range tags1 {
		if tag != tags2[i] {
			return false
		}
	}
	return true
}

func printSecret(secret SecretData, sourceUrl string, cacheFile string) {

	fmt.Printf(`
SECRET DETECTED:
	- Type:            %s
	- Variable Name:   %s
	- Value:           %s
	- Position:        %s
	- Source:          %s
	- Cached Location: %s
	- Tags:            %s
	- Tsallis Entropy: %f
`,
		secret.Provider+" "+secret.ServiceName,
		secret.Variable,
		secret.Secret,
		secret.Position,
		sourceUrl,
		cacheFile,
		secret.Tags,
		secret.TsallisEntropy,
	)

}

func truncateGitBinaryData(text string) string {
	patchRegex := regexp.MustCompile(`(?s)(diff --git.*?GIT binary .*?literal \d+.*?HcmV\?d00001)`)
	match := patchRegex.FindString(text)
	if match == "" {
		return ""
	}
	return match
}

func splitText(text string) []string {

	delimiters := map[rune]struct{}{
		'{':  {},
		',':  {},
		';':  {},
		'\n': {},
	}

	var result []string
	var builder strings.Builder

	for _, char := range text {
		if _, isDelimiter := delimiters[char]; isDelimiter {
			if builder.Len() > 0 {
				result = append(result, builder.String())
				builder.Reset()
			}
		} else {
			builder.WriteRune(char)
		}
	}

	if builder.Len() > 0 {
		result = append(result, builder.String())
	}

	return result
}
