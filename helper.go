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
