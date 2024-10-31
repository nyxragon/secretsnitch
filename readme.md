[![Golang](https://img.shields.io/badge/Golang-fff.svg?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-GPLv3-purple?style=flat-square&logo=libreoffice)](LICENSE)
[![Latest Version](https://img.shields.io/github/v/tag/0x4f53/secretsnitch?label=Version&style=flat-square&logo=semver)](https://github.com/0x4f53/secretsnitch/releases)

Presented at 

[![BlackHat MEA 2024](https://img.shields.io/badge/BlackHat%20MEA%202024-222.svg?style=flat-square&logo=redhat)](https://blackhatmea.com/blackhat-arsenal)

<img src = "media/logo.png" alt = "Secretsnitch logo" width = "60dp">

# Secretsnitch

A lightning-fast, modular secret scanner and endpoint extractor in Golang! 

- Concurrent Scanning: fast and efficient scanning with thousands of Goroutines at once

- Modular Design: Supports GitHub, GitLab, Phishtank and random web scraping via flags

- Efficient Network usage: reduced network usage with caching and instant output logging

- Comprehensive Secret checks: huge signature list for variable names, secret strings etc combined with metadata and entropy scoring

- User-Friendly: Designed for ease of use by pentesters, bounty hunters, or enterprise users via simple command-line execution

- Community-Driven Signatures: sourced from Google searches, ChatGPT, and open-source lists like GitGuardian.

- Easy Contribution: Find a missing secret or blacklist regular expression? Simply make a pull request by following the [contributing.md] guide!

## How it works

### Examples 

#### GitHub commits in a range

reserved

#### Stolen Phishing keys

Running secretsnitch with the following command:

```
go build
./secretsnitch --phishtank
```

Grabs the latest URL archive from the Phishtank API. It then begins downloading and scraping all the pages from Phishtank first, post which the secret analysis begins.


#### Single URLs

Say you have the following page:

https://0x4f.in

This page has a hardcoded OpenAI API Key in a Javascript file that it calls, named `security.js`. Simply running secretsnitch with the following command:

```
go build
./secretsnitch --url=https://0x4f.in --recursions=1
```

### Modules

reserved

### Caching

reserved


### Tunables

Secretsnitch is extremely tunable via for different use cases, whether its the worker count to prevent slowdowns on older devices, or the output destination for logshipping via tools like Filebeat.

Tunables available:

- output: Save scan output to a custom location. Directories and subdirectories will be created if they don't exist.

- workers: The amount of workers that can run each operation concurrently. This should be set according to hardware factors like CPU threads, rate-limits etc. When you're bruteforcing a large list of URLs or a directory on a powerful server, set this number to a high number.

- recursions: Crawl URLs inside files, then crawl the URLs inside those URLs, then the URLs in the URLs in the URLs, then the URLs in the URLs in the URLs in the URLs, then... you get the point.

- retries: Give up after trying so many times. Useful if the destination is misbehaving or crashing.

- secrets-optional: Display other data such as URLs and domains even if there are no secrets. Useful for asset extraction.

- selenium: If a site uses client-side rendering (CSR), you can use the Selenium plugin to have the Javascript be rendered first, then extract secrets from it. Please note that Docker needs to be installed for this to work.

### Scanning

reserved

#### Tokenization

<img src = "media/secretsnitch_tokenizer.drawio.png" alt = "Tokenizer workflow">

When a file containing code is passed to the tool, it uses tokenization techniques via in-built regular expressions, string splitting and so on. 

These techniques are tested and optimized for languages commonly used with backend development such as - Javascript, 
- Golang, 
- Bash, 
- Python, 
- Java 
- etc. 

There is also support for common structured file formats such as 
- JSON, 
- env, 
- XML, 
- HTML,
- etc.

#### Parsing

<img src = "media/secretsnitch_variable_scanner.drawio.png" alt = "Scanner workflow">

Secretsnitch looks for two classifications of secrets:

1. Single secrets: These include
    - API Keys
    - Password strings
    - URLs with leaked authentication (such as `mysql://` connection strings)

2. Secret files: These include
    - SSH Private Keys
    - `.pem` files

### Detection

reserved

### Contribution

reserved

### Metadata

reserved

## Troubleshooting

### GitHub rate limits

If your worker count is above GitHub's permitted public API limits, blasting multiple queries will result in an error `429` and a rate-limit. To prevent this, simply set the `workers` flag to 100 or a lower number. This trick also works with the URL list option if the source URLs have rate-limiting enable.d

### Tool stops instantly

Sometimes, the tool just stops as soon as it is started. This is due to a bug with the concurrency. Simply re-run the tool a few times if this happens.

### Selenium mode

If you receive a message like the one below on Linux

```
Error creating container: Post "http://unix.sock/containers/create?": dial unix /var/run/docker.sock: connect: permission denied
```

Simply run

```
sudo usermod -aG docker $USER
newgrp docker
```

And try again. 

For Windows, try the following

1. Open Docker Desktop Settings
2. Right-click the Docker icon in the system tray and select Settings.
3. Enable the TCP Endpoint
4. Go to the General or Resources > Advanced settings (the exact menu depends on the Docker Desktop version).
5. Look for an option to Expose daemon on tcp://localhost:2375 without TLS.
6. Enable this option if you’re okay with using an insecure endpoint, or set up certificates for localhost:2376 if you prefer TLS.
7. Go to the `dockerSelenium.go` file and replace `unix:///var/run/docker.sock` with `http://localhost:2375`.

If you can successfully build the docker image manually but can't trigger it via secretsnitch, try running the tool as superuser via `sudo`.

---

Copyright (c) 2024 Owais Shaikh (https://0x4f.in)