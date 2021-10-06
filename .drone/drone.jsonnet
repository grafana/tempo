local apps = ['tempo', 'tempo-vulture', 'tempo-query'];
local archs = ['amd64', 'arm64'];

## Building blocks ##

local pipeline(name, arch = 'amd64') = {
  kind: 'pipeline',
  name: name,
  platform: {
    os: 'linux',
    arch: arch,
  },
  steps: [],
  depends_on: [],
  trigger: {
    ref: [
        'refs/heads/main',
        'refs/tags/v*',
        // weekly release branches
        'refs/heads/r?',
        'refs/heads/r??',
    ],
  },
};

local secret(name, vault_path, vault_key) = {
  kind: 'secret',
  name: name,
  get: {
    path: vault_path,
    name: vault_key,
  },
};

local docker_username_secret = secret('docker_username', 'infra/data/ci/docker_hub', 'username');
local docker_password_secret = secret('docker_password', 'infra/data/ci/docker_hub', 'password');

## Steps ##

local image_tag(arch = '') = {
  name: 'image-tag',
  image: 'alpine/git',
  commands: (
    // the alpine/git image has apk errors when run in arm64, fix them before running any other apk command
    // https://github.com/alpine-docker/git/issues/35
    if arch == 'arm64' then ['apk fix'] else []
  ) + [
    'apk --update --no-cache add bash',
    'git fetch origin --tags',
  ] + (
    if arch == '' then [
      'echo $(./tools/image-tag) > .tags'
    ] else [
      'echo $(./tools/image-tag)-%s > .tags' % arch
    ]
  ),
};

local build_binaries(arch) = {
  name: 'build-tempo-binaries',
  image: 'golang:1.17-alpine',
  commands: [
    'apk add make git',
  ] + [
    'COMPONENT=%s GOARCH=%s make exe' % [app, arch]
    for app in apps
  ],
};

local docker_build(arch, app) = {
  name: 'build-%s-image' % app,
  image: 'plugins/docker',
  settings: {
    dockerfile: 'cmd/%s/Dockerfile' % app,
    repo: 'grafana/%s' % app,
    username: { from_secret: docker_username_secret.name },
    password: { from_secret: docker_password_secret.name },
    build_args: [
      'TARGETARCH=' + arch,
    ],
  },
};

local docker_manifest(app) = {
  name: 'manifest-%s' % app,
  image: 'plugins/manifest',
  settings: {
    username: { from_secret: docker_username_secret.name },
    password: { from_secret: docker_password_secret.name },
    spec: '.drone/docker-manifest.tmpl',
    target: app,
  },
};

## Pipelines & resources

[
  // A pipeline to build Docker images for every app and for every arch
  (
    pipeline('docker-' + arch) {
      steps+: [
        image_tag(arch),
        build_binaries(arch),
      ] + [
        docker_build(arch, app)
        for app in apps
      ],
    }
  )
  for arch in archs
] + [
  // Publish Docker manifests
  pipeline('manifest') {
    steps+: [
      image_tag(),
    ] + [
      docker_manifest(app)
      for app in apps
    ],
    depends_on+: [
      'docker-%s' % arch
      for arch in archs
    ],
  },
] + [
  docker_username_secret,
  docker_password_secret,
]
