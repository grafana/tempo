// This file is generated, do not manually edit.
(import '../metadata.libsonnet')
+ {
  '#': { help: 'grafonnet.apps.preferences.v1alpha1', name: 'v1alpha1' },
  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withSpec(value): {
    spec: value,
  },
  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withSpecMixin(value): {
    spec+: value,
  },
  spec+:
    {
      '#withCookiePreferences': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Cookie preferences' } },
      withCookiePreferences(value): {
        spec+: {
          cookiePreferences: value,
        },
      },
      '#withCookiePreferencesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Cookie preferences' } },
      withCookiePreferencesMixin(value): {
        spec+: {
          cookiePreferences+: value,
        },
      },
      cookiePreferences+:
        {
          '#withAnalytics': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withAnalytics(value): {
            spec+: {
              cookiePreferences+: {
                analytics: value,
              },
            },
          },
          '#withAnalyticsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withAnalyticsMixin(value): {
            spec+: {
              cookiePreferences+: {
                analytics+: value,
              },
            },
          },
          '#withFunctional': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withFunctional(value): {
            spec+: {
              cookiePreferences+: {
                functional: value,
              },
            },
          },
          '#withFunctionalMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withFunctionalMixin(value): {
            spec+: {
              cookiePreferences+: {
                functional+: value,
              },
            },
          },
          '#withPerformance': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withPerformance(value): {
            spec+: {
              cookiePreferences+: {
                performance: value,
              },
            },
          },
          '#withPerformanceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withPerformanceMixin(value): {
            spec+: {
              cookiePreferences+: {
                performance+: value,
              },
            },
          },
        },
      '#withHomeDashboardUID': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'UID for the home dashboard' } },
      withHomeDashboardUID(value): {
        spec+: {
          homeDashboardUID: value,
        },
      },
      '#withLanguage': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Selected language (beta)' } },
      withLanguage(value): {
        spec+: {
          language: value,
        },
      },
      '#withNavbar': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Navigation preferences' } },
      withNavbar(value): {
        spec+: {
          navbar: value,
        },
      },
      '#withNavbarMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Navigation preferences' } },
      withNavbarMixin(value): {
        spec+: {
          navbar+: value,
        },
      },
      navbar+:
        {
          '#withBookmarkUrls': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
          withBookmarkUrls(value): {
            spec+: {
              navbar+: {
                bookmarkUrls:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withBookmarkUrlsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
          withBookmarkUrlsMixin(value): {
            spec+: {
              navbar+: {
                bookmarkUrls+:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
        },
      '#withQueryHistory': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Explore query history preferences' } },
      withQueryHistory(value): {
        spec+: {
          queryHistory: value,
        },
      },
      '#withQueryHistoryMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Explore query history preferences' } },
      withQueryHistoryMixin(value): {
        spec+: {
          queryHistory+: value,
        },
      },
      queryHistory+:
        {
          '#withHomeTab': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: "one of: '' | 'query' | 'starred';" } },
          withHomeTab(value): {
            spec+: {
              queryHistory+: {
                homeTab: value,
              },
            },
          },
        },
      '#withRegionalFormat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Selected locale (beta)' } },
      withRegionalFormat(value): {
        spec+: {
          regionalFormat: value,
        },
      },
      '#withTheme': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'light, dark, empty is default' } },
      withTheme(value): {
        spec+: {
          theme: value,
        },
      },
      '#withTimezone': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The timezone selection\nTODO: this should use the timezone defined in common' } },
      withTimezone(value): {
        spec+: {
          timezone: value,
        },
      },
      '#withWeekStart': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'day of the week (sunday, monday, etc)' } },
      withWeekStart(value): {
        spec+: {
          weekStart: value,
        },
      },
    },
}
+ {
  '#new': {
    'function': {
      args: [
        {
          default: null,
          enums: null,
          name: 'name',
          type: 'string',
        },
      ],
      help: 'Creates a new preferences.Preferences resource.',
    },
  },
  new: function(name) {
    apiVersion: 'preferences.grafana.app/v1alpha1',
    kind: 'Preferences',
    metadata+: { name: name },
  },
  '#withApiVersion': {
    'function': {
      args: [

      ],
      help: "set the resource's apiVersion",
    },
  },
  withApiVersion: function()
    {
      apiVersion: 'preferences.grafana.app/v1alpha1',
    },
  '#withKind': {
    'function': {
      args: [

      ],
      help: "set the resource's kind",
    },
  },
  withKind: function()
    {
      kind: 'Preferences',
    },
}
