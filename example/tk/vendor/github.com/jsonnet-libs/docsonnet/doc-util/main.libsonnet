{
  local d = self,

  '#': d.pkg(
    name='d',
    url='github.com/jsonnet-libs/docsonnet/doc-util',
    help=|||
      `doc-util` provides a Jsonnet interface for `docsonnet`,
       a Jsonnet API doc generator that uses structured data instead of comments.
    |||
  ),

  package:: {
    '#new':: d.fn('new creates a new package with given `name`, `import` URL and `help` text', [d.arg('name', d.T.string), d.arg('url', d.T.string), d.arg('help', d.T.string)]),
    new(name, url, help):: {
      name: name,
      'import': url,
      help: help,
    },
  },

  '#pkg':: self.package['#new'] + d.func.withHelp('`new` is a shorthand for `package.new`'),
  pkg:: self.package.new,

  '#object': d.obj('Utilities for documenting Jsonnet objects (`{ }`).'),
  object:: {
    '#new': d.fn('new creates a new object, optionally with description and fields', [d.arg('help', d.T.string), d.arg('fields', d.T.object)]),
    new(help='', fields={}):: { object: {
      help: help,
      fields: fields,
    } },

    '#withFields': d.fn('The `withFields` modifier overrides the fields property of an already created object', [d.arg('fields', d.T.object)]),
    withFields(fields):: { object+: {
      fields: fields,
    } },
  },

  '#obj': self.object['#new'] + d.func.withHelp('`obj` is a shorthand for `object.new`'),
  obj:: self.object.new,

  '#func': d.obj('Utilities for documenting Jsonnet methods (functions of objects)'),
  func:: {
    '#new': d.fn('new creates a new function, optionally with description and arguments', [d.arg('help', d.T.string), d.arg('args', d.T.array)]),
    new(help='', args=[]):: { 'function': {
      help: help,
      args: args,
    } },

    '#withHelp': d.fn('The `withHelp` modifier overrides the help text of that function', [d.arg('help', d.T.string)]),
    withHelp(help):: { 'function'+: {
      help: help,
    } },

    '#withArgs': d.fn('The `withArgs` modifier overrides the arguments of that function', [d.arg('args', d.T.array)]),
    withArgs(args):: { 'function'+: {
      args: args,
    } },
  },

  '#fn': self.func['#new'] + d.func.withHelp('`fn` is a shorthand for `func.new`'),
  fn:: self.func.new,

  '#argument': d.obj('Utilities for creating function arguments'),
  argument:: {
    '#new': d.fn('new creates a new function argument, taking the name, the type and optionally a default value', [d.arg('name', d.T.string), d.arg('type', d.T.string), d.arg('default', d.T.any)]),
    new(name, type, default=null): {
      name: name,
      type: type,
      default: default,
    },
  },
  '#arg': self.argument['#new'] + self.func.withHelp('`arg` is a shorthand for `argument.new`'),
  arg:: self.argument.new,

  "#value": d.obj("Utilities for documenting plain Jsonnet values (primitives)"),
  value:: {
    "#new": d.fn("new creates a new object of given type, optionally with description and default value", [d.arg("type", d.T.string), d.arg("help", d.T.string), d.arg("default", d.T.any)]),
    new(type, help='', default=null): { 'value': {
      help: help,
      type: type,
      default: default,
    } }
  },
  '#val': self.value['#new'] + self.func.withHelp('`val` is a shorthand for `value.new`'),
  val: self.value.new,

  // T contains constants for the Jsonnet types
  T:: {
    string: 'string',

    number: 'number',
    int: self.number,
    integer: self.number,

    boolean: 'bool',
    bool: self.boolean,

    object: 'object',
    array: 'array',
    any: 'any',

    'null': "null",
    nil: self["null"],

    func: 'function',
    'function': self.func,
  },
}
