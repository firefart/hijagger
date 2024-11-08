package main

import (
	"errors"
	"fmt"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"github.com/miekg/dns"
)

func (a *app) domainMX(domain string) ([]string, error) {
	if res := a.getAlreadyCheckedMX(domain); res != nil {
		return res, nil
	}

	mx, err := a.dnsClient.Query(domain, dns.TypeMX)
	if err != nil {
		return nil, fmt.Errorf("Error when resolving MX for %s: %w", domain, err)
	}
	a.setalreadyCheckedMX(domain, mx.MX)
	return mx.MX, nil
}

func (a *app) domainNS(domain string) ([]string, error) {
	// make sure we check the root domain here and not a subdomain
	rootDomain, err := getRootDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("could not get root domain for %s: %w", domain, err)
	}

	if res := a.getAlreadyCheckedNS(rootDomain); res != nil {
		return res, nil
	}

	ns, err := a.dnsClient.Query(rootDomain, dns.TypeNS)
	if err != nil {
		return nil, fmt.Errorf("Error when resolving %s: %w", rootDomain, err)
	}
	a.setalreadyCheckedNS(rootDomain, ns.NS)
	return ns.NS, nil
}

func (a *app) domainResolve(domain string) ([]string, error) {
	if res := a.getAlreadyCheckedDomain(domain); res != nil {
		return res, nil
	}

	ns, err := a.dnsClient.Resolve(domain)
	if err != nil {
		return nil, fmt.Errorf("Error when resolving %s: %w", domain, err)
	}

	var result []string
	result = append(result, ns.A...)
	result = append(result, ns.AAAA...)

	a.setalreadyCheckedDomain(domain, result)
	return result, nil
}

func (a *app) getAlreadyCheckedNS(domain string) []string {
	a.muNS.RLock()
	defer a.muNS.RUnlock()
	if result, ok := a.alreadyCheckedNS[domain]; ok {
		return result
	}

	return nil
}

func (a *app) getAlreadyCheckedMX(domain string) []string {
	a.muMX.RLock()
	defer a.muMX.RUnlock()
	if result, ok := a.alreadyCheckedMX[domain]; ok {
		return result
	}

	return nil
}

func (a *app) getAlreadyCheckedDomain(domain string) []string {
	a.muRS.RLock()
	defer a.muRS.RUnlock()
	if result, ok := a.alreadyCheckedRecords[domain]; ok {
		return result
	}

	return nil
}

func (a *app) getAlreadyCheckedWhois(domain string) *whoisparser.WhoisInfo {
	a.muWhois.RLock()
	defer a.muWhois.RUnlock()
	if result, ok := a.alreadyCheckedWhois[domain]; ok {
		return result
	}

	return nil
}

func (a *app) getAlreadyCheckedWhoisError(domain string) error {
	a.muWhoisError.RLock()
	defer a.muWhoisError.RUnlock()
	if result, ok := a.alreadyCheckedWhoisError[domain]; ok {
		return result
	}

	return nil
}

func (a *app) setalreadyCheckedNS(domain string, result []string) {
	a.muNS.Lock()
	defer a.muNS.Unlock()
	a.alreadyCheckedNS[domain] = result
}

func (a *app) setalreadyCheckedMX(domain string, result []string) {
	a.muMX.Lock()
	defer a.muMX.Unlock()
	a.alreadyCheckedMX[domain] = result
}

func (a *app) setalreadyCheckedDomain(domain string, result []string) {
	a.muRS.Lock()
	defer a.muRS.Unlock()
	a.alreadyCheckedRecords[domain] = result
}

func (a *app) setalreadyCheckedWhois(domain string, result *whoisparser.WhoisInfo) {
	a.muWhois.Lock()
	defer a.muWhois.Unlock()
	a.alreadyCheckedWhois[domain] = result
}

func (a *app) setalreadyCheckedWhoisError(domain string, result error) {
	a.muWhoisError.Lock()
	defer a.muWhoisError.Unlock()
	a.alreadyCheckedWhoisError[domain] = result
}

func (a *app) whois(domain string) (*whoisparser.WhoisInfo, error) {
	// make sure we always check the root domain here and not a subdomain
	rootDomain, err := getRootDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("could not get root domain for %s: %w", domain, err)
	}

	if res := a.getAlreadyCheckedWhois(rootDomain); res != nil {
		return res, nil
	}

	// if we got a whois error before, return this so we don't query
	// the server over and over again
	if res := a.getAlreadyCheckedWhoisError(rootDomain); res != nil {
		return nil, &WhoisError{err: err, repeatedError: true}
	}

	resp, err := whois.Whois(rootDomain)
	if err != nil {
		a.setalreadyCheckedWhoisError(rootDomain, err)
		return nil, err
	}
	parsed, err := whoisparser.Parse(resp)
	if err != nil {
		if errors.Is(err, whoisparser.ErrNotFoundDomain) {
			// domain is free, also cache the free result
			a.setalreadyCheckedWhois(rootDomain, nil)
			return nil, nil
		}
		a.setalreadyCheckedWhoisError(rootDomain, err)
		return nil, err
	}
	a.setalreadyCheckedWhois(rootDomain, &parsed)

	return &parsed, nil
}
