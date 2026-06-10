package backend

import (
	"strings"

	"github.com/google/go-cmp/cmp"
)

// cmpMetaOpt lets cmp compare wiresmith-generated metas whose XXX_fieldsPresent
// presence bitmap differs between struct literals and unmarshaled values.
// (The field is exported, so cmpopts.IgnoreUnexported does not cover it.)
var cmpMetaOpt = cmp.FilterPath(func(p cmp.Path) bool {
	return strings.HasSuffix(p.Last().String(), ".XXX_fieldsPresent")
}, cmp.Ignore())
