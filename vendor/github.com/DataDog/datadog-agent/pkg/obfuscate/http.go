// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package obfuscate

import (
	"net/url"
	"strings"
)

// ObfuscateURLString obfuscates the given URL. It must be a valid URL and at least one
// HTTP obfuscation option must be enabled at Obfuscator instantiation time.
func (o *Obfuscator) ObfuscateURLString(val string) string {
	if !o.opts.HTTP.RemoveQueryString && !o.opts.HTTP.RemovePathDigits {
		// nothing to do
		return val
	}
	u, err := url.Parse(val)
	if err != nil {
		// should not happen for valid URLs, but better obfuscate everything
		// rather than expose sensitive information when this option is on.
		return "?"
	}
	if o.opts.HTTP.RemoveQueryString && u.RawQuery != "" {
		u.ForceQuery = true // add the '?'
		u.RawQuery = ""
	}
	if o.opts.HTTP.RemovePathDigits {
		segs := strings.Split(u.Path, "/")
		var changed bool
		for i, seg := range segs {
			for _, ch := range []byte(seg) {
				if ch >= '0' && ch <= '9' {
					// we can not set the question mark directly here because the url
					// package will escape it into %3F, so we use this placeholder and
					// replace it further down.
					segs[i] = "/REDACTED/"
					changed = true
					break
				}
			}
		}
		if changed {
			u.Path = strings.Join(segs, "/")
		}
	}
	return strings.Replace(u.String(), "/REDACTED/", "?", -1)
}
