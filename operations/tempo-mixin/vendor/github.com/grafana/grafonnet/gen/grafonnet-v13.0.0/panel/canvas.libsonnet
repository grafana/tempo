// This file is generated, do not manually edit.
(import '../panel.libsonnet')
+ {
  '#': { help: 'grafonnet.panel.canvas', name: 'canvas' },
  panelOptions+:
    {
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'canvas',
      },
    },
  options+:
    {
      '#withInfinitePan': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Enable infinite pan' } },
      withInfinitePan(value=true): {
        options+: {
          infinitePan: value,
        },
      },
      '#withInlineEditing': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Enable inline editing' } },
      withInlineEditing(value=true): {
        options+: {
          inlineEditing: value,
        },
      },
      '#withPanZoom': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Enable pan and zoom' } },
      withPanZoom(value=true): {
        options+: {
          panZoom: value,
        },
      },
      '#withRoot': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The root element of canvas (frame), where all canvas elements are nested\nTODO: Figure out how to define a default value for this' } },
      withRoot(value): {
        options+: {
          root: value,
        },
      },
      '#withRootMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The root element of canvas (frame), where all canvas elements are nested\nTODO: Figure out how to define a default value for this' } },
      withRootMixin(value): {
        options+: {
          root+: value,
        },
      },
      root+:
        {
          '#withElements': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'The list of canvas elements attached to the root element' } },
          withElements(value): {
            options+: {
              root+: {
                elements:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withElementsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'The list of canvas elements attached to the root element' } },
          withElementsMixin(value): {
            options+: {
              root+: {
                elements+:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          elements+:
            {
              '#': { help: '', name: 'elements' },
              '#withBackground': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withBackground(value): {
                background: value,
              },
              '#withBackgroundMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withBackgroundMixin(value): {
                background+: value,
              },
              background+:
                {
                  '#withColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withColor(value): {
                    background+: {
                      color: value,
                    },
                  },
                  '#withColorMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withColorMixin(value): {
                    background+: {
                      color+: value,
                    },
                  },
                  color+:
                    {
                      '#withField': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'fixed: T -- will be added by each element' } },
                      withField(value): {
                        background+: {
                          color+: {
                            field: value,
                          },
                        },
                      },
                      '#withFixed': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'color value' } },
                      withFixed(value): {
                        background+: {
                          color+: {
                            fixed: value,
                          },
                        },
                      },
                    },
                  '#withImage': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Links to a resource (image/svg path)' } },
                  withImage(value): {
                    background+: {
                      image: value,
                    },
                  },
                  '#withImageMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Links to a resource (image/svg path)' } },
                  withImageMixin(value): {
                    background+: {
                      image+: value,
                    },
                  },
                  image+:
                    {
                      '#withField': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'fixed: T -- will be added by each element' } },
                      withField(value): {
                        background+: {
                          image+: {
                            field: value,
                          },
                        },
                      },
                      '#withFixed': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withFixed(value): {
                        background+: {
                          image+: {
                            fixed: value,
                          },
                        },
                      },
                      '#withMode': { 'function': { args: [{ default: null, enums: ['fixed', 'field', 'mapping'], name: 'value', type: ['string'] }], help: '' } },
                      withMode(value): {
                        background+: {
                          image+: {
                            mode: value,
                          },
                        },
                      },
                    },
                  '#withSize': { 'function': { args: [{ default: null, enums: ['original', 'contain', 'cover', 'fill', 'tile'], name: 'value', type: ['string'] }], help: '' } },
                  withSize(value): {
                    background+: {
                      size: value,
                    },
                  },
                },
              '#withBorder': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withBorder(value): {
                border: value,
              },
              '#withBorderMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withBorderMixin(value): {
                border+: value,
              },
              border+:
                {
                  '#withColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withColor(value): {
                    border+: {
                      color: value,
                    },
                  },
                  '#withColorMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withColorMixin(value): {
                    border+: {
                      color+: value,
                    },
                  },
                  color+:
                    {
                      '#withField': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'fixed: T -- will be added by each element' } },
                      withField(value): {
                        border+: {
                          color+: {
                            field: value,
                          },
                        },
                      },
                      '#withFixed': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'color value' } },
                      withFixed(value): {
                        border+: {
                          color+: {
                            fixed: value,
                          },
                        },
                      },
                    },
                  '#withRadius': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withRadius(value): {
                    border+: {
                      radius: value,
                    },
                  },
                  '#withWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withWidth(value): {
                    border+: {
                      width: value,
                    },
                  },
                },
              '#withConfig': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO: figure out how to define this (element config(s))' } },
              withConfig(value): {
                config: value,
              },
              '#withConfigMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO: figure out how to define this (element config(s))' } },
              withConfigMixin(value): {
                config+: value,
              },
              '#withConnections': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withConnections(value): {
                connections:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
              '#withConnectionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withConnectionsMixin(value): {
                connections+:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
              connections+:
                {
                  '#': { help: '', name: 'connections' },
                  '#withColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withColor(value): {
                    color: value,
                  },
                  '#withColorMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withColorMixin(value): {
                    color+: value,
                  },
                  color+:
                    {
                      '#withField': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'fixed: T -- will be added by each element' } },
                      withField(value): {
                        color+: {
                          field: value,
                        },
                      },
                      '#withFixed': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'color value' } },
                      withFixed(value): {
                        color+: {
                          fixed: value,
                        },
                      },
                    },
                  '#withPath': { 'function': { args: [{ default: 'straight', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withPath(value='straight'): {
                    path: value,
                  },
                  '#withPathMixin': { 'function': { args: [{ default: 'straight', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withPathMixin(value): {
                    path+: value,
                  },
                  path+:
                    {
                      '#withConnectionPath': { 'function': { args: [{ default: null, enums: ['straight'], name: 'value', type: ['string'] }], help: '' } },
                      withConnectionPath(value): {
                        path+: {
                          ConnectionPath: value,
                        },
                      },
                    },
                  '#withSize': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSize(value): {
                    size: value,
                  },
                  '#withSizeMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSizeMixin(value): {
                    size+: value,
                  },
                  size+:
                    {
                      '#withField': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'fixed: T -- will be added by each element' } },
                      withField(value): {
                        size+: {
                          field: value,
                        },
                      },
                      '#withFixed': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withFixed(value): {
                        size+: {
                          fixed: value,
                        },
                      },
                      '#withMax': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withMax(value): {
                        size+: {
                          max: value,
                        },
                      },
                      '#withMin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withMin(value): {
                        size+: {
                          min: value,
                        },
                      },
                      '#withMode': { 'function': { args: [{ default: null, enums: ['linear', 'quad'], name: 'value', type: ['string'] }], help: '| *"linear"' } },
                      withMode(value): {
                        size+: {
                          mode: value,
                        },
                      },
                    },
                  '#withSource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSource(value): {
                    source: value,
                  },
                  '#withSourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSourceMixin(value): {
                    source+: value,
                  },
                  source+:
                    {
                      '#withX': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withX(value): {
                        source+: {
                          x: value,
                        },
                      },
                      '#withY': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withY(value): {
                        source+: {
                          y: value,
                        },
                      },
                    },
                  '#withSourceOriginal': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSourceOriginal(value): {
                    sourceOriginal: value,
                  },
                  '#withSourceOriginalMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSourceOriginalMixin(value): {
                    sourceOriginal+: value,
                  },
                  sourceOriginal+:
                    {
                      '#withX': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withX(value): {
                        sourceOriginal+: {
                          x: value,
                        },
                      },
                      '#withY': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withY(value): {
                        sourceOriginal+: {
                          y: value,
                        },
                      },
                    },
                  '#withTarget': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withTarget(value): {
                    target: value,
                  },
                  '#withTargetMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withTargetMixin(value): {
                    target+: value,
                  },
                  target+:
                    {
                      '#withX': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withX(value): {
                        target+: {
                          x: value,
                        },
                      },
                      '#withY': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withY(value): {
                        target+: {
                          y: value,
                        },
                      },
                    },
                  '#withTargetName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withTargetName(value): {
                    targetName: value,
                  },
                  '#withTargetOriginal': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withTargetOriginal(value): {
                    targetOriginal: value,
                  },
                  '#withTargetOriginalMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withTargetOriginalMixin(value): {
                    targetOriginal+: value,
                  },
                  targetOriginal+:
                    {
                      '#withX': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withX(value): {
                        targetOriginal+: {
                          x: value,
                        },
                      },
                      '#withY': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withY(value): {
                        targetOriginal+: {
                          y: value,
                        },
                      },
                    },
                  '#withVertices': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withVertices(value): {
                    vertices:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                  '#withVerticesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withVerticesMixin(value): {
                    vertices+:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                  vertices+:
                    {
                      '#': { help: '', name: 'vertices' },
                      '#withX': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withX(value): {
                        x: value,
                      },
                      '#withY': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withY(value): {
                        y: value,
                      },
                    },
                },
              '#withConstraint': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withConstraint(value): {
                constraint: value,
              },
              '#withConstraintMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withConstraintMixin(value): {
                constraint+: value,
              },
              constraint+:
                {
                  '#withHorizontal': { 'function': { args: [{ default: null, enums: ['left', 'right', 'leftright', 'center', 'scale'], name: 'value', type: ['string'] }], help: '' } },
                  withHorizontal(value): {
                    constraint+: {
                      horizontal: value,
                    },
                  },
                  '#withVertical': { 'function': { args: [{ default: null, enums: ['top', 'bottom', 'topbottom', 'center', 'scale'], name: 'value', type: ['string'] }], help: '' } },
                  withVertical(value): {
                    constraint+: {
                      vertical: value,
                    },
                  },
                },
              '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
              withName(value): {
                name: value,
              },
              '#withPlacement': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withPlacement(value): {
                placement: value,
              },
              '#withPlacementMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withPlacementMixin(value): {
                placement+: value,
              },
              placement+:
                {
                  '#withBottom': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withBottom(value): {
                    placement+: {
                      bottom: value,
                    },
                  },
                  '#withHeight': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withHeight(value): {
                    placement+: {
                      height: value,
                    },
                  },
                  '#withLeft': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withLeft(value): {
                    placement+: {
                      left: value,
                    },
                  },
                  '#withRight': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withRight(value): {
                    placement+: {
                      right: value,
                    },
                  },
                  '#withRotation': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withRotation(value): {
                    placement+: {
                      rotation: value,
                    },
                  },
                  '#withTop': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withTop(value): {
                    placement+: {
                      top: value,
                    },
                  },
                  '#withWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withWidth(value): {
                    placement+: {
                      width: value,
                    },
                  },
                },
              '#withType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
              withType(value): {
                type: value,
              },
            },
          '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Name of the root element' } },
          withName(value): {
            options+: {
              root+: {
                name: value,
              },
            },
          },
          '#withType': { 'function': { args: [], help: 'Type of root element (frame)' } },
          withType(): {
            options+: {
              root+: {
                type: 'frame',
              },
            },
          },
        },
      '#withShowAdvancedTypes': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Show all available element types' } },
      withShowAdvancedTypes(value=true): {
        options+: {
          showAdvancedTypes: value,
        },
      },
    },
}
+ {
  panelOptions+: {
    '#withType':: {
      ignore: true,
    },
  },
}
