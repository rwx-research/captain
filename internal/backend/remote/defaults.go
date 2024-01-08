package remote

import "regexp"

const (
	defaultHost = "cloud.rwx.com"

	contentTypeJSON   = "application/json"
	headerContentType = "Content-Type"
)

var (
	bearerTokenRegexp     = regexp.MustCompile(`Bearer.*`)
	setCookieHeaderRegexp = regexp.MustCompile(`Set-Cookie:.*`)
)
