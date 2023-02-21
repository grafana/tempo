// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package scrubber

import (
	"fmt"
	"regexp"
	"strings"
)

// DefaultScrubber is the scrubber used by the package-level cleaning functions.
//
// It includes a set of agent-specific replacers.  It can scrub DataDog App
// and API keys, passwords from URLs, and multi-line PEM-formatted TLS keys and
// certificates.  It contains special handling for YAML-like content (with
// lines of the form "key: value") and can scrub passwords, tokens, and SNMP
// community strings in such content.
//
// See default.go for details of these replacers.
var DefaultScrubber = &Scrubber{}

func init() {
	AddDefaultReplacers(DefaultScrubber)
}

// AddDefaultReplacers to a scrubber. This is called automatically for
// DefaultScrubber, but can be used to initialize other, custom scrubbers with
// the default replacers.
func AddDefaultReplacers(scrubber *Scrubber) {
	hintedAPIKeyReplacer := Replacer{
		// If hinted, mask the value regardless if it doesn't match 32-char hexadecimal string
		Regex: regexp.MustCompile(`(api_?key=)\b[a-zA-Z0-9]+([a-zA-Z0-9]{5})\b`),
		Hints: []string{"api_key", "apikey"},
		Repl:  []byte(`$1***************************$2`),
	}
	hintedAPPKeyReplacer := Replacer{
		// If hinted, mask the value regardless if it doesn't match 40-char hexadecimal string
		Regex: regexp.MustCompile(`(ap(?:p|plication)_?key=)\b[a-zA-Z0-9]+([a-zA-Z0-9]{5})\b`),
		Hints: []string{"app_key", "appkey", "application_key"},
		Repl:  []byte(`$1***********************************$2`),
	}
	hintedBearerReplacer := Replacer{
		Regex: regexp.MustCompile(`\bBearer [a-fA-F0-9]{59}([a-fA-F0-9]{5})\b`),
		Hints: []string{"Bearer"},
		Repl:  []byte(`Bearer ***********************************************************$1`),
	}
	apiKeyReplacerYAML := Replacer{
		Regex: regexp.MustCompile(`(\-|\:|,|\[|\{)(\s+)?\b[a-fA-F0-9]{27}([a-fA-F0-9]{5})\b`),
		Repl:  []byte(`$1$2"***************************$3"`),
	}
	apiKeyReplacer := Replacer{
		Regex: regexp.MustCompile(`\b[a-fA-F0-9]{27}([a-fA-F0-9]{5})\b`),
		Repl:  []byte(`***************************$1`),
	}
	appKeyReplacerYAML := Replacer{
		Regex: regexp.MustCompile(`(\-|\:|,|\[|\{)(\s+)?\b[a-fA-F0-9]{35}([a-fA-F0-9]{5})\b`),
		Repl:  []byte(`$1$2"***********************************$3"`),
	}
	appKeyReplacer := Replacer{
		Regex: regexp.MustCompile(`\b[a-fA-F0-9]{35}([a-fA-F0-9]{5})\b`),
		Repl:  []byte(`***********************************$1`),
	}
	rcAppKeyReplacer := Replacer{
		Regex: regexp.MustCompile(`\bDDRCM_[A-Z0-9]+([A-Z0-9]{5})\b`),
		Repl:  []byte(`***********************************$1`),
	}
	// URI Generic Syntax
	// https://tools.ietf.org/html/rfc3986
	uriPasswordReplacer := Replacer{
		Regex: regexp.MustCompile(`([A-Za-z][A-Za-z0-9+-.]+\:\/\/|\b)([^\:]+)\:([^\s]+)\@`),
		Repl:  []byte(`$1$2:********@`),
	}
	passwordReplacer := Replacer{
		Regex: matchYAMLKeyPart(`(pass(word)?|pwd)`),
		Hints: []string{"pass", "pwd"},
		Repl:  []byte(`$1 "********"`),
	}
	tokenReplacer := Replacer{
		Regex: matchYAMLKeyEnding(`token`),
		Hints: []string{"token"},
		Repl:  []byte(`$1 "********"`),
	}
	snmpReplacer := Replacer{
		Regex: matchYAMLKey(`(community_string|authKey|privKey|community|authentication_key|privacy_key)`),
		Hints: []string{"community_string", "authKey", "privKey", "community", "authentication_key", "privacy_key"},
		Repl:  []byte(`$1 "********"`),
	}
	snmpMultilineReplacer := Replacer{
		Regex: matchYAMLKeyWithListValue("(community_strings)"),
		Hints: []string{"community_strings"},
		Repl:  []byte(`$1 "********"`),
	}
	certReplacer := Replacer{
		Regex: matchCert(),
		Hints: []string{"BEGIN"},
		Repl:  []byte(`********`),
	}
	scrubber.AddReplacer(SingleLine, hintedAPIKeyReplacer)
	scrubber.AddReplacer(SingleLine, hintedAPPKeyReplacer)
	scrubber.AddReplacer(SingleLine, hintedBearerReplacer)
	scrubber.AddReplacer(SingleLine, apiKeyReplacerYAML)
	scrubber.AddReplacer(SingleLine, apiKeyReplacer)
	scrubber.AddReplacer(SingleLine, appKeyReplacerYAML)
	scrubber.AddReplacer(SingleLine, appKeyReplacer)
	scrubber.AddReplacer(SingleLine, rcAppKeyReplacer)
	scrubber.AddReplacer(SingleLine, uriPasswordReplacer)
	scrubber.AddReplacer(SingleLine, passwordReplacer)
	scrubber.AddReplacer(SingleLine, tokenReplacer)
	scrubber.AddReplacer(SingleLine, snmpReplacer)
	scrubber.AddReplacer(MultiLine, snmpMultilineReplacer)
	scrubber.AddReplacer(MultiLine, certReplacer)
}

