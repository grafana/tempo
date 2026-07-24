//go:build !amd64 || noasm || appengine
// +build !amd64 noasm appengine

/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package base64x

import (
	"encoding/base64"
	"encoding/json"
)

func encodingForMode(mode int) *base64.Encoding {
	switch mode & (_MODE_URL | _MODE_RAW) {
	case _MODE_URL:
		return base64.URLEncoding
	case _MODE_RAW:
		return base64.RawStdEncoding
	case _MODE_URL | _MODE_RAW:
		return base64.RawURLEncoding
	default:
		return base64.StdEncoding
	}
}

func encode(out *[]byte, src []byte, mode int) {
	enc := encodingForMode(mode)
	dst := *out
	start := len(dst)
	need := enc.EncodedLen(len(src))
	dst = dst[:start+need]
	enc.Encode(dst[start:], src)
	*out = dst
}

func decode(out *[]byte, src []byte, mode int) (int, error) {
	input := src
	if mode&_MODE_JSON != 0 {
		unquoted, err := unquoteJSONBase64(src)
		if err != nil {
			return 0, err
		}
		input = unquoted
	}

	enc := encodingForMode(mode)
	dst := *out
	start := len(dst)
	buf := dst[start:cap(dst)]
	n, err := enc.Decode(buf, input)
	if err != nil {
		return 0, err
	}
	*out = dst[:start+n]
	return n, nil
}

func unquoteJSONBase64(src []byte) ([]byte, error) {
	quoted := make([]byte, 0, len(src)+2)
	quoted = append(quoted, '"')
	quoted = append(quoted, src...)
	quoted = append(quoted, '"')

	var decoded string
	if err := json.Unmarshal(quoted, &decoded); err != nil {
		return nil, err
	}
	return []byte(decoded), nil
}
