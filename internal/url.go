package internal

import (
	"net/url"
	"strings"

	"github.com/goware/urlx"
)

func IntentWebsocketURL(urlString string) (*url.URL, error) {

	parsedURL, err := urlx.Parse(urlString)
	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme == "ws" {
		parsedURL.Scheme = "http"
	}
	if parsedURL.Scheme == "wss" {
		parsedURL.Scheme = "https"
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		parsedURL.Scheme = "http"
	}

	// Remove trailing slash if any
	parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")

	urlPath := parsedURL.Path

	// If the URL path doesn't start with /intent/ws, set it
	expectedPrefix := "/intent/ws"
	if !strings.HasPrefix(urlPath, expectedPrefix) {
		parsedURL.Path = expectedPrefix
	}
	return parsedURL, nil
}

func IntentHTTPURL(urlString string) (*url.URL, error) {

	parsedURL, err := IntentWebsocketURL(urlString)
	if err != nil {
		return nil, err
	}

	expectedSuffix := "/intent"
	if !strings.HasSuffix(parsedURL.Path, expectedSuffix) {
		parsedURL.Path = expectedSuffix
	}

	return parsedURL, nil
}
