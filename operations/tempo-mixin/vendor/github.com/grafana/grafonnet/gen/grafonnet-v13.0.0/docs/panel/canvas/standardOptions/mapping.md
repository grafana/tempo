# mapping



## Index

* [`obj RangeMap`](#obj-rangemap)
  * [`fn withOptions(value)`](#fn-rangemapwithoptions)
  * [`fn withOptionsMixin(value)`](#fn-rangemapwithoptionsmixin)
  * [`fn withType(value="range")`](#fn-rangemapwithtype)
  * [`fn withTypeMixin(value="range")`](#fn-rangemapwithtypemixin)
  * [`obj options`](#obj-rangemapoptions)
    * [`fn withFrom(value)`](#fn-rangemapoptionswithfrom)
    * [`fn withResult(value)`](#fn-rangemapoptionswithresult)
    * [`fn withResultMixin(value)`](#fn-rangemapoptionswithresultmixin)
    * [`fn withTo(value)`](#fn-rangemapoptionswithto)
    * [`obj result`](#obj-rangemapoptionsresult)
      * [`fn withColor(value)`](#fn-rangemapoptionsresultwithcolor)
      * [`fn withIcon(value)`](#fn-rangemapoptionsresultwithicon)
      * [`fn withIndex(value)`](#fn-rangemapoptionsresultwithindex)
      * [`fn withText(value)`](#fn-rangemapoptionsresultwithtext)
  * [`obj type`](#obj-rangemaptype)
    * [`fn withMappingType(value)`](#fn-rangemaptypewithmappingtype)
* [`obj RegexMap`](#obj-regexmap)
  * [`fn withOptions(value)`](#fn-regexmapwithoptions)
  * [`fn withOptionsMixin(value)`](#fn-regexmapwithoptionsmixin)
  * [`fn withType(value="regex")`](#fn-regexmapwithtype)
  * [`fn withTypeMixin(value="regex")`](#fn-regexmapwithtypemixin)
  * [`obj options`](#obj-regexmapoptions)
    * [`fn withPattern(value)`](#fn-regexmapoptionswithpattern)
    * [`fn withResult(value)`](#fn-regexmapoptionswithresult)
    * [`fn withResultMixin(value)`](#fn-regexmapoptionswithresultmixin)
    * [`obj result`](#obj-regexmapoptionsresult)
      * [`fn withColor(value)`](#fn-regexmapoptionsresultwithcolor)
      * [`fn withIcon(value)`](#fn-regexmapoptionsresultwithicon)
      * [`fn withIndex(value)`](#fn-regexmapoptionsresultwithindex)
      * [`fn withText(value)`](#fn-regexmapoptionsresultwithtext)
  * [`obj type`](#obj-regexmaptype)
    * [`fn withMappingType(value)`](#fn-regexmaptypewithmappingtype)
* [`obj SpecialValueMap`](#obj-specialvaluemap)
  * [`fn withOptions(value)`](#fn-specialvaluemapwithoptions)
  * [`fn withOptionsMixin(value)`](#fn-specialvaluemapwithoptionsmixin)
  * [`fn withType(value="special")`](#fn-specialvaluemapwithtype)
  * [`fn withTypeMixin(value="special")`](#fn-specialvaluemapwithtypemixin)
  * [`obj options`](#obj-specialvaluemapoptions)
    * [`fn withMatch(value)`](#fn-specialvaluemapoptionswithmatch)
    * [`fn withResult(value)`](#fn-specialvaluemapoptionswithresult)
    * [`fn withResultMixin(value)`](#fn-specialvaluemapoptionswithresultmixin)
    * [`obj result`](#obj-specialvaluemapoptionsresult)
      * [`fn withColor(value)`](#fn-specialvaluemapoptionsresultwithcolor)
      * [`fn withIcon(value)`](#fn-specialvaluemapoptionsresultwithicon)
      * [`fn withIndex(value)`](#fn-specialvaluemapoptionsresultwithindex)
      * [`fn withText(value)`](#fn-specialvaluemapoptionsresultwithtext)
  * [`obj type`](#obj-specialvaluemaptype)
    * [`fn withMappingType(value)`](#fn-specialvaluemaptypewithmappingtype)
* [`obj ValueMap`](#obj-valuemap)
  * [`fn withOptions(value)`](#fn-valuemapwithoptions)
  * [`fn withOptionsMixin(value)`](#fn-valuemapwithoptionsmixin)
  * [`fn withType(value="value")`](#fn-valuemapwithtype)
  * [`fn withTypeMixin(value="value")`](#fn-valuemapwithtypemixin)
  * [`obj type`](#obj-valuemaptype)
    * [`fn withMappingType(value)`](#fn-valuemaptypewithmappingtype)

## Fields

### obj RangeMap


#### fn RangeMap.withOptions

```jsonnet
RangeMap.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

Range to match against and the result to apply when the value is within the range
#### fn RangeMap.withOptionsMixin

```jsonnet
RangeMap.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Range to match against and the result to apply when the value is within the range
#### fn RangeMap.withType

```jsonnet
RangeMap.withType(value="range")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"range"`


#### fn RangeMap.withTypeMixin

```jsonnet
RangeMap.withTypeMixin(value="range")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"range"`


#### obj RangeMap.options


##### fn RangeMap.options.withFrom

```jsonnet
RangeMap.options.withFrom(value)
```

PARAMETERS:

* **value** (`number`)

Min value of the range. It can be null which means -Infinity
##### fn RangeMap.options.withResult

```jsonnet
RangeMap.options.withResult(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
##### fn RangeMap.options.withResultMixin

```jsonnet
RangeMap.options.withResultMixin(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
##### fn RangeMap.options.withTo

```jsonnet
RangeMap.options.withTo(value)
```

PARAMETERS:

* **value** (`number`)

Max value of the range. It can be null which means +Infinity
##### obj RangeMap.options.result


###### fn RangeMap.options.result.withColor

```jsonnet
RangeMap.options.result.withColor(value)
```

PARAMETERS:

* **value** (`string`)

Text to use when the value matches
###### fn RangeMap.options.result.withIcon

```jsonnet
RangeMap.options.result.withIcon(value)
```

PARAMETERS:

* **value** (`string`)

Icon to display when the value matches. Only specific visualizations.
###### fn RangeMap.options.result.withIndex

```jsonnet
RangeMap.options.result.withIndex(value)
```

PARAMETERS:

* **value** (`integer`)

Position in the mapping array. Only used internally.
###### fn RangeMap.options.result.withText

```jsonnet
RangeMap.options.result.withText(value)
```

PARAMETERS:

* **value** (`string`)

Text to display when the value matches
#### obj RangeMap.type


##### fn RangeMap.type.withMappingType

```jsonnet
RangeMap.type.withMappingType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"value"`, `"range"`, `"regex"`, `"special"`

Supported value mapping types
`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.
`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.
`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.
`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.
### obj RegexMap


#### fn RegexMap.withOptions

```jsonnet
RegexMap.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

Regular expression to match against and the result to apply when the value matches the regex
#### fn RegexMap.withOptionsMixin

```jsonnet
RegexMap.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Regular expression to match against and the result to apply when the value matches the regex
#### fn RegexMap.withType

```jsonnet
RegexMap.withType(value="regex")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"regex"`


#### fn RegexMap.withTypeMixin

```jsonnet
RegexMap.withTypeMixin(value="regex")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"regex"`


#### obj RegexMap.options


##### fn RegexMap.options.withPattern

```jsonnet
RegexMap.options.withPattern(value)
```

PARAMETERS:

* **value** (`string`)

Regular expression to match against
##### fn RegexMap.options.withResult

```jsonnet
RegexMap.options.withResult(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
##### fn RegexMap.options.withResultMixin

```jsonnet
RegexMap.options.withResultMixin(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
##### obj RegexMap.options.result


###### fn RegexMap.options.result.withColor

```jsonnet
RegexMap.options.result.withColor(value)
```

PARAMETERS:

* **value** (`string`)

Text to use when the value matches
###### fn RegexMap.options.result.withIcon

```jsonnet
RegexMap.options.result.withIcon(value)
```

PARAMETERS:

* **value** (`string`)

Icon to display when the value matches. Only specific visualizations.
###### fn RegexMap.options.result.withIndex

```jsonnet
RegexMap.options.result.withIndex(value)
```

PARAMETERS:

* **value** (`integer`)

Position in the mapping array. Only used internally.
###### fn RegexMap.options.result.withText

```jsonnet
RegexMap.options.result.withText(value)
```

PARAMETERS:

* **value** (`string`)

Text to display when the value matches
#### obj RegexMap.type


##### fn RegexMap.type.withMappingType

```jsonnet
RegexMap.type.withMappingType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"value"`, `"range"`, `"regex"`, `"special"`

Supported value mapping types
`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.
`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.
`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.
`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.
### obj SpecialValueMap


#### fn SpecialValueMap.withOptions

```jsonnet
SpecialValueMap.withOptions(value)
```

PARAMETERS:

* **value** (`object`)


#### fn SpecialValueMap.withOptionsMixin

```jsonnet
SpecialValueMap.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn SpecialValueMap.withType

```jsonnet
SpecialValueMap.withType(value="special")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"special"`


#### fn SpecialValueMap.withTypeMixin

```jsonnet
SpecialValueMap.withTypeMixin(value="special")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"special"`


#### obj SpecialValueMap.options


##### fn SpecialValueMap.options.withMatch

```jsonnet
SpecialValueMap.options.withMatch(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"true"`, `"false"`, `"null"`, `"nan"`, `"null+nan"`, `"empty"`

Special value types supported by the `SpecialValueMap`
##### fn SpecialValueMap.options.withResult

```jsonnet
SpecialValueMap.options.withResult(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
##### fn SpecialValueMap.options.withResultMixin

```jsonnet
SpecialValueMap.options.withResultMixin(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
##### obj SpecialValueMap.options.result


###### fn SpecialValueMap.options.result.withColor

```jsonnet
SpecialValueMap.options.result.withColor(value)
```

PARAMETERS:

* **value** (`string`)

Text to use when the value matches
###### fn SpecialValueMap.options.result.withIcon

```jsonnet
SpecialValueMap.options.result.withIcon(value)
```

PARAMETERS:

* **value** (`string`)

Icon to display when the value matches. Only specific visualizations.
###### fn SpecialValueMap.options.result.withIndex

```jsonnet
SpecialValueMap.options.result.withIndex(value)
```

PARAMETERS:

* **value** (`integer`)

Position in the mapping array. Only used internally.
###### fn SpecialValueMap.options.result.withText

```jsonnet
SpecialValueMap.options.result.withText(value)
```

PARAMETERS:

* **value** (`string`)

Text to display when the value matches
#### obj SpecialValueMap.type


##### fn SpecialValueMap.type.withMappingType

```jsonnet
SpecialValueMap.type.withMappingType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"value"`, `"range"`, `"regex"`, `"special"`

Supported value mapping types
`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.
`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.
`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.
`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.
### obj ValueMap


#### fn ValueMap.withOptions

```jsonnet
ValueMap.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

Map with <value_to_match>: ValueMappingResult. For example: { "10": { text: "Perfection!", color: "green" } }
#### fn ValueMap.withOptionsMixin

```jsonnet
ValueMap.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Map with <value_to_match>: ValueMappingResult. For example: { "10": { text: "Perfection!", color: "green" } }
#### fn ValueMap.withType

```jsonnet
ValueMap.withType(value="value")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"value"`


#### fn ValueMap.withTypeMixin

```jsonnet
ValueMap.withTypeMixin(value="value")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"value"`


#### obj ValueMap.type


##### fn ValueMap.type.withMappingType

```jsonnet
ValueMap.type.withMappingType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"value"`, `"range"`, `"regex"`, `"special"`

Supported value mapping types
`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.
`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.
`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.
`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.