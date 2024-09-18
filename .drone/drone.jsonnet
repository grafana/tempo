local apps = ['tempo', 'tempo-vulture', 'tempo-query', 'tempo-cli'];
local archs = ['amd64', 'arm64'];

//# Building blocks ##

local pipeline(name, arch='amd64') = {
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
      'refs/heads/r???',
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

// secrets for pushing serverless code packages
local image_upload_ops_tools_secret = secret('ops_tools_img_upload', 'infra/data/ci/tempo-ops-tools-function-upload', 'credentials.json');

// secret needed to access us.gcr.io in deploy_to_dev()
local docker_config_json_secret = secret('dockerconfigjson', 'secret/data/common/gcr', '.dockerconfigjson');

// secret needed for dep-tools
local gh_token_secret = secret('gh_token', 'infra/data/ci/github/grafanabot', 'pat');
local tempo_app_id_secret = secret('tempo_app_id_secret', 'ci/data/repo/grafana/tempo/github-app', 'app-id');
local tempo_app_installation_id_secret = secret('tempo_app_installation_id_secret', 'ci/data/repo/grafana/tempo/github-app', 'app-installation-id');
local tempo_app_private_key_secret = secret('tempo_app_private_key_secret', 'ci/data/repo/grafana/tempo/github-app', 'app-private-key');

// secret to sign linux packages
local gpg_passphrase = secret('gpg_passphrase', 'infra/data/ci/packages-publish/gpg', 'passphrase');
local gpg_private_key = secret('gpg_private_key', 'infra/data/ci/packages-publish/gpg', 'private-key');

local aws_dev_access_key_id = secret('AWS_ACCESS_KEY_ID-dev', 'infra/data/ci/tempo-dev/aws-credentials-drone', 'access_key_id');
local aws_dev_secret_access_key = secret('AWS_SECRET_ACCESS_KEY-dev', 'infra/data/ci/tempo-dev/aws-credentials-drone', 'secret_access_key');
local aws_prod_access_key_id = secret('AWS_ACCESS_KEY_ID-prod', 'infra/data/ci/tempo-prod/aws-credentials-drone', 'access_key_id');
local aws_prod_secret_access_key = secret('AWS_SECRET_ACCESS_KEY-prod', 'infra/data/ci/tempo-prod/aws-credentials-drone', 'secret_access_key');

local aws_serverless_deployments = [
  {
    env: 'dev',
    bucket: 'dev-tempo-fn-source',
    access_key_id: aws_dev_access_key_id.name,
    secret_access_key: aws_dev_secret_access_key.name,
  },
  {
    env: 'prod',
    bucket: 'prod-tempo-fn-source',
    access_key_id: aws_prod_access_key_id.name,
    secret_access_key: aws_prod_secret_access_key.name,
  },
];


//# Steps ##

// the alpine/git image has apk errors when run on aarch64, this is the most recent image that does not have this issue
// https://github.com/alpine-docker/git/issues/35
local alpine_git_image = 'alpine/git:v2.30.2';

local image_tag(arch='') = {
  name: 'image-tag',
  image: alpine_git_image,
  commands: [
    'apk --update --no-cache add bash',
    'git fetch origin --tags',
  ] + (
    if arch == '' then [
      'echo $(./tools/image-tag) > .tags',
    ] else [
      'echo $(./tools/image-tag)-%s > .tags' % arch,
    ]
  ),
};

local image_tag_for_cd() = {
  name: 'image-tag-for-cd',
  image: alpine_git_image,
  commands: [
    'apk --update --no-cache add bash',
    'git fetch origin --tags',
    'echo "grafana/tempo:$(./tools/image-tag)" > .tags-for-cd-tempo',
    'echo "grafana/tempo-query:$(./tools/image-tag)" > .tags-for-cd-tempo_query',
    'echo "grafana/tempo-vulture:$(./tools/image-tag)" > .tags-for-cd-tempo_vulture',
  ],
};

local build_binaries(arch) = {
  name: 'build-tempo-binaries',
  image: 'golang:1.22-alpine',
  commands: [
    'apk --update --no-cache add make git bash',
  ] + [
    'COMPONENT=%s GOARCH=%s make exe' % [app, arch]
    for app in apps
  ],
};

local docker_build(arch, app, dockerfile='') = {
  name: 'build-%s-image' % app,
  image: 'plugins/docker',
  settings: {
    dockerfile: if dockerfile != '' then dockerfile else 'cmd/%s/Dockerfile' % app,
    repo: 'grafana/%s' % app,
    username: { from_secret: docker_username_secret.name },
    password: { from_secret: docker_password_secret.name },
    platform: '%s/%s' % ['linux', arch],
    build_args: [
      'TARGETARCH=' + arch,
    ],
  },
};

local docker_manifest(app) = {
  name: 'manifest-%s' % app,
  image: 'plugins/manifest:1.4.0',
  settings: {
    username: { from_secret: docker_username_secret.name },
    password: { from_secret: docker_password_secret.name },
    spec: '.drone/docker-manifest.tmpl',
    target: app,
  },
};

local deploy_to_dev() = {
  image: 'us.gcr.io/kubernetes-dev/drone/plugins/updater',
  name: 'update-dev-images',
  settings: {
    config_json: std.manifestJsonEx(
      {
        destination_branch: 'master',
        pull_request_branch_prefix: 'cd-tempo-dev',
        pull_request_enabled: true,
        pull_request_existing_strategy: "ignore",
        repo_name: 'deployment_tools',
        update_jsonnet_attribute_configs: [
          {
            file_path: 'ksonnet/environments/tempo/dev-us-central-0.tempo-dev-01/images.libsonnet',
            jsonnet_key: app,
            jsonnet_value_file: '.tags-for-cd-' + app,
          }
          for app in ['tempo', 'tempo_query', 'tempo_vulture']
        ],
      },
      '  '
    ),
    github_app_id: {
      from_secret: tempo_app_id_secret.name,
    },
    github_app_installation_id: {
      from_secret: tempo_app_installation_id_secret.name,
    },
    github_app_private_key: {
      from_secret: tempo_app_private_key_secret.name,
    },
  },
};

//# Pipelines & resources

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
  // Publish tools Docker manifests
  (
    pipeline('docker-ci-tools-%s' % arch, arch) {
      steps+: [
        image_tag(arch),
        docker_build(arch, 'tempo-ci-tools', dockerfile='tools/Dockerfile'),
      ],
    }
  )
  for arch in archs
] + [
  // Publish Docker manifests
  pipeline('manifest-ci-tools') {
    steps+: [
      image_tag(),
      docker_manifest('tempo-ci-tools'),
    ],
    depends_on+: [
      'docker-ci-tools-%s' % arch
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
      image_tag_for_cd(),
    ] + [
      deploy_to_dev(),
    ],
    depends_on+: [
      // wait for images to be published on dockerhub
      'manifest',
    ],
  },
] + [
  // Build and deploy serverless code packages
  pipeline('build-deploy-serverless') {
    steps+: [
              {
                name: 'build-tempo-serverless',
                image: 'golang:1.22-alpine',
                commands: [
                  'apk add make git zip bash',
                  './tools/image-tag | cut -d, -f 1 | tr A-Z a-z > .tags',  // values in .tags are used by the next step when pushing the image
                  'cd ./cmd/tempo-serverless',
                  'make build-docker-gcr-binary',
                  'make build-lambda-zip',
                ],
              },
              {
                name: 'deploy-tempo-serverless-gcr',
                image: 'plugins/gcr',
                settings: {
                  repo: 'ops-tools-1203/tempo-serverless',
                  context: './cmd/tempo-serverless/cloud-run',
                  dockerfile: './cmd/tempo-serverless/cloud-run/Dockerfile',
                  json_key: {
                    from_secret: image_upload_ops_tools_secret.name,
                  },
                },
              },
            ] +
            [
              {
                name: 'deploy-tempo-%s-serverless-lambda' % d.env,
                image: 'amazon/aws-cli',
                environment: {
                  AWS_DEFAULT_REGION: 'us-east-2',
                  AWS_ACCESS_KEY_ID: {
                    from_secret: d.access_key_id,
                  },
                  AWS_SECRET_ACCESS_KEY: {
                    from_secret: d.secret_access_key,
                  },
                },
                commands: [
                  'cd ./cmd/tempo-serverless/lambda',
                  'aws s3 cp tempo-serverless*.zip s3://%s' % d.bucket,
                ],
              }

              for d in aws_serverless_deployments
            ],
  },
  // Build and release packages
  // Tested by installing the packages on a systemd container
  pipeline('release') {
    trigger: {
      event: ['tag', 'pull_request'],
    },
    volumes+: [
      {
        name: 'cgroup',
        host: {
          path: '/sys/fs/cgroup',
        },
      },
      {
        name: 'docker',
        host: {
          path: '/var/run/docker.sock',
        },
      },
    ],
    // Launch systemd containers to test the packages
    services: [
      {
        name: 'systemd-debian',
        image: 'jrei/systemd-debian:12',
        volumes: [
          {
            name: 'cgroup',
            path: '/sys/fs/cgroup',
          },
        ],
        privileged: true,
      },
      {
        name: 'systemd-centos',
        image: 'jrei/systemd-centos:8',
        volumes: [
          {
            name: 'cgroup',
            path: '/sys/fs/cgroup',
          },
        ],
        privileged: true,
      },
    ],
    steps+: [
      {
        name: 'fetch',
        image: 'docker:git',
        commands: ['git fetch --tags'],
      },
      {
        name: 'write-key',
        image: 'golang:1.22',
        commands: ['printf "%s" "$NFPM_SIGNING_KEY" > $NFPM_SIGNING_KEY_FILE'],
        environment: {
          NFPM_SIGNING_KEY: { from_secret: gpg_private_key.name },
          NFPM_SIGNING_KEY_FILE: '/drone/src/private-key.key',
        },
      },
      {
        name: 'test release',
        image: 'golang:1.22',
        commands: ['make release-snapshot'],
        environment: {
          NFPM_DEFAULT_PASSPHRASE: { from_secret: gpg_passphrase.name },
          NFPM_SIGNING_KEY_FILE: '/drone/src/private-key.key',
        },
      },
      {
        name: 'test deb package',
        image: 'docker',
        commands: ['./tools/packaging/verify-deb-install.sh'],
        volumes: [
          {
            name: 'docker',
            path: '/var/run/docker.sock',
          },
        ],
        privileged: true,
      },
      {
        name: 'test rpm package',
        image: 'docker',
        commands: ['./tools/packaging/verify-rpm-install.sh'],
        volumes: [
          {
            name: 'docker',
            path: '/var/run/docker.sock',
          },
        ],
        privileged: true,
      },
      {
        name: 'release',
        image: 'golang:1.22',
        commands: ['make release'],
        environment: {
          NFPM_DEFAULT_PASSPHRASE: { from_secret: gpg_passphrase.name },
          NFPM_SIGNING_KEY_FILE: '/drone/src/private-key.key',
        },
        when: {
          event: ['tag'],
        },
      },
    ],
  },
] + [
  docker_username_secret,
  docker_password_secret,
  docker_config_json_secret,
  gh_token_secret,
  tempo_app_id_secret,
  tempo_app_installation_id_secret,
  tempo_app_private_key_secret,
  image_upload_ops_tools_secret,
  aws_dev_access_key_id,
  aws_dev_secret_access_key,
  aws_prod_access_key_id,
  aws_prod_secret_access_key,
  gpg_private_key,
  gpg_passphrase,
]
