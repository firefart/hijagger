# hijagger - check package registries for hijackable packages

This tool checks every maintainer from every package in the NPM and Python Pypi registry for unregistered domains or unregistered MX records on those domains. If a domain is unregistered you can grab the domain and initiate a password reset on the account if it has no 2 factor auth enabled. This enables you to hijack a package and do whatever you want with it.

**Please do not use it for illegal purposes, only use it to check packages and submit them to bug bounty programs.**

## NPM

I contacted the NPM security team about this but they are not interested in this kind of vulnerability.

Please also note that the returned maintainers returned from the API not always reflect the real maintainers but often you can get lucky.

### Usage

Download the package index first! This can take a long time as the server is extremely slow (takes more than 30 mins):

```
wget https://skimdb.npmjs.com/registry/_all_docs
```

After this simply run the tool with `./hijagger npm`. To see all options use the `--help` switch. The output is automatically saved to `output.txt` too. This tool will most probably run multiple days due to the high number of packages.

To easily find hits in the output, grep for `HIT`. The coloring is based on the number of downloads during the last year of the package.

The tool does a lot of DNS and whois requests so I suggest running this tool from a dedicated server to not risk having your private ip blocked.

## Pypi

This mode works exactly the same as the NPM mode, but the pypi api does not return the number of downloads for a package. This data is only available via google bigquery but this complicates things a lot so download counters and colors based on downloads are not implemented on Pypi packages.

### Usage

`./hijagger pypi`
