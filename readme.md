# Secretsnitch

A lightning-fast secret scanner in Golang!

## Features

### Fast and efficient

Secretsnitch is a rapid scanner for secrets, written in Golang. It utilizes concurrency via thousands of goroutines and downloads
and processes files, runs regular expression checks on them for secrets, grabs associated URLs and domains, tags them and performs tons of other operations concurrently.

Features like caching make sure files aren't re-downloaded. This speeds up the tool significantly while keeping network resource
consumption low.

### Modular

It supports tons of modules for popular online sources that may potentially contain thousands of secrets. This includes GitHub commits,
GitHub Gists, GitLab, Phishtank, random webpage scraping etc. More modules can be added this way, making the tool extremely dynamic.

### Smart and accurate

Secretsnitch doesn't just use regular expressions, it also scans files for common cloud provider strings, performs entropy checks on
captured strings, and gives you the variable/filename associated with a captured secret and gives you a precise indication on whether something may be an actual secret or not.

### Easy to use

Secretsnitch was designed to be easy to use, whether you are a pentester, bounty hunter or want to deploy it across your organization. The
tool can be run in singular commands as shown in the examples below.

### Community-driven

The signatures list is completely community-driven and is a combination of trial-and-error, Google searching, ChatGPT and from existing lists like that of GitGuardian. Pull requests for signature additions and corrections are welcome, and feel free to use these signatures in other cybersecurity tools you build.

### Troubleshooting

#### Selenium mode

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