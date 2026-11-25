# preferences

grafonnet.preferences

## Index

* [`fn withCookiePreferences(value)`](#fn-withcookiepreferences)
* [`fn withCookiePreferencesMixin(value)`](#fn-withcookiepreferencesmixin)
* [`fn withHomeDashboardUID(value)`](#fn-withhomedashboarduid)
* [`fn withLanguage(value)`](#fn-withlanguage)
* [`fn withNavbar(value)`](#fn-withnavbar)
* [`fn withNavbarMixin(value)`](#fn-withnavbarmixin)
* [`fn withQueryHistory(value)`](#fn-withqueryhistory)
* [`fn withQueryHistoryMixin(value)`](#fn-withqueryhistorymixin)
* [`fn withTheme(value)`](#fn-withtheme)
* [`fn withTimezone(value)`](#fn-withtimezone)
* [`fn withWeekStart(value)`](#fn-withweekstart)
* [`obj cookiePreferences`](#obj-cookiepreferences)
  * [`fn withAnalytics(value)`](#fn-cookiepreferenceswithanalytics)
  * [`fn withAnalyticsMixin(value)`](#fn-cookiepreferenceswithanalyticsmixin)
  * [`fn withFunctional(value)`](#fn-cookiepreferenceswithfunctional)
  * [`fn withFunctionalMixin(value)`](#fn-cookiepreferenceswithfunctionalmixin)
  * [`fn withPerformance(value)`](#fn-cookiepreferenceswithperformance)
  * [`fn withPerformanceMixin(value)`](#fn-cookiepreferenceswithperformancemixin)
* [`obj navbar`](#obj-navbar)
  * [`fn withBookmarkUrls(value)`](#fn-navbarwithbookmarkurls)
  * [`fn withBookmarkUrlsMixin(value)`](#fn-navbarwithbookmarkurlsmixin)
* [`obj queryHistory`](#obj-queryhistory)
  * [`fn withHomeTab(value)`](#fn-queryhistorywithhometab)

## Fields

### fn withCookiePreferences

```jsonnet
withCookiePreferences(value)
```

PARAMETERS:

* **value** (`object`)

Cookie preferences
### fn withCookiePreferencesMixin

```jsonnet
withCookiePreferencesMixin(value)
```

PARAMETERS:

* **value** (`object`)

Cookie preferences
### fn withHomeDashboardUID

```jsonnet
withHomeDashboardUID(value)
```

PARAMETERS:

* **value** (`string`)

UID for the home dashboard
### fn withLanguage

```jsonnet
withLanguage(value)
```

PARAMETERS:

* **value** (`string`)

Selected language (beta)
### fn withNavbar

```jsonnet
withNavbar(value)
```

PARAMETERS:

* **value** (`object`)

Navigation preferences
### fn withNavbarMixin

```jsonnet
withNavbarMixin(value)
```

PARAMETERS:

* **value** (`object`)

Navigation preferences
### fn withQueryHistory

```jsonnet
withQueryHistory(value)
```

PARAMETERS:

* **value** (`object`)

Explore query history preferences
### fn withQueryHistoryMixin

```jsonnet
withQueryHistoryMixin(value)
```

PARAMETERS:

* **value** (`object`)

Explore query history preferences
### fn withTheme

```jsonnet
withTheme(value)
```

PARAMETERS:

* **value** (`string`)

light, dark, empty is default
### fn withTimezone

```jsonnet
withTimezone(value)
```

PARAMETERS:

* **value** (`string`)

The timezone selection
TODO: this should use the timezone defined in common
### fn withWeekStart

```jsonnet
withWeekStart(value)
```

PARAMETERS:

* **value** (`string`)

day of the week (sunday, monday, etc)
### obj cookiePreferences


#### fn cookiePreferences.withAnalytics

```jsonnet
cookiePreferences.withAnalytics(value)
```

PARAMETERS:

* **value** (`object`)


#### fn cookiePreferences.withAnalyticsMixin

```jsonnet
cookiePreferences.withAnalyticsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn cookiePreferences.withFunctional

```jsonnet
cookiePreferences.withFunctional(value)
```

PARAMETERS:

* **value** (`object`)


#### fn cookiePreferences.withFunctionalMixin

```jsonnet
cookiePreferences.withFunctionalMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn cookiePreferences.withPerformance

```jsonnet
cookiePreferences.withPerformance(value)
```

PARAMETERS:

* **value** (`object`)


#### fn cookiePreferences.withPerformanceMixin

```jsonnet
cookiePreferences.withPerformanceMixin(value)
```

PARAMETERS:

* **value** (`object`)


### obj navbar


#### fn navbar.withBookmarkUrls

```jsonnet
navbar.withBookmarkUrls(value)
```

PARAMETERS:

* **value** (`array`)


#### fn navbar.withBookmarkUrlsMixin

```jsonnet
navbar.withBookmarkUrlsMixin(value)
```

PARAMETERS:

* **value** (`array`)


### obj queryHistory


#### fn queryHistory.withHomeTab

```jsonnet
queryHistory.withHomeTab(value)
```

PARAMETERS:

* **value** (`string`)

one of: '' | 'query' | 'starred';