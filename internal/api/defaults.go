package api

import "regexp"

const (
	defaultHost = "captain.build"

	contentTypeJSON   = "application/json"
	headerContentType = "Content-Type"
)

var (
	bearerTokenRegexp     = regexp.MustCompile(`Bearer.*`)
	setCookieHeaderRegexp = regexp.MustCompile(`Set-Cookie:.*`)
)
