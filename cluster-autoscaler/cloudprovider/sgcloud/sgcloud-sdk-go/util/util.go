package util

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	klog "k8s.io/klog/v2"
)

// Debug generates debug info for debug mode.
func Debug(title, message string) {
	if title != "" {
		klog.V(5).Infof("----------------------------DEBUG: start of %s ----------------------------", title)
	}

	klog.V(5).Infof(message)

	if title != "" {
		klog.V(5).Infof("----------------------------DEBUG: end of %s------------------------------", title)
	}
}

// GetURL gets the full URL for a http request.
func GetURL(protocol, host, uriPath string, params map[string]string) string {
	if strings.Index(uriPath, "/") == 0 {
		uriPath = uriPath[1:]
	}

	query := strings.Trim(ToCanonicalQueryString(params), " ")

	if query == "" {
		return fmt.Sprintf("%s/%s", HostToURL(host, protocol), uriPath)
	}

	return fmt.Sprintf("%s/%s?%s", HostToURL(host, protocol), uriPath, query)
}

// HostToURL returns the whole URL string.
func HostToURL(host, protocol string) string {
	if matched, _ := regexp.MatchString("^[[:alpha:]]+:", host); matched {
		return host
	}

	if protocol == "" {
		protocol = "http"
	}

	return fmt.Sprintf("%s://%s", protocol, host)
}

// ToCanonicalQueryString returns the canonicalized query string.
func ToCanonicalQueryString(params map[string]string) string {
	if params == nil {
		return ""
	}

	encodedQueryStrings := make([]string, 0, 10)
	var query string

	for key, value := range params {
		if key != "" {
			query = URLEncode(key) + "="
			if value != "" {
				query += URLEncode(value)
			}
			encodedQueryStrings = append(encodedQueryStrings, query)
		}
	}

	sort.Strings(encodedQueryStrings)

	return strings.Join(encodedQueryStrings, "&")
}

// URLEncode encodes a string like Javascript's encodeURIComponent()
func URLEncode(str string) string {
	// BUG(go): see https://github.com/golang/go/issues/4013
	// use %20 instead of the + character for encoding a space
	return strings.Replace(url.QueryEscape(str), "+", "%20", -1)
}
