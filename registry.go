package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func (a *app) getPackageMaintainer(name string) ([]string, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", name)
	resp, err := a.httpRequest(url)
	if err != nil {
		return nil, err
	}
	var data packageJSON
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("error on json unmarshal for %s: %w", url, err)
	}

	maintainers := make(map[string]struct{})

	if data.NPMUser.Email != "" {
		maintainers[data.NPMUser.Email] = struct{}{}
	}
	for _, email := range data.Maintainers {
		maintainers[email.Email] = struct{}{}
	}

	return keysFromMap(maintainers), nil
}

func (a *app) getAllPackageNames(local string) ([]string, error) {
	resp, err := os.ReadFile(local)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("the local index was not found. Please download before running this tool from https://skimdb.npmjs.com/registry/_all_docs")
		}
		return nil, err
	}
	var data all
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("error on json unmarshal for all package names: %w", err)
	}

	packages := make([]string, len(data.Rows))
	for i := range data.Rows {
		packages[i] = data.Rows[i].Key
	}

	return packages, nil
}

func (a *app) getDownloadCountLastYear(packageName string) (int64, error) {
	url := fmt.Sprintf("https://api.npmjs.org/downloads/point/last-year/%s", packageName)
	resp, err := a.httpRequest(url)
	if err != nil {
		return -1, fmt.Errorf("could not get download count for %s: %w", packageName, err)
	}

	var data downloadJSON
	if err := json.Unmarshal(resp, &data); err != nil {
		return -1, fmt.Errorf("error on json unmarshal for %s: %w", url, err)
	}

	return data.Downloads, nil
}

func keysFromMap[V any](m map[string]V) []string {
	ret := make([]string, len(m))
	i := 0
	for k := range m {
		ret[i] = k
		i += 1
	}
	return ret
}

func getNPMLink(name string) string {
	return fmt.Sprintf("https://www.npmjs.com/package/%s", name)
}
