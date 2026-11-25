// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.apps', name: 'apps' },
  dashboard:
    {
      '#': { help: 'grafonnet.apps.dashboard', name: 'dashboard' },
      v2: import 'apps/dashboard/v2.libsonnet',
    },
  folder:
    {
      '#': { help: 'grafonnet.apps.folder', name: 'folder' },
      v1beta1: import 'apps/folder/v1beta1.libsonnet',
      v1: import 'apps/folder/v1.libsonnet',
    },
  playlist:
    {
      '#': { help: 'grafonnet.apps.playlist', name: 'playlist' },
      v1: import 'apps/playlist/v1.libsonnet',
      v0alpha1: import 'apps/playlist/v0alpha1.libsonnet',
    },
  preferences:
    {
      '#': { help: 'grafonnet.apps.preferences', name: 'preferences' },
      v1alpha1: import 'apps/preferences/v1alpha1.libsonnet',
    },
}
