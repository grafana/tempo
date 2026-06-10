package backend

import (
	"github.com/google/go-cmp/cmp/cmpopts"
)

// cmpMetaOpt lets cmp compare wiresmith-generated metas whose unexported
// presence bitmap differs between struct literals and unmarshaled values.
var cmpMetaOpt = cmpopts.IgnoreUnexported(BlockMeta{}, CompactedBlockMeta{}, TenantIndex{})
