package main

import (
	"log"
	"os"
	"os/exec"

	githubPatches "github.com/0x4f53/github-patches"
	gitlabPatches "github.com/0x4f53/gitlab-patches"
)

var signatures []Signature

func main() {

	logo()

	err := makeDir(cacheDir)
	if err != nil {
		log.Println(err)
	}

	setFlags()

	signatures = readSignatures()

	if *urlList != "" {
		urls, _ := readLines(*urlList)
		successfulUrls := fetchFromUrlList(urls)
		ScanFiles(successfulUrls)
		return
	}

	if *URL != "" {
		var successfulUrls []string
		if *selenium {
			if !checkDockerInstalled() {
				log.Fatalf("Please install Docker to use Selenium mode!")
			}
			if !checkImageBuilt() {
				log.Println("Attempting to build Selenium testing image from Dockerfile...")
				err := exec.Command("docker", "build", "-t", "selenium-integration", ".").Run()
				if err != nil {
					log.Fatalf("Failed to build Docker image: %v", err)
				}
			}
			successfulUrls = []string{scrapeWithSelenium(*URL)}
		} else {
			successfulUrls = fetchFromUrlList([]string{*URL})
		}
		ScanFiles(successfulUrls)
		return
	}

	if *directory != "" {
		files, err := getAllFiles(*directory)
		if err != nil {
			log.Fatalf("Error getting files from directory: %v", err)
		}
		ScanFiles(files)
		return
	}

	if *file != "" {
		ScanFiles([]string{*file})
		return
	}

	if *github {
		githubPatches.GetCommitsInRange(githubPatches.GithubCacheDir, *from, *to, false)
		chunks, err := listFiles(githubPatches.GithubCacheDir)
		if err != nil {
			log.Fatalf("Error listing GitHub cache files: %v", err)
		}

		var patches []string

		for _, chunk := range chunks {
			events, err := githubPatches.ParseGitHubCommits(githubPatches.GithubCacheDir + chunk)
			if err != nil {
				log.Printf("Error parsing GitHub commits from %s: %v", chunk, err)
				continue
			}

			for _, event := range events {
				for _, commit := range event.Payload.Commits {
					patches = append(patches, commit.PatchURL)
				}
			}

		}

		successfulUrls := fetchFromUrlList(patches)
		ScanFiles(successfulUrls)
		defer os.RemoveAll(githubPatches.GithubCacheDir)
		return
	}

	if *gitlab {
		commitData := gitlabPatches.GetGitlabCommits(100, 100)

		var patches []string
		for _, patch := range commitData {
			patches = append(patches, patch.CommitPatchURL)
		}

		successfulUrls := fetchFromUrlList(patches)
		ScanFiles(successfulUrls)
		defer os.RemoveAll(gitlabPatches.GitlabCacheDir)
		return
	}

	if *githubGists {
		gistData := githubPatches.GetLast100Gists()
		parsedGists, err := githubPatches.ParseGistData(gistData)
		if err != nil {
			log.Fatalf("Error parsing GitHub gists: %v", err)
		}

		var gists []string
		for _, gist := range parsedGists {
			gists = append(gists, gist.RawURL)
		}

		successfulUrls := fetchFromUrlList(gists)
		ScanFiles(successfulUrls)
		return
	}

	if *phishtank {
		savePhishtankDataset()
		urls, err := readLines(phishtankURLCache)
		if err != nil {
			log.Fatalf("Error reading phishtank URLs: %v", err)
		}
		successfulUrls := fetchFromUrlList(urls)
		ScanFiles(successfulUrls)
		return
	}

}
