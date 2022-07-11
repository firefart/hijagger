package main

import (
	"fmt"

	"github.com/jpillora/go-tld"
)

func getRootDomain(domain string) (string, error) {
	// add a fake schema so the parser works
	u, err := tld.Parse("http://" + domain)
	if err != nil {
		return "", err
	}
	rootDomain := fmt.Sprintf("%s.%s", u.Domain, u.TLD)
	return rootDomain, nil
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
