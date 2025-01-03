# Lunes

---

**Lunes** is a [Go](http://golang.org) library for parsing localized time strings into `time.Time`.

There's no intention to replace the standard `time` package parsing functions, instead, it acts as wrapper
translating the provided value to English before invoking the `time.Parse` and `time.ParseInLocation`.

It currently supports almost all [CLDR](https://cldr.unicode.org/) core locales (+900 including drafts),
being limited to the **gregorian** calendars.

Once the official Go i18n features for time parsing are ready, it should be replaced.

## Usage

#### Parse

```go
// it's like time.Parse, but with an additional locale parameter to perform the value translation.
// the language argument must be a well-formed BCP 47 language tag, e.g ("en", "en-US") and
// a known locale. If no data is found for the language, it returns ErrUnsupportedLocale.
// If the given locale does not support any layout element specified on the layout argument,
// it results in an ErrUnsupportedLayoutElem error. On the other hand, if the value does not
// match the layout, an ErrLayoutMismatch is returned.
t, err := lunes.Parse("Monday Jan _2 2006 15:04:05", "lunes oct 27 1988 11:53:29", lunes.LocaleEsES)

// parse in specific time zones.
t, err := lunes.ParseInLocation("Monday Jan _2 2006 15:04:05", "lunes oct 27 1988 11:53:29", time.UTC, lunes.LocaleEsES)
```

```go
// creates a new generic locale for the given BCP 47 language tag, using the default CLDR
// gregorian calendars data of the specified language. If the locale is unknown and/or no
// default data is found, it returns ErrUnsupportedLocale.
locale, err := lunes.NewDefaultLocale(lunes.LocaleEsES)

// ParseWithLocale has a better performance for multiple parse operations, as it does not
// need to look up the locale data in each iteration.
for _, val := range valuesToParse {
    t, err := lunes.ParseWithLocale("Monday Jan _2 2006 15:04:05", val, locale)
}
```

#### Translate

```go
// translates the value, without parsing it to time.Time. The language argument must be a
// well-formed BCP 47 language tag, e.g ("en", "en-US") and a known locale. If no data is
// found for the language, it returns ErrUnsupportedLocale.
// If the given locale does not support any layout element specified on the layout argument,
// it results in an ErrUnsupportedLayoutElem error. On the other hand, if the values does not
// match the layout, an ErrLayoutMismatch is returned.
// For the following example, it results in: Friday Jan 27 11:53:29.
str, err := lunes.Translate("Monday Jan _2 15:04:05", "viernes ene 27 11:53:29", lunes.LocaleEsES)

// the translated value is meant to be used with the time package functions
t, err := time.Parse("Monday Jan _2 15:04:05", str)
```

#### Custom Locales

A `lunes.Locale` provides a collection of time layouts values in a specific language.
It is used to provide a map between the time layout elements in foreign language to English.
In oder to use custom locales, the following functions must be implemented:

```go
// Language represents a BCP 47 tag, specifying this locale language.
Language() string

// LongDayNames returns the long day names translations for the week days.
// It must be sorted, starting from Sunday to Saturday, and contains all 7 elements,
// even if one or more days are empty. If this locale does not support this format,
// it should return an empty slice.
LongDayNames() []string

// ShortDayNames returns the short day names translations for the week days.
// It must be sorted, starting from Sunday to Saturday, and contains all 7 elements,
// even if one or more days are empty. If this locale does not support this format,
// it should return an empty slice.
ShortDayNames() []string

// LongMonthNames returns the long day names translations for the months names.
// It must be sorted, starting from January to December, and contains all 12 elements,
// even if one or more months are empty. If this locale does not support this format,
// it should return an empty slice.
LongMonthNames() []string

// ShortMonthNames returns the short day names translations for the months names.
// It must be sorted, starting from January to December, and contains all 12 elements,
// even if one or more months are empty. If this locale does not support this format,
// it should return an empty slice.
ShortMonthNames() []string

// DayPeriods returns the periods of day translations for the AM and PM abbreviations.
// It must be sorted, starting from AM to PM, and contains both elements, even if one
// of them is empty. If this locale does not support this format, it should return an
// empty slice.
DayPeriods() []string
```

Custom locales can be used with the `lunes.ParseWithLocale`, `lunes.ParseInLocationWithLocale`, and `lunes.TranslateWithLocale`
functions:

```go
locale := &CustomLocale{}

// It's like Parse, but instead of receiving a BCP 47 language tag argument, it receives a lunes.Locale
t, err := lunes.ParseWithLocale("Monday Jan _2 2006 15:04:05", "lunes oct 27 1988 11:53:29", locale)

// It's like ParseInLocation, but instead of receiving a BCP 47 language tag argument, it receives a lunes.Locale
t, err := lunes.ParseInLocationWithLocale("Monday Jan _2 2006 15:04:05", "lunes oct 27 1988 11:53:29", time.UTC, locale)

// It's like Translate, but instead of receiving a BCP 47 language tag argument, it receives a lunes.Locale
t, err := lunes.TranslateWithLocale("Monday Jan _2 2006 15:04:05", "lunes oct 27 1988 11:53:29", locale)
```

## Benchmarks

Comparing to [github.com/goodsign/monday](https://github.com/goodsign/monday)

```
BenchmarkTranslate-10                    	 3850832	       303.2 ns/op	     220 B/op	       5 allocs/op
BenchmarkTranslateWithLocale-10          	 5149981	       235.1 ns/op	      76 B/op	       4 allocs/op
BenchmarkParse-10                        	 2811612	       428.1 ns/op	     220 B/op	       5 allocs/op
BenchmarkParseInLocation-10              	 2792997	       439.2 ns/op	     220 B/op	       5 allocs/op
BenchmarkParseWithLocale-10              	 3268903	       362.7 ns/op	      76 B/op	       4 allocs/op
BenchmarkParseInLocationWithLocale-10    	 2974732	       390.2 ns/op	      76 B/op	       4 allocs/op
BenchmarkParseMonday-10                  	  213014	      5584 ns/op	    3754 B/op	     117 allocs/op
BenchmarkParseInLocationMonday-10        	  211826	      5593 ns/op	    3754 B/op	     117 allocs/op
```

### Usage notes

- It currently supports the following time layout replacements:
  - Short days names (`Mon`)
  - Long days names (`Monday`)
  - Short month names (`Jan`)
  - Long month names (`January`)
  - Day periods (`PM`)
- Translations are auto-generated, and it might be inconsistent depending on the CLDR locale [stage](https://cldr.unicode.org/index/process).
- A few locales does not support (or are missing) translations for specific layout elements (short/long days/month names or day periods), in that case,
  an ErrUnsupportedLayoutElem will be reported.

