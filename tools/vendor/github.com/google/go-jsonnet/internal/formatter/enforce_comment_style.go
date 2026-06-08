/*
Copyright 2019 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package formatter

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/pass"
)

// EnforceCommentStyle is a formatter pass that ensures the comments are styled
// according to the configuration in Options.
type EnforceCommentStyle struct {
	pass.Base
	Options         Options
	seenFirstFodder bool
}

// FodderElement implements this pass.
func (c *EnforceCommentStyle) FodderElement(p pass.ASTPass, element *ast.FodderElement, ctx pass.Context) {
	if element.Kind != ast.FodderInterstitial {
		if len(element.Comment) == 1 {
			comment := &element.Comment[0]
			if c.Options.CommentStyle == CommentStyleHash && (*comment)[0] == '/' {
				*comment = "#" + (*comment)[2:]
			}
			if c.Options.CommentStyle == CommentStyleSlash && (*comment)[0] == '#' {
				if !c.seenFirstFodder && len(*comment) > 1 && (*comment)[1] == '!' {
					return
				}
				*comment = "//" + (*comment)[1:]
			}
		}
		c.seenFirstFodder = true
	}
}
