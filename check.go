package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// main function
func (a *app) checkPackage(registryType registryType, p string, checkExpired bool) error {
	a.log.Debugf("checking package %s", p)

	var maintainer []string
	var downloads int64
	var err error
	switch registryType {
	case RegistryNPM:
		maintainer, err = a.npmGetPackageMaintainer(p)
		if err != nil {
			return fmt.Errorf("could not get maintainer for package %s: %w", p, err)
		}

		downloads, err = a.npmGetDownloadCountLastYear(p)
		if err != nil {
			return fmt.Errorf("could not get downloadcount for package %s: %w", p, err)
		}
	case RegistryPypi:
		maintainer, err = a.pypiGetPackageMaintainer(p)
		if err != nil {
			return fmt.Errorf("could not get maintainer for package %s: %w", p, err)
		}

		// not returned by API, need to use google bigquery for this which complicates things a lot
		downloads = -1
	default:
		return fmt.Errorf("unknown registry type")
	}

	for _, m := range maintainer {
		m = strings.TrimSpace(m)
		// npm user if it starts with @
		if m == "" || strings.HasPrefix(m, "@") {
			continue
		}

		split := strings.Split(m, "@")
		if len(split) != 2 {
			a.log.Debugf("maintainer %s is no valid email address", m)
			continue
		}
		domain := strings.TrimSpace(split[1])
		if domain == "users.noreply.github.com" {
			continue
		}

		if domain == "" {
			continue
		}

		maintainerReturn, err := a.checkDomain(domain, checkExpired)
		if err != nil {
			a.log.WithError(err).Error("")
			continue
		}

		// we return nil on repeated errors
		if maintainerReturn == nil {
			continue
		}

		var text string
		fields := logrus.Fields{
			"package":    p,
			"maintainer": m,
		}

		switch registryType {
		case RegistryNPM:
			fields["link"] = getNPMLink(p)
		case RegistryPypi:
			fields["link"] = getPypiLink(p)
		default:
			return fmt.Errorf("invalid registry type")
		}

		if downloads >= 0 {
			// pypi downloads are currently always -1
			fields["downloads"] = downloads
		}

		if maintainerReturn.unregistered {
			text = "[HIT] DOMAIN UNREGISTERED"
			fields["domain"] = maintainerReturn.domain
		} else if maintainerReturn.expired && checkExpired {
			text = "[POSSIBLE HIT] DOMAIN EXPIRES WITHIN 7 DAYS OR IS ALREADY EXPIRED"
			fields["domain"] = maintainerReturn.domain
			fields["expiration"] = maintainerReturn.expireDate
			fields["registrar"] = maintainerReturn.registrar
		} else if maintainerReturn.unregisteredMX {
			text = "[HIT] UNREGISTERED MX"
			fields["domain"] = maintainerReturn.domain
			fields["mx"] = strings.Join(maintainerReturn.mxDomains, ", ")
		}
		if text != "" {
			a.printResult(downloads, text, fields)
		}
	}

	return nil
}

type checkReturn struct {
	unregistered   bool
	domain         string
	expired        bool
	registrar      string
	expireDate     string
	unregisteredMX bool
	mxDomains      []string
}

func (a *app) checkDomain(domain string, checkExpired bool) (*checkReturn, error) {
	unregistered, err := a.checkDomainUnregistered(domain)
	if err != nil {
		var whoiserr *WhoisError
		if errors.As(err, &whoiserr) {
			if whoiserr.repeatedError {
				// ignore already printed errors
				return nil, nil
			}
		}
		return nil, fmt.Errorf("could not check domain %s for unregistered state: %w", domain, err)
	}
	mxUnregisteredDomains, err := a.checkDomainMXUnregistered(domain)
	if err != nil {
		return nil, fmt.Errorf("could not check domain %s for unregistered MX state: %w", domain, err)
	}

	ret := checkReturn{
		unregistered:   unregistered,
		domain:         domain,
		unregisteredMX: len(mxUnregisteredDomains) > 0,
		mxDomains:      mxUnregisteredDomains,
	}

	if checkExpired {
		expired, date, registrar, err := a.checkDomainExpiresSoon(domain)
		if err != nil {
			var whoiserr *WhoisError
			if errors.As(err, &whoiserr) {
				if whoiserr.repeatedError {
					// ignore already printed errors
					return nil, nil
				}
			}
			return nil, fmt.Errorf("could not check domain %s for expiry: %w", domain, err)
		}
		ret.expired = expired
		ret.expireDate = date
		ret.registrar = registrar
	}

	return &ret, nil
}

