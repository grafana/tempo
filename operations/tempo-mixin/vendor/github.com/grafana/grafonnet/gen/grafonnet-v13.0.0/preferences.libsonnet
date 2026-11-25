// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.preferences', name: 'preferences' },
  '#withCookiePreferences': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Cookie preferences' } },
  withCookiePreferences(value): {
    cookiePreferences: value,
  },
  '#withCookiePreferencesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Cookie preferences' } },
  withCookiePreferencesMixin(value): {
    cookiePreferences+: value,
  },
  cookiePreferences+:
    {
      '#withAnalytics': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withAnalytics(value): {
        cookiePreferences+: {
          analytics: value,
        },
      },
      '#withAnalyticsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withAnalyticsMixin(value): {
        cookiePreferences+: {
          analytics+: value,
        },
      },
      '#withFunctional': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withFunctional(value): {
        cookiePreferences+: {
          functional: value,
        },
      },
      '#withFunctionalMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withFunctionalMixin(value): {
        cookiePreferences+: {
          functional+: value,
        },
      },
      '#withPerformance': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withPerformance(value): {
        cookiePreferences+: {
          performance: value,
        },
      },
      '#withPerformanceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withPerformanceMixin(value): {
        cookiePreferences+: {
          performance+: value,
        },
      },
    },
  '#withHomeDashboardUID': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'UID for the home dashboard' } },
  withHomeDashboardUID(value): {
    homeDashboardUID: value,
  },
  '#withLanguage': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Selected language (beta)' } },
  withLanguage(value): {
    language: value,
  },
  '#withNavbar': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Navigation preferences' } },
  withNavbar(value): {
    navbar: value,
  },
  '#withNavbarMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Navigation preferences' } },
  withNavbarMixin(value): {
    navbar+: value,
  },
  navbar+:
    {
      '#withBookmarkUrls': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withBookmarkUrls(value): {
        navbar+: {
          bookmarkUrls:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withBookmarkUrlsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withBookmarkUrlsMixin(value): {
        navbar+: {
          bookmarkUrls+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
    },
  '#withQueryHistory': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Explore query history preferences' } },
  withQueryHistory(value): {
    queryHistory: value,
  },
  '#withQueryHistoryMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Explore query history preferences' } },
  withQueryHistoryMixin(value): {
    queryHistory+: value,
  },
  queryHistory+:
    {
      '#withHomeTab': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: "one of: '' | 'query' | 'starred';" } },
      withHomeTab(value): {
        queryHistory+: {
          homeTab: value,
        },
      },
    },
  '#withTheme': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'light, dark, empty is default' } },
  withTheme(value): {
    theme: value,
  },
  '#withTimezone': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The timezone selection\nTODO: this should use the timezone defined in common' } },
  withTimezone(value): {
    timezone: value,
  },
  '#withWeekStart': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'day of the week (sunday, monday, etc)' } },
  withWeekStart(value): {
    weekStart: value,
  },
}
