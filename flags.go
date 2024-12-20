package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/pflag"
)

var (
	// directory module
	directory *string

	// file module
	file *string

	// url module
	URL     *string
	urlList *string

	// github module
	github      *bool
	from        *string
	to          *string
	githubGists *bool

	// gitlab module
	gitlab *bool

	// phishtank module
	phishtank *bool

	// output file name
	outputFile *string

	// maximum number of permitted workers
	maxWorkers *int

	// maximum number of page recursions
	maxRecursions *int

	// maximum number of page recursions
	maxRetries *int

	// Use selenium via Docker
	selenium *bool

	// Maximimum file size to scan for secrets
	maxFileSize = 20 * 1024 * 1024
)

func logo() {
	fmt.Println("\nSecretsnitch - A lightning-fast secret scanner in Golang!")
	fmt.Println("https://github.com/0x4f53/secretsnitch")
	fmt.Println("Copyright © 2024 Owais Shaikh")
	fmt.Println()
}

func customUsage() {
	logo()
	fmt.Println("")
	fmt.Fprintf(os.Stderr, "Usage:\n%s [input options] [output options]\n", os.Args[0])
	fmt.Println("")
	fmt.Println("Input (pick at least one):")
	fmt.Println("")
	fmt.Println("  --github                  Scan public GitHub commits from the past hour")
	fmt.Println("    --from                  (optional) Timestamp to start from (format: 2006-01-02-15)")
	fmt.Println("    --to                    (optional) Timestamp to stop at (format: 2006-01-02-15)")
	fmt.Println("")
	fmt.Println("  --github-gists            Scan the last 100 public GitHub Gists")
	fmt.Println("")
	fmt.Println("  --gitlab                  Scan the last 100 public GitLab commits")
	fmt.Println("")
	fmt.Println("  --phishtank               Scan reported phishtank.org URLs from the past day")
	fmt.Println("")
	fmt.Println("  --url=<http://url>        A single URL to scan")
	fmt.Println("  --urlList=<file>          A line-separated file containing a list of URLs to scan for secrets")
	fmt.Println("")
	fmt.Println("  --directory=<directory/>  Scan an entire directory")
	fmt.Println("  --file=<file.js>          Scan a file")
	fmt.Println("")
	fmt.Println("Optional arguments:")
	fmt.Println("")
	fmt.Println("  --output                  Save scan output to a custom location")
	fmt.Println("")
	fmt.Println("  --workers                 Maximum number of workers to use (default: " + strconv.Itoa(*maxWorkers) + ")")
	fmt.Println("")
	fmt.Println("  --recursions=<number>     Crawl URLs and hyperlinks inside targets (default: " + strconv.Itoa(*maxRecursions) + ")")
	fmt.Println("")
	fmt.Println("  --retries=<number>        Maximum retries before giving up (default: " + strconv.Itoa(*maxRetries) + ")")
	fmt.Println("")
	fmt.Println("  --selenium                Scrape page using Selenium. This helps with pages that run client-side Javascript (note: this is slower)")
	fmt.Println("")
}

func setFlags() {
	github = pflag.Bool("github", false, "")
	from = pflag.String("from", "", "")
	to = pflag.String("to", "", "")

	githubGists = pflag.Bool("github-gists", false, "")

	URL = pflag.String("url", "", "")
	urlList = pflag.String("urlList", "", "")
	directory = pflag.String("directory", "", "")
	file = pflag.String("file", "", "")

	gitlab = pflag.Bool("gitlab", false, "")

	phishtank = pflag.Bool("phishtank", false, "")

	maxWorkers = pflag.Int("workers", 5000, "")
	maxRecursions = pflag.Int("recursions", 0, "")
	maxRetries = pflag.Int("retries", 3, "")
	selenium = pflag.Bool("selenium", false, "")
	outputFile = pflag.String("output", defaultOutputDir, "")

	pflag.Usage = customUsage
	pflag.Parse()

	if *maxWorkers < 2 {
		//pflag.Usage()
		fmt.Println("Please use at least 2 workers for efficient concurrency.")
		os.Exit(-1)
	}

	if !*github && !*gitlab && !*phishtank && *URL == "" && *urlList == "" && *directory == "" && *file == "" && !*githubGists {
		pflag.Usage()
		fmt.Println("Come on, you'll have to pick some option!")
		os.Exit(-1)
	}

}