func (a *app) checkMXUnregistered(mx string) (bool, error) {
	// check A entry of MX, if it exists we do not need to do a whois which is rate limited
	aRecords, err := a.domainResolve(mx)
	if err != nil {
		return false, err
	}
	if len(aRecords) > 0 {
		return false, nil
	}

	whois, err := a.whois(mx)
	if err != nil {
		var whoiserr *WhoisError
		if errors.As(err, &whoiserr) {
			if whoiserr.repeatedError {
				// ignore already printed errors
				return false, nil
			}
		}
		return false, fmt.Errorf("error on checking mx whois for %s: %w", mx, err)
	}
	if whois == nil {
		return true, nil
	}

	return false, err
}

func (a *app) checkDomainExpiresSoon(domain string) (bool, string, string, error) {
	// make sure we check the root domain here and not a subdomain
	rootDomain, err := getRootDomain(domain)
	if err != nil {
		return false, "", "", fmt.Errorf("could not get root domain for %s: %w", domain, err)
	}

	whois, err := a.whois(rootDomain)
	if err != nil {
		return false, "", "", fmt.Errorf("error on whois for %s: %w", rootDomain, err)
	}
	if whois == nil {
		return false, "", "", nil
	}
	expires, err := a.compareDomainExpiryDate(whois.Domain.ExpirationDateInTime, 7)
	if err != nil {
		return false, "", "", err
	}

	if expires {
		expirationDate := ""
		registrar := ""
		if whois.Domain != nil {
			expirationDate = whois.Domain.ExpirationDate
		}
		if whois.Registrar != nil {
			registrar = whois.Registrar.Name
		}
		return true, expirationDate, registrar, nil
	}

	return false, "", "", nil
}

func (a *app) checkDomainUnregistered(domain string) (bool, error) {
	// make sure we check the root domain here and not a subdomain
	rootDomain, err := getRootDomain(domain)
	if err != nil {
		return false, fmt.Errorf("could not get root domain for %s: %w", domain, err)
	}

	nameServer, err := a.domainNS(rootDomain)
	if err != nil {
		return false, err
	}

	// no nameservers returned, do a whois
	if len(nameServer) == 0 {
		whois, err := a.whois(rootDomain)
		if err != nil {
			return false, fmt.Errorf("error on whois for %s: %w", rootDomain, err)
		}
		if whois == nil {
			// domain unregistered, return this
			return true, nil
		}
	}
	return false, nil
}

func (a *app) checkDomainMXUnregistered(domain string) ([]string, error) {
	var unregisteredDomains []string

	// check mx records
	mx, err := a.domainMX(domain)
	if err != nil {
		return unregisteredDomains, err
	}

	if len(mx) > 0 {
		for _, mxEntry := range mx {
			if mxEntry == "" {
				continue
			}
			unregistered, err := a.checkMXUnregistered(mxEntry)
			if err != nil {
				a.log.WithError(err).Error("")
				continue
			}
			if unregistered {
				unregisteredDomains = append(unregisteredDomains, mxEntry)
			}
		}
	}

	return unregisteredDomains, nil
}

func (a *app) compareDomainExpiryDate(expirationDate *time.Time, daysBefore int) (bool, error) {
	if expirationDate == nil {
		return false, nil
	}

	date := time.Now()
	alertDate := date.AddDate(0, 0, daysBefore)
	if expirationDate.Before(alertDate) { // nolint: gosimple
		return true, nil
	}

	return false, nil
}

func (a *app) printResult(downloads int64, text string, fields logrus.Fields) {
	switch {
	case downloads >= 1000000:
		a.log.WithFields(fields).Error(text)
	case downloads >= 100000 && downloads < 1000000:
		a.log.WithFields(fields).Warn(text)
	default:
		a.log.WithFields(fields).Info(text)
	}
}
