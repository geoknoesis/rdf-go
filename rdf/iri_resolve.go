package rdf

import (
	"net/url"
	"strings"
)

// resolveIRI resolves a relative IRI against a base IRI according to RFC 3986.
func resolveIRI(baseStr, relative string) string {
	// Use Go's net/url for proper RFC 3986 resolution.
	baseURL, err := url.Parse(baseStr)
	if err != nil {
		// Fallback to simple concatenation if base is invalid.
		if strings.HasSuffix(baseStr, "/") {
			return baseStr + relative
		}
		lastSlash := strings.LastIndex(baseStr, "/")
		if lastSlash >= 0 {
			return baseStr[:lastSlash+1] + relative
		}
		return baseStr + "/" + relative
	}

	relURL, err := url.Parse(relative)
	if err != nil {
		// Fallback if relative is invalid.
		if strings.HasSuffix(baseStr, "/") {
			return baseStr + relative
		}
		lastSlash := strings.LastIndex(baseStr, "/")
		if lastSlash >= 0 {
			return baseStr[:lastSlash+1] + relative
		}
		return baseStr + "/" + relative
	}

	// If relative URL has a scheme, it's absolute - return as-is.
	if relURL.Scheme != "" {
		return relative
	}

	resolved := baseURL.ResolveReference(relURL)
	return resolved.String()
}
