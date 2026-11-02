// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package azurevm provides a [resource.Detector] which supports detecting
attributes specific to Azure VMs.

According to semantic conventions for [host], [cloud], and [os] attributes,
each of the following attributes is added if it is available:

  - cloud.provider
  - cloud.platform
  - cloud.region
  - cloud.resource_id
  - cloud.account.id
  - cloud.availability_zone
  - host.id
  - host.name
  - host.type
  - os.type
  - os.version

The following Azure-specific attributes are also added if available:

  - azure.vm.name
  - azure.vm.size
  - azure.vm.scaleset.name
  - azure.resource_group.name

[NewResourceDetector] accepts options that configure what the detector emits.

When configured with [WithTagKeyFilter], the detector additionally emits an
azure.tag.<name> attribute for every VM tag whose key satisfies the configured
filter. No VM tags are emitted otherwise.

[WithAttributeFilter] restricts the returned resource to the attributes for
which the filter returns true. It is applied last, so it also applies to the
azure.tag.<name> attributes selected by [WithTagKeyFilter].

[host]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/host.md
[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[os]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/os.md
*/
package azurevm
