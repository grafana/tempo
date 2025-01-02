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

// secret needed to access us.gcr.io in deploy_to_dev()
local docker_config_json_secret = secret('dockerconfigjson', 'secret/data/common/gcr', '.dockerconfigjson');

// secret needed for dep-tools
local gh_token_secret = secret('gh_token', 'infra/data/ci/github/grafanabot', 'pat');
local tempo_app_id_secret = secret('tempo_app_id_secret', 'infra/data/ci/tempo/github-app', 'app-id');
local tempo_app_installation_id_secret = secret('tempo_app_installation_id_secret', 'infra/data/ci/tempo/github-app', 'app-installation-id');
local tempo_app_private_key_secret = secret('tempo_app_private_key_secret', 'infra/data/ci/tempo/github-app', 'app-private-key');

// secret to sign linux packages
local gpg_passphrase = secret('gpg_passphrase', 'infra/data/ci/packages-publish/gpg', 'passphrase');
local gpg_private_key = secret('gpg_private_key', 'infra/data/ci/packages-publish/gpg', 'private-key');

local aws_dev_access_key_id = secret('AWS_ACCESS_KEY_ID-dev', 'infra/data/ci/tempo-dev/aws-credentials-drone', 'access_key_id');
local aws_dev_secret_access_key = secret('AWS_SECRET_ACCESS_KEY-dev', 'infra/data/ci/tempo-dev/aws-credentials-drone', 'secret_access_key');
local aws_prod_access_key_id = secret('AWS_ACCESS_KEY_ID-prod', 'infra/data/ci/tempo-prod/aws-credentials-drone', 'access_key_id');
local aws_prod_secret_access_key = secret('AWS_SECRET_ACCESS_KEY-prod', 'infra/data/ci/tempo-prod/aws-credentials-drone', 'secret_access_key');

//# Steps ##

// the alpine/git image has apk errors when run on aarch64, this is the most recent image that does not have this issue
// https://github.com/alpine-docker/git/issues/35
local alpine_git_image = 'alpine/git:v2.30.2';

//# Pipelines & resources

[
  local ghTokenFilename = '/drone/src/gh-token.txt';
  // Build and release packages
  // Tested by installing the packages on a systemd container
  pipeline('release') {
    trigger: {
      event: ['tag', 'pull_request'],
    },
    image_pull_secrets: [
      docker_config_json_secret.name,
    ],
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
        name: 'Generate GitHub token',
        image: 'us.gcr.io/kubernetes-dev/github-app-secret-writer:latest',
        environment: {
          GITHUB_APP_ID: { from_secret:  tempo_app_id_secret.name },
          GITHUB_APP_INSTALLATION_ID: { from_secret:  tempo_app_installation_id_secret.name },
          GITHUB_APP_PRIVATE_KEY: { from_secret: tempo_app_private_key_secret.name },
        },
        commands: [
          '/usr/bin/github-app-external-token > %s' % ghTokenFilename,
        ],
      },
      {
        name: 'write-key',
        image: 'golang:1.23',
        commands: ['printf "%s" "$NFPM_SIGNING_KEY" > $NFPM_SIGNING_KEY_FILE'],
        environment: {
          NFPM_SIGNING_KEY: { from_secret: gpg_private_key.name },
          NFPM_SIGNING_KEY_FILE: '/drone/src/private-key.key',
        },
      },
      {
        name: 'test release',
        image: 'golang:1.23',
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
        image: 'golang:1.23',
        commands: [
          'export GITHUB_TOKEN=$(cat %s)' % ghTokenFilename,
          'make release'
        ],
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
  aws_dev_access_key_id,
  aws_dev_secret_access_key,
  aws_prod_access_key_id,
  aws_prod_secret_access_key,
  gpg_private_key,
  gpg_passphrase,
]
