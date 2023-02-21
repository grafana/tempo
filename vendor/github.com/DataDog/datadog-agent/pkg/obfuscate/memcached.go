// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package obfuscate

import "strings"

// ObfuscateMemcachedString obfuscates the Memcached command cmd.
func (*Obfuscator) ObfuscateMemcachedString(cmd string) string {
	// All memcached commands end with new lines [1]. In the case of storage
	// commands, key values follow after. Knowing this, all we have to do
	// to obfuscate sensitive information is to remove everything that follows
	// a new line. For non-storage commands, this will have no effect.
	// [1]: https://github.com/memcached/memcached/blob/master/doc/protocol.txt
	out := strings.SplitN(cmd, "\r\n", 2)[0]
	return strings.TrimSpace(out)
}
