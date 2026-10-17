//go:build amd64 && !noasm && !appengine
// +build amd64,!noasm,!appengine

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

	"github.com/cloudwego/base64x/internal/native"
)

func encode(out *[]byte, src []byte, mode int) {
	native.B64Encode(out, &src, mode)
}

func decode(out *[]byte, src []byte, mode int) (int, error) {
	n := native.B64Decode(out, mem2addr(src), len(src), mode)
	if n >= 0 {
		return n, nil
	}
	return 0, base64.CorruptInputError(-n - 1)
}
