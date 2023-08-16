package main

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"os"

	"golang.org/x/exp/maps"
)

type npmAll struct {
	TotalRows int64 `json:"totalrows"`
	Offset    int64 `json:"offset"`
	Rows      []struct {
		ID    string `json:"id"`
		Key   string `json:"key"`
		Value struct {
			Rev string `json:"rev"`
		} `json:"value"`
	} `json:"rows"`
}

type npmPackageJSON struct {
	Name    string `json:"name"`
	NPMUser struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"_npmUser"`
	Maintainers []struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"maintainers"`
}

type npmDownloadJSON struct {
	Downloads int64  `json:"downloads"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Package   string `json:"package"`
}

func (a *app) npmGetPackageMaintainer(name string) ([]string, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", name)
	resp, err := a.httpRequest(url)
	if err != nil {
		return nil, err
	}
	var data npmPackageJSON
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("error on json unmarshal for %s: %w", url, err)
	}

	maintainers := make(map[string]struct{})

	if data.NPMUser.Email != "" {
		email, err := mail.ParseAddress(data.NPMUser.Email)
		if err != nil {
			a.log.Debugf("invalid maintainer %s: %v", data.NPMUser.Email, err)
		} else {
			maintainers[email.Address] = struct{}{}
		}
	}
	for _, email := range data.Maintainers {
		mail, err := mail.ParseAddress(email.Email)
		if err != nil {
			a.log.Debugf("invalid maintainer %s: %v", email.Email, err)
		} else {
			maintainers[mail.Address] = struct{}{}
		}
	}

	return maps.Keys(maintainers), nil
}

func (a *app) npmGetAllPackageNames(local string) ([]string, error) {
	resp, err := os.ReadFile(local)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("the local index was not found. Please download before running this tool from https://skimdb.npmjs.com/registry/_all_docs")
		}
		return nil, err
	}
	var data npmAll
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("error on json unmarshal for all package names: %w", err)
	}

	packages := make([]string, len(data.Rows))
	for i := range data.Rows {
		packages[i] = data.Rows[i].Key
	}

	return packages, nil
}

func (a *app) npmGetDownloadCountLastYear(packageName string) (int64, error) {
	url := fmt.Sprintf("https://api.npmjs.org/downloads/point/last-year/%s", packageName)
	resp, err := a.httpRequest(url)
	if err != nil {
		return -1, fmt.Errorf("could not get download count for %s: %w", packageName, err)
	}

	var data npmDownloadJSON
	if err := json.Unmarshal(resp, &data); err != nil {
		return -1, fmt.Errorf("error on json unmarshal for %s: %w", url, err)
	}

	return data.Downloads, nil
}

func getNPMLink(name string) string {
	return fmt.Sprintf("https://www.npmjs.com/package/%s", name)
}
