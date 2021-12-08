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

// secret needed to access us.gcr.io in deploy_to_dev()
local docker_config_json_secret = secret('dockerconfigjson', 'secret/data/common/gcr', '.dockerconfigjson');

// secret needed for dep-tools
local gh_token_secret = secret('gh_token', 'infra/data/ci/github/grafanabot', 'pat');

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

local deploy_to_dev() = {
  image: 'us.gcr.io/kubernetes-dev/drone/plugins/updater',
  settings: {
    config_json: |||
        {
          "destination_branch": "master",
          "pull_request_branch_prefix": "cd-tempo-dev",
          "pull_request_enabled": false,
          "pull_request_team_reviewers": [
            "tempo"
          ],
          "repo_name": "deployment_tools",
          "update_jsonnet_attribute_configs": [
            {
              "file_path": "ksonnet/environments/tempo/dev-us-central-0.tempo-dev-01/images.libsonnet",
              "jsonnet_key": %s,
              "jsonnet_value_file": ".tags"
            }
            {
              "file_path": "ksonnet/environments/tempo/dev-us-central-0.tempo-dev-01/images.libsonnet",
              "jsonnet_key": %s,
              "jsonnet_value_file": ".tags"
            }
            {
              "file_path": "ksonnet/environments/tempo/dev-us-central-0.tempo-dev-01/images.libsonnet",
              "jsonnet_key": %s,
              "jsonnet_value_file": ".tags"
            }
          ]
        }
      ||| % ['tempo', 'tempo-query', 'tempo-vulture'],
    github_token: {
        from_secret: gh_token_secret.name,
    },
  },
};

## Pipelines & resources

[
  // A pipeline to build Docker images for every app and for every arch
  (
    pipeline('docker-' + arch, arch) {
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
  // Continuously Deploy to dev env
  pipeline('cd-to-dev-env') {
    trigger: {
      ref: [
        // always deploy tip of main to dev
        'refs/heads/main',
      ],
    },
    image_pull_secrets: [
      docker_config_json_secret.name,
    ],
    steps+: [
      image_tag(),
    ] + [
      deploy_to_dev(),
    ],
    depends_on+: [
      // wait for images to be published on dockerhub
      'manifest',
    ],
  },
] + [
  docker_username_secret,
  docker_password_secret,
  docker_config_json_secret,
  gh_token_secret,
]
