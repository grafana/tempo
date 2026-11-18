package ingest

import (
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/pkg/kversion"
)

// DefaultAPIVersionsMaxVersion is the highest ApiVersions request version the
// Kafka clients send by default.
//
// franz-go >= 1.22 sends ApiVersions v5 first. WarpStream agents close the
// connection on v5 instead of replying UNSUPPORTED_VERSION, so the client
// can't fall back on its own. franz-go 1.21.x already sends v4, so the cap
// changes nothing there.
const DefaultAPIVersionsMaxVersion = 4

const apiVersionsMaxVersionUsage = "The highest ApiVersions request version the Kafka client sends, up to the highest version the client supports. " +
	"The default of 4 keeps the client compatible with backends that close the connection on ApiVersions v5, such as WarpStream. " +
	"0 removes the cap, so the client sends the highest version it supports."

// supportedAPIVersionsMaxVersion returns the highest ApiVersions version this
// franz-go supports, so the allowed values move up with franz-go upgrades.
func supportedAPIVersionsMaxVersion() int {
	if v, ok := kversion.Stable().LookupMaxKeyVersion(kmsg.ApiVersions.Int16()); ok {
		return int(v)
	}
	return DefaultAPIVersionsMaxVersion
}

// apiVersionsMaxVersionOpts returns the client options that cap ApiVersions at
// maxVersion, or none if maxVersion is 0 or less. franz-go sends whatever
// version it is given, so values above supportedAPIVersionsMaxVersion are
// lowered to it, for callers that don't validate their config.
func apiVersionsMaxVersionOpts(maxVersion int) []kgo.Opt {
	if maxVersion <= 0 {
		return nil
	}
	maxVersion = min(maxVersion, supportedAPIVersionsMaxVersion())
	// kversion.Stable() is franz-go's default, so only ApiVersions changes.
	versions := kversion.Stable()
	versions.SetMaxKeyVersion(kmsg.ApiVersions.Int16(), int16(maxVersion))
	return []kgo.Opt{kgo.MaxVersions(versions)}
}
