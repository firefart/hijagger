package main

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"golang.org/x/exp/maps"
)

func (a *app) pypiGetAllPackageNames() ([]string, error) {
	html, err := a.httpRequest("https://pypi.org/simple/")
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`<a href="/simple/(.+?)/"`)
	matches := re.FindAllSubmatch(html, -1)
	if matches == nil {
		return nil, fmt.Errorf("could not get package names")
	}
	packages := make([]string, len(matches))
	for i := range matches {
		packages[i] = string(matches[i][1])
	}
	return packages, nil
}

type pypiPackageJSON struct {
	Info struct {
		AuthorEmail     string `json:"author_email"`
		MaintainerEmail string `json:"maintainer_email"`
	} `json:"info"`
}

func (a *app) pypiGetPackageMaintainer(name string) ([]string, error) {
	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", name)
	resp, err := a.httpRequest(url)
	if err != nil {
		return nil, err
	}
	var data pypiPackageJSON
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("error on json unmarshal for %s: %w", url, err)
	}

	maintainers := make(map[string]struct{})

	data.Info.AuthorEmail = strings.TrimSpace(data.Info.AuthorEmail)
	data.Info.MaintainerEmail = strings.TrimSpace(data.Info.MaintainerEmail)

	if data.Info.AuthorEmail != "" && data.Info.AuthorEmail != "UNKNOWN" {
		email, err := mail.ParseAddress(data.Info.AuthorEmail)
		if err != nil {
			a.log.Debugf("invalid author %s: %v", data.Info.AuthorEmail, err)
		} else {
			maintainers[email.Address] = struct{}{}
		}
	}
	if data.Info.MaintainerEmail != "" && data.Info.MaintainerEmail != "UNKNOWN" {
		email, err := mail.ParseAddress(data.Info.MaintainerEmail)
		if err != nil {
			a.log.Debugf("invalid maintainer %s: %v", data.Info.MaintainerEmail, err)
		} else {
			maintainers[email.Address] = struct{}{}
		}
	}

	return maps.Keys(maintainers), nil
}

func getPypiLink(name string) string {
	return fmt.Sprintf("https://pypi.org/project/%s/", name)
}