func matchYAMLKeyPart(part string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`(\s*(\w|_)*%s(\w|_)*\s*:).+`, part))
}

func matchYAMLKey(key string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`(\s*%s\s*:).+`, key))
}

func matchYAMLKeyEnding(ending string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`(^\s*(\w|_)*%s\s*:).+`, ending))
}

func matchCert() *regexp.Regexp {
	/*
	   Try to match as accurately as possible. RFC 7468's ABNF
	   Backreferences are not available in go, so we cannot verify
	   here that the BEGIN label is the same as the END label.
	*/
	return regexp.MustCompile(
		`-----BEGIN (?:.*)-----[A-Za-z0-9=\+\/\s]*-----END (?:.*)-----`,
	)
}

// matchYAMLKeyWithListValue matches YAML keys with array values.
// caveat: doesn't work if the array contain nested arrays. Example:
//
//	key: [
//	 [a, b, c],
//	 def]
func matchYAMLKeyWithListValue(key string) *regexp.Regexp {
	/*
				Example 1:
				network_devices:
		  		  snmp_traps:
		            community_strings:
				    - 'pass1'
				    - 'pass2'

				Example 2:
				network_devices:
		  		  snmp_traps:
				    community_strings: ['pass1', 'pass2']

				Example 3:
				network_devices:
		  		  snmp_traps:
				    community_strings: [
				    'pass1',
				    'pass2']
	*/
	return regexp.MustCompile(
		fmt.Sprintf(`(\s*%s\s*:)\s*(?:\n(?:\s+-\s+.*)*|\[(?:\n?.*?)*?\])`, key),
		/*           -----------      ---------------  -------------
		             match key(s)     |                |
		                              match multiple   match anything
		                              lines starting   enclosed between `[` and `]`
		                              with `-`
		*/
	)
}

// ScrubFile scrubs credentials from the given file, using the
// default scrubber.
func ScrubFile(filePath string) ([]byte, error) {
	return DefaultScrubber.ScrubFile(filePath)
}

// ScrubBytes scrubs credentials from the given slice of bytes,
// using the default scrubber.
func ScrubBytes(file []byte) ([]byte, error) {
	return DefaultScrubber.ScrubBytes(file)
}

// ScrubString scrubs credentials from the given string, using the default scrubber.
func ScrubString(data string) (string, error) {
	res, err := DefaultScrubber.ScrubBytes([]byte(data))
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// ScrubLine scrubs credentials from a single line of text, using the default
// scrubber.  It can be safely applied to URLs or to strings containing URLs.
// It does not run multi-line replacers, and should not be used on multi-line
// inputs.
func ScrubLine(url string) string {
	return DefaultScrubber.ScrubLine(url)
}

// AddStrippedKeys adds to the set of YAML keys that will be recognized and have
// their values stripped.  This modifies the DefaultScrubber directly.
func AddStrippedKeys(strippedKeys []string) {
	if len(strippedKeys) > 0 {
		configReplacer := Replacer{
			Regex: matchYAMLKey(fmt.Sprintf("(%s)", strings.Join(strippedKeys, "|"))),
			Hints: strippedKeys,
			Repl:  []byte(`$1 "********"`),
		}
		DefaultScrubber.AddReplacer(SingleLine, configReplacer)
	}
}
