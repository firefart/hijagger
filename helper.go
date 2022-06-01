package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/jpillora/go-tld"
)

func parseUnknownDate(dateString string) (time.Time, error) {
	formats := [...]string{
		"2006-01-02T15:04:05Z",
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006. 01. 02.",
		"02-Jan-2006",
		"02/01/2006 15:04:05",
		"02.01.2006",
		"02-01-2006",
		"02.01.2006 15:04:05",
		"02.1.2006 15:04:05",
		"2.1.2006 15:04:05",
		"2006-01-02 15:04:05-07",
		"02-Jan-2006 15:04:05",
		"January _2 2006",
		"02/01/2006",
		"01/02/2006",
		"2006-01-02 15:04:05 MST",
		"2006-Jan-02",
		"2006-Jan-02.",
		"2006-01-02 15:04:05 (MST+3)",
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano,
		time.Stamp,
		time.StampMilli,
		time.StampMicro,
		time.StampNano,
	}

	for _, format := range formats {
		result, err := time.Parse(format, dateString)
		if err != nil {
			var parserError *time.ParseError
			if errors.As(err, &parserError) {
				// if it's a parser error try the next format
				continue
			} else {
				return time.Now(), err
			}
		} else {
			return result, nil
		}
	}

	return time.Now(), fmt.Errorf("could not parse %s as a date", dateString)
}

func getRootDomain(domain string) (string, error) {
	// add a fake schema so the parser works
	u, err := tld.Parse("http://" + domain)
	if err != nil {
		return "", err
	}
	rootDomain := fmt.Sprintf("%s.%s", u.Domain, u.TLD)
	return rootDomain, nil
}
