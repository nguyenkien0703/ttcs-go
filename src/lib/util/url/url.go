package lib_util_url

import (
	"fmt"
	"net/url"
	"strings"
)

func AddQuery(src, key, value string) string {
	sep := "?"
	if strings.Index(src, "?") != -1 {
		sep = "&"
	}
	return fmt.Sprintf("%s%s%s=%s", src, sep, key, url.QueryEscape(value))
}
