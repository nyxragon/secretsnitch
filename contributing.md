# Contributing

Thank you for taking your time to contribute to secretsnitch. Let's get started.

YAML file signatures were a conscious design decision I made. In addition to looking pretty, they help me quickly comment out secrets or write comments explaining certain regular expression (regex) patterns. This is why maintaining the signatures is so easy.

### Adding new signatures to `signatures.yaml`

Signatures are stored in [signatures.yaml](signatures.yaml). If you'd like to add a pattern missing there, please follow the convention.

The format to be followed is as follows

```yaml
- Company: # The organization, such as OpenAI
  - Service: # The service, such as ChatGPT
    - Pattern ABC: <regex>  # A pattern, such as project secret key
    - Pattern DEF: <regex> # Another pattern, such as user secret key
    - XYZ Variable: <regex> # A variable pattern, such as OPENAI_API_KEY
```

There are two types of signatures 

- Variable name patterns: these are denoted by the use of the word "Variable" in their key. These are for variable names in code, where the secret may not have a recognizable pattern, but the name of the variable generally follows a pattern.
For example, passwords are random, but their field tends to be named "password" in most instances. Therefore, this is a Generic password variable pattern.

- Block patterns: these are denoted by the use of the word "Block" in their key. They contain patterns that commonly exist in large blocks of text, for example, private key files.
For example, `-----BEGIN OPENSSH PRIVATE KEY-----`.

- Secret patterns: these are patterns that are commonly used by secrets. Their keys don't consist of reserved words like the ones above. The more unique the regular expressions are, the fewer false positives they result in.
For example, `AIza...` for GCP keys, `AKIA` for AWS keys and so on.

**Note 1: Do not use the word 'variable' or 'block' in the key unless required since they're reserved and will be picked up as variable names and their values won't be searched.**

**Note 2: Always check the signatures list for existing entries thoroughly before adding new ones.** If they exist, modify them instead. If they don't feel free to add new ones in the exact convention specified above.

### Adding to `blacklist.yaml`

The format to be followed is as follows

```yaml
- <regex 1> # e.g.: data:image/
- <regex 2> # e.g.: sha(1|256|512)-
- <regex 3> # e.g.: ----- BEGIN OPENPGP PUBLIC KEY -----
```

Since the blacklist is also regex compatible, you can also specify patterns that blacklisted entries may follow. For example, `$YOUR_[\w]_KEY` is a good example of a blacklist pattern that will ignore variable substitutions in several shell scripts.

### New modules

Just like GitHub, GitLab and Phishtank, you can add more modules to the tool and have it scrape those sites in one command.

To do this, simply turn your proposed module into a Go package [and upload it as a module to pkg.go.dev](https://go.dev/doc/modules/publishing). Make sure it returns a list of the URLs pages you want to scrape as at least a slice of string data.

Then fork this tool and simply call the appropriate functions (such as `fetchFromUrlList()` and `ScanFiles()` to scrape the pages and scan them for secrets)

### Changes in logic

This tool has a few bugs. Please feel free to submit a pull request and I'll merge it happily.
