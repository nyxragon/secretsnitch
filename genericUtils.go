package main

import (
	"crypto/md5"
	"fmt"
	"net/url"
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
