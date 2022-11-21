local pipeline(name, steps) = {
  kind: 'pipeline',
  name: name,
  steps: steps,
  trigger: {
    event: {
      include: ['pull_request'],
    },
  },
};

local run(name, commands) = {
  name: name,
  image: 'golang:1.18',
  commands: commands,
};

[
  pipeline('build', [
    run('lint-fmt', [
      'make install-ci-deps',
      'make lint-fmt',
      'make lint-mixins',
    ]),
  ]),
]
