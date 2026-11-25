# v2

grafonnet.apps.dashboard.v2

## Subpackages

* [spec.annotations](spec/annotations.md)
* [spec.layout.AutoGridLayoutKind.spec.items](spec/layout/AutoGridLayoutKind/spec/items.md)
* [spec.layout.GridLayoutKind.spec.items](spec/layout/GridLayoutKind/spec/items.md)
* [spec.layout.RowsLayoutKind.spec.rows](spec/layout/RowsLayoutKind/spec/rows/index.md)
* [spec.layout.TabsLayoutKind.spec.tabs](spec/layout/TabsLayoutKind/spec/tabs/index.md)
* [spec.links](spec/links.md)
* [spec.preferences.layout.AutoGridLayoutKind.spec.items](spec/preferences/layout/AutoGridLayoutKind/spec/items.md)
* [spec.preferences.layout.GridLayoutKind.spec.items](spec/preferences/layout/GridLayoutKind/spec/items.md)
* [spec.timeSettings.quickRanges](spec/timeSettings/quickRanges.md)
* [spec.variables.AdhocVariableKind.spec.baseFilters](spec/variables/AdhocVariableKind/spec/baseFilters.md)
* [spec.variables.AdhocVariableKind.spec.defaultKeys](spec/variables/AdhocVariableKind/spec/defaultKeys.md)
* [spec.variables.AdhocVariableKind.spec.filters](spec/variables/AdhocVariableKind/spec/filters.md)
* [spec.variables.CustomVariableKind.spec.options](spec/variables/CustomVariableKind/spec/options.md)
* [spec.variables.DatasourceVariableKind.spec.options](spec/variables/DatasourceVariableKind/spec/options.md)
* [spec.variables.GroupByVariableKind.spec.options](spec/variables/GroupByVariableKind/spec/options.md)
* [spec.variables.IntervalVariableKind.spec.options](spec/variables/IntervalVariableKind/spec/options.md)
* [spec.variables.QueryVariableKind.spec.options](spec/variables/QueryVariableKind/spec/options.md)
* [spec.variables.QueryVariableKind.spec.staticOptions](spec/variables/QueryVariableKind/spec/staticOptions.md)

## Index

* [`fn new(name, title)`](#fn-new)
* [`fn withApiVersion()`](#fn-withapiversion)
* [`fn withKind()`](#fn-withkind)
* [`fn withMetadata(value)`](#fn-withmetadata)
* [`fn withMetadataMixin(value)`](#fn-withmetadatamixin)
* [`fn withSpec(value)`](#fn-withspec)
* [`fn withSpecMixin(value)`](#fn-withspecmixin)
* [`obj metadata`](#obj-metadata)
  * [`fn withAnnotations(value)`](#fn-metadatawithannotations)
  * [`fn withAnnotationsMixin(value)`](#fn-metadatawithannotationsmixin)
  * [`fn withCreationTimestamp(value)`](#fn-metadatawithcreationtimestamp)
  * [`fn withDeletionTimestamp(value)`](#fn-metadatawithdeletiontimestamp)
  * [`fn withGeneration(value)`](#fn-metadatawithgeneration)
  * [`fn withLabels(value)`](#fn-metadatawithlabels)
  * [`fn withLabelsMixin(value)`](#fn-metadatawithlabelsmixin)
  * [`fn withName(value)`](#fn-metadatawithname)
  * [`fn withNamespace(value)`](#fn-metadatawithnamespace)
  * [`fn withResourceVersion(value)`](#fn-metadatawithresourceversion)
  * [`fn withUid(value)`](#fn-metadatawithuid)
  * [`fn withUpdateTimestamp(value)`](#fn-metadatawithupdatetimestamp)
* [`obj spec`](#obj-spec)
  * [`fn withAnnotations(value)`](#fn-specwithannotations)
  * [`fn withAnnotationsMixin(value)`](#fn-specwithannotationsmixin)
  * [`fn withCursorSync(value="Off")`](#fn-specwithcursorsync)
  * [`fn withDescription(value)`](#fn-specwithdescription)
  * [`fn withEditable(value=true)`](#fn-specwitheditable)
  * [`fn withElements(value)`](#fn-specwithelements)
  * [`fn withElementsMixin(value)`](#fn-specwithelementsmixin)
  * [`fn withLayout(value)`](#fn-specwithlayout)
  * [`fn withLayoutMixin(value)`](#fn-specwithlayoutmixin)
  * [`fn withLinks(value)`](#fn-specwithlinks)
  * [`fn withLinksMixin(value)`](#fn-specwithlinksmixin)
  * [`fn withLiveNow(value=true)`](#fn-specwithlivenow)
  * [`fn withPreferences(value)`](#fn-specwithpreferences)
  * [`fn withPreferencesMixin(value)`](#fn-specwithpreferencesmixin)
  * [`fn withPreload(value=true)`](#fn-specwithpreload)
  * [`fn withRevision(value)`](#fn-specwithrevision)
  * [`fn withTags(value)`](#fn-specwithtags)
  * [`fn withTagsMixin(value)`](#fn-specwithtagsmixin)
  * [`fn withTimeSettings(value)`](#fn-specwithtimesettings)
  * [`fn withTimeSettingsMixin(value)`](#fn-specwithtimesettingsmixin)
  * [`fn withTitle(value)`](#fn-specwithtitle)
  * [`fn withVariables(value)`](#fn-specwithvariables)
  * [`fn withVariablesMixin(value)`](#fn-specwithvariablesmixin)
  * [`obj layout`](#obj-speclayout)
    * [`fn withAutoGridLayoutKind(value)`](#fn-speclayoutwithautogridlayoutkind)
    * [`fn withAutoGridLayoutKindMixin(value)`](#fn-speclayoutwithautogridlayoutkindmixin)
    * [`fn withGridLayoutKind(value)`](#fn-speclayoutwithgridlayoutkind)
    * [`fn withGridLayoutKindMixin(value)`](#fn-speclayoutwithgridlayoutkindmixin)
    * [`fn withRowsLayoutKind(value)`](#fn-speclayoutwithrowslayoutkind)
    * [`fn withRowsLayoutKindMixin(value)`](#fn-speclayoutwithrowslayoutkindmixin)
    * [`fn withTabsLayoutKind(value)`](#fn-speclayoutwithtabslayoutkind)
    * [`fn withTabsLayoutKindMixin(value)`](#fn-speclayoutwithtabslayoutkindmixin)
    * [`obj AutoGridLayoutKind`](#obj-speclayoutautogridlayoutkind)
      * [`fn withKind()`](#fn-speclayoutautogridlayoutkindwithkind)
      * [`fn withSpec(value)`](#fn-speclayoutautogridlayoutkindwithspec)
      * [`fn withSpecMixin(value)`](#fn-speclayoutautogridlayoutkindwithspecmixin)
      * [`obj spec`](#obj-speclayoutautogridlayoutkindspec)
        * [`fn withColumnWidth(value)`](#fn-speclayoutautogridlayoutkindspecwithcolumnwidth)
        * [`fn withColumnWidthMode(value="standard")`](#fn-speclayoutautogridlayoutkindspecwithcolumnwidthmode)
        * [`fn withFillScreen(value=true)`](#fn-speclayoutautogridlayoutkindspecwithfillscreen)
        * [`fn withItems(value)`](#fn-speclayoutautogridlayoutkindspecwithitems)
        * [`fn withItemsMixin(value)`](#fn-speclayoutautogridlayoutkindspecwithitemsmixin)
        * [`fn withMaxColumnCount(value=3)`](#fn-speclayoutautogridlayoutkindspecwithmaxcolumncount)
        * [`fn withRowHeight(value)`](#fn-speclayoutautogridlayoutkindspecwithrowheight)
        * [`fn withRowHeightMode(value="standard")`](#fn-speclayoutautogridlayoutkindspecwithrowheightmode)
    * [`obj GridLayoutKind`](#obj-speclayoutgridlayoutkind)
      * [`fn withKind()`](#fn-speclayoutgridlayoutkindwithkind)
      * [`fn withSpec(value)`](#fn-speclayoutgridlayoutkindwithspec)
      * [`fn withSpecMixin(value)`](#fn-speclayoutgridlayoutkindwithspecmixin)
      * [`obj spec`](#obj-speclayoutgridlayoutkindspec)
        * [`fn withItems(value)`](#fn-speclayoutgridlayoutkindspecwithitems)
        * [`fn withItemsMixin(value)`](#fn-speclayoutgridlayoutkindspecwithitemsmixin)
    * [`obj RowsLayoutKind`](#obj-speclayoutrowslayoutkind)
      * [`fn withKind()`](#fn-speclayoutrowslayoutkindwithkind)
      * [`fn withSpec(value)`](#fn-speclayoutrowslayoutkindwithspec)
      * [`fn withSpecMixin(value)`](#fn-speclayoutrowslayoutkindwithspecmixin)
      * [`obj spec`](#obj-speclayoutrowslayoutkindspec)
        * [`fn withRows(value)`](#fn-speclayoutrowslayoutkindspecwithrows)
        * [`fn withRowsMixin(value)`](#fn-speclayoutrowslayoutkindspecwithrowsmixin)
    * [`obj TabsLayoutKind`](#obj-speclayouttabslayoutkind)
      * [`fn withKind()`](#fn-speclayouttabslayoutkindwithkind)
      * [`fn withSpec(value)`](#fn-speclayouttabslayoutkindwithspec)
      * [`fn withSpecMixin(value)`](#fn-speclayouttabslayoutkindwithspecmixin)
      * [`obj spec`](#obj-speclayouttabslayoutkindspec)
        * [`fn withTabs(value)`](#fn-speclayouttabslayoutkindspecwithtabs)
        * [`fn withTabsMixin(value)`](#fn-speclayouttabslayoutkindspecwithtabsmixin)
  * [`obj preferences`](#obj-specpreferences)
    * [`fn withLayout(value)`](#fn-specpreferenceswithlayout)
    * [`fn withLayoutMixin(value)`](#fn-specpreferenceswithlayoutmixin)
    * [`obj layout`](#obj-specpreferenceslayout)
      * [`fn withAutoGridLayoutKind(value)`](#fn-specpreferenceslayoutwithautogridlayoutkind)
      * [`fn withAutoGridLayoutKindMixin(value)`](#fn-specpreferenceslayoutwithautogridlayoutkindmixin)
      * [`fn withGridLayoutKind(value)`](#fn-specpreferenceslayoutwithgridlayoutkind)
      * [`fn withGridLayoutKindMixin(value)`](#fn-specpreferenceslayoutwithgridlayoutkindmixin)
      * [`obj AutoGridLayoutKind`](#obj-specpreferenceslayoutautogridlayoutkind)
        * [`fn withKind()`](#fn-specpreferenceslayoutautogridlayoutkindwithkind)
        * [`fn withSpec(value)`](#fn-specpreferenceslayoutautogridlayoutkindwithspec)
        * [`fn withSpecMixin(value)`](#fn-specpreferenceslayoutautogridlayoutkindwithspecmixin)
        * [`obj spec`](#obj-specpreferenceslayoutautogridlayoutkindspec)
          * [`fn withColumnWidth(value)`](#fn-specpreferenceslayoutautogridlayoutkindspecwithcolumnwidth)
          * [`fn withColumnWidthMode(value="standard")`](#fn-specpreferenceslayoutautogridlayoutkindspecwithcolumnwidthmode)
          * [`fn withFillScreen(value=true)`](#fn-specpreferenceslayoutautogridlayoutkindspecwithfillscreen)
          * [`fn withItems(value)`](#fn-specpreferenceslayoutautogridlayoutkindspecwithitems)
          * [`fn withItemsMixin(value)`](#fn-specpreferenceslayoutautogridlayoutkindspecwithitemsmixin)
          * [`fn withMaxColumnCount(value=3)`](#fn-specpreferenceslayoutautogridlayoutkindspecwithmaxcolumncount)
          * [`fn withRowHeight(value)`](#fn-specpreferenceslayoutautogridlayoutkindspecwithrowheight)
          * [`fn withRowHeightMode(value="standard")`](#fn-specpreferenceslayoutautogridlayoutkindspecwithrowheightmode)
      * [`obj GridLayoutKind`](#obj-specpreferenceslayoutgridlayoutkind)
        * [`fn withKind()`](#fn-specpreferenceslayoutgridlayoutkindwithkind)
        * [`fn withSpec(value)`](#fn-specpreferenceslayoutgridlayoutkindwithspec)
        * [`fn withSpecMixin(value)`](#fn-specpreferenceslayoutgridlayoutkindwithspecmixin)
        * [`obj spec`](#obj-specpreferenceslayoutgridlayoutkindspec)
          * [`fn withItems(value)`](#fn-specpreferenceslayoutgridlayoutkindspecwithitems)
          * [`fn withItemsMixin(value)`](#fn-specpreferenceslayoutgridlayoutkindspecwithitemsmixin)
  * [`obj timeSettings`](#obj-spectimesettings)
    * [`fn withAutoRefresh(value="")`](#fn-spectimesettingswithautorefresh)
    * [`fn withAutoRefreshIntervals(value=["5s","10s","30s","1m","5m","15m","30m","1h","2h","1d"])`](#fn-spectimesettingswithautorefreshintervals)
    * [`fn withAutoRefreshIntervalsMixin(value=["5s","10s","30s","1m","5m","15m","30m","1h","2h","1d"])`](#fn-spectimesettingswithautorefreshintervalsmixin)
    * [`fn withFiscalYearStartMonth(value=0)`](#fn-spectimesettingswithfiscalyearstartmonth)
    * [`fn withFrom(value="now-6h")`](#fn-spectimesettingswithfrom)
    * [`fn withHideTimepicker(value=true)`](#fn-spectimesettingswithhidetimepicker)
    * [`fn withNowDelay(value)`](#fn-spectimesettingswithnowdelay)
    * [`fn withQuickRanges(value)`](#fn-spectimesettingswithquickranges)
    * [`fn withQuickRangesMixin(value)`](#fn-spectimesettingswithquickrangesmixin)
    * [`fn withTimezone(value="browser")`](#fn-spectimesettingswithtimezone)
    * [`fn withTo(value="now")`](#fn-spectimesettingswithto)
    * [`fn withWeekStart(value)`](#fn-spectimesettingswithweekstart)
  * [`obj variables`](#obj-specvariables)
    * [`obj AdhocVariableKind`](#obj-specvariablesadhocvariablekind)
      * [`fn withDatasource(value)`](#fn-specvariablesadhocvariablekindwithdatasource)
      * [`fn withDatasourceMixin(value)`](#fn-specvariablesadhocvariablekindwithdatasourcemixin)
      * [`fn withGroup(value)`](#fn-specvariablesadhocvariablekindwithgroup)
      * [`fn withKind()`](#fn-specvariablesadhocvariablekindwithkind)
      * [`fn withLabels(value)`](#fn-specvariablesadhocvariablekindwithlabels)
      * [`fn withLabelsMixin(value)`](#fn-specvariablesadhocvariablekindwithlabelsmixin)
      * [`fn withSpec(value)`](#fn-specvariablesadhocvariablekindwithspec)
      * [`fn withSpecMixin(value)`](#fn-specvariablesadhocvariablekindwithspecmixin)
      * [`obj datasource`](#obj-specvariablesadhocvariablekinddatasource)
        * [`fn withName(value)`](#fn-specvariablesadhocvariablekinddatasourcewithname)
      * [`obj spec`](#obj-specvariablesadhocvariablekindspec)
        * [`fn withAllowCustomValue(value=true)`](#fn-specvariablesadhocvariablekindspecwithallowcustomvalue)
        * [`fn withBaseFilters(value)`](#fn-specvariablesadhocvariablekindspecwithbasefilters)
        * [`fn withBaseFiltersMixin(value)`](#fn-specvariablesadhocvariablekindspecwithbasefiltersmixin)
        * [`fn withDefaultKeys(value)`](#fn-specvariablesadhocvariablekindspecwithdefaultkeys)
        * [`fn withDefaultKeysMixin(value)`](#fn-specvariablesadhocvariablekindspecwithdefaultkeysmixin)
        * [`fn withDescription(value)`](#fn-specvariablesadhocvariablekindspecwithdescription)
        * [`fn withEnableGroupBy(value=true)`](#fn-specvariablesadhocvariablekindspecwithenablegroupby)
        * [`fn withFilters(value)`](#fn-specvariablesadhocvariablekindspecwithfilters)
        * [`fn withFiltersMixin(value)`](#fn-specvariablesadhocvariablekindspecwithfiltersmixin)
        * [`fn withHide(value="dontHide")`](#fn-specvariablesadhocvariablekindspecwithhide)
        * [`fn withLabel(value)`](#fn-specvariablesadhocvariablekindspecwithlabel)
        * [`fn withName(value="")`](#fn-specvariablesadhocvariablekindspecwithname)
        * [`fn withOrigin(value)`](#fn-specvariablesadhocvariablekindspecwithorigin)
        * [`fn withOriginMixin(value)`](#fn-specvariablesadhocvariablekindspecwithoriginmixin)
        * [`fn withSkipUrlSync(value=true)`](#fn-specvariablesadhocvariablekindspecwithskipurlsync)
        * [`obj origin`](#obj-specvariablesadhocvariablekindspecorigin)
          * [`fn withGroup(value)`](#fn-specvariablesadhocvariablekindspecoriginwithgroup)
          * [`fn withType()`](#fn-specvariablesadhocvariablekindspecoriginwithtype)
    * [`obj ConstantVariableKind`](#obj-specvariablesconstantvariablekind)
      * [`fn withKind()`](#fn-specvariablesconstantvariablekindwithkind)
      * [`fn withSpec(value)`](#fn-specvariablesconstantvariablekindwithspec)
      * [`fn withSpecMixin(value)`](#fn-specvariablesconstantvariablekindwithspecmixin)
      * [`obj spec`](#obj-specvariablesconstantvariablekindspec)
        * [`fn withCurrent(value={"text": "","value": ""})`](#fn-specvariablesconstantvariablekindspecwithcurrent)
        * [`fn withCurrentMixin(value={"text": "","value": ""})`](#fn-specvariablesconstantvariablekindspecwithcurrentmixin)
        * [`fn withDescription(value)`](#fn-specvariablesconstantvariablekindspecwithdescription)
        * [`fn withHide(value="dontHide")`](#fn-specvariablesconstantvariablekindspecwithhide)
        * [`fn withLabel(value)`](#fn-specvariablesconstantvariablekindspecwithlabel)
        * [`fn withName(value="")`](#fn-specvariablesconstantvariablekindspecwithname)
        * [`fn withOrigin(value)`](#fn-specvariablesconstantvariablekindspecwithorigin)
        * [`fn withOriginMixin(value)`](#fn-specvariablesconstantvariablekindspecwithoriginmixin)
        * [`fn withQuery(value="")`](#fn-specvariablesconstantvariablekindspecwithquery)
        * [`fn withSkipUrlSync(value=true)`](#fn-specvariablesconstantvariablekindspecwithskipurlsync)
        * [`obj current`](#obj-specvariablesconstantvariablekindspeccurrent)
          * [`fn withProperties(value)`](#fn-specvariablesconstantvariablekindspeccurrentwithproperties)
          * [`fn withPropertiesMixin(value)`](#fn-specvariablesconstantvariablekindspeccurrentwithpropertiesmixin)
          * [`fn withSelected(value=true)`](#fn-specvariablesconstantvariablekindspeccurrentwithselected)
          * [`fn withText(value)`](#fn-specvariablesconstantvariablekindspeccurrentwithtext)
          * [`fn withTextMixin(value)`](#fn-specvariablesconstantvariablekindspeccurrentwithtextmixin)
          * [`fn withValue(value)`](#fn-specvariablesconstantvariablekindspeccurrentwithvalue)
          * [`fn withValueMixin(value)`](#fn-specvariablesconstantvariablekindspeccurrentwithvaluemixin)
        * [`obj origin`](#obj-specvariablesconstantvariablekindspecorigin)
          * [`fn withGroup(value)`](#fn-specvariablesconstantvariablekindspecoriginwithgroup)
          * [`fn withType()`](#fn-specvariablesconstantvariablekindspecoriginwithtype)
    * [`obj CustomVariableKind`](#obj-specvariablescustomvariablekind)
      * [`fn withKind()`](#fn-specvariablescustomvariablekindwithkind)
      * [`fn withSpec(value)`](#fn-specvariablescustomvariablekindwithspec)
      * [`fn withSpecMixin(value)`](#fn-specvariablescustomvariablekindwithspecmixin)
      * [`obj spec`](#obj-specvariablescustomvariablekindspec)
        * [`fn withAllValue(value)`](#fn-specvariablescustomvariablekindspecwithallvalue)
        * [`fn withAllowCustomValue(value=true)`](#fn-specvariablescustomvariablekindspecwithallowcustomvalue)
        * [`fn withCurrent(value)`](#fn-specvariablescustomvariablekindspecwithcurrent)
        * [`fn withCurrentMixin(value)`](#fn-specvariablescustomvariablekindspecwithcurrentmixin)
        * [`fn withDescription(value)`](#fn-specvariablescustomvariablekindspecwithdescription)
        * [`fn withHide(value="dontHide")`](#fn-specvariablescustomvariablekindspecwithhide)
        * [`fn withIncludeAll(value=true)`](#fn-specvariablescustomvariablekindspecwithincludeall)
        * [`fn withLabel(value)`](#fn-specvariablescustomvariablekindspecwithlabel)
        * [`fn withMulti(value=true)`](#fn-specvariablescustomvariablekindspecwithmulti)
        * [`fn withName(value="")`](#fn-specvariablescustomvariablekindspecwithname)
        * [`fn withOptions(value)`](#fn-specvariablescustomvariablekindspecwithoptions)
        * [`fn withOptionsMixin(value)`](#fn-specvariablescustomvariablekindspecwithoptionsmixin)
        * [`fn withOrigin(value)`](#fn-specvariablescustomvariablekindspecwithorigin)
        * [`fn withOriginMixin(value)`](#fn-specvariablescustomvariablekindspecwithoriginmixin)
        * [`fn withQuery(value="")`](#fn-specvariablescustomvariablekindspecwithquery)
        * [`fn withSkipUrlSync(value=true)`](#fn-specvariablescustomvariablekindspecwithskipurlsync)
        * [`fn withValuesFormat(value)`](#fn-specvariablescustomvariablekindspecwithvaluesformat)
        * [`obj current`](#obj-specvariablescustomvariablekindspeccurrent)
          * [`fn withProperties(value)`](#fn-specvariablescustomvariablekindspeccurrentwithproperties)
          * [`fn withPropertiesMixin(value)`](#fn-specvariablescustomvariablekindspeccurrentwithpropertiesmixin)
          * [`fn withSelected(value=true)`](#fn-specvariablescustomvariablekindspeccurrentwithselected)
          * [`fn withText(value)`](#fn-specvariablescustomvariablekindspeccurrentwithtext)
          * [`fn withTextMixin(value)`](#fn-specvariablescustomvariablekindspeccurrentwithtextmixin)
          * [`fn withValue(value)`](#fn-specvariablescustomvariablekindspeccurrentwithvalue)
          * [`fn withValueMixin(value)`](#fn-specvariablescustomvariablekindspeccurrentwithvaluemixin)
        * [`obj origin`](#obj-specvariablescustomvariablekindspecorigin)
          * [`fn withGroup(value)`](#fn-specvariablescustomvariablekindspecoriginwithgroup)
          * [`fn withType()`](#fn-specvariablescustomvariablekindspecoriginwithtype)
    * [`obj DatasourceVariableKind`](#obj-specvariablesdatasourcevariablekind)
      * [`fn withKind()`](#fn-specvariablesdatasourcevariablekindwithkind)
      * [`fn withSpec(value)`](#fn-specvariablesdatasourcevariablekindwithspec)
      * [`fn withSpecMixin(value)`](#fn-specvariablesdatasourcevariablekindwithspecmixin)
      * [`obj spec`](#obj-specvariablesdatasourcevariablekindspec)
        * [`fn withAllValue(value)`](#fn-specvariablesdatasourcevariablekindspecwithallvalue)
        * [`fn withAllowCustomValue(value=true)`](#fn-specvariablesdatasourcevariablekindspecwithallowcustomvalue)
        * [`fn withCurrent(value={"text": "","value": ""})`](#fn-specvariablesdatasourcevariablekindspecwithcurrent)
        * [`fn withCurrentMixin(value={"text": "","value": ""})`](#fn-specvariablesdatasourcevariablekindspecwithcurrentmixin)
        * [`fn withDescription(value)`](#fn-specvariablesdatasourcevariablekindspecwithdescription)
        * [`fn withHide(value="dontHide")`](#fn-specvariablesdatasourcevariablekindspecwithhide)
        * [`fn withIncludeAll(value=true)`](#fn-specvariablesdatasourcevariablekindspecwithincludeall)
        * [`fn withLabel(value)`](#fn-specvariablesdatasourcevariablekindspecwithlabel)
        * [`fn withMulti(value=true)`](#fn-specvariablesdatasourcevariablekindspecwithmulti)
        * [`fn withName(value="")`](#fn-specvariablesdatasourcevariablekindspecwithname)
        * [`fn withOptions(value)`](#fn-specvariablesdatasourcevariablekindspecwithoptions)
        * [`fn withOptionsMixin(value)`](#fn-specvariablesdatasourcevariablekindspecwithoptionsmixin)
        * [`fn withOrigin(value)`](#fn-specvariablesdatasourcevariablekindspecwithorigin)
        * [`fn withOriginMixin(value)`](#fn-specvariablesdatasourcevariablekindspecwithoriginmixin)
        * [`fn withPluginId(value="")`](#fn-specvariablesdatasourcevariablekindspecwithpluginid)
        * [`fn withRefresh(value="never")`](#fn-specvariablesdatasourcevariablekindspecwithrefresh)
        * [`fn withRegex(value="")`](#fn-specvariablesdatasourcevariablekindspecwithregex)
        * [`fn withSkipUrlSync(value=true)`](#fn-specvariablesdatasourcevariablekindspecwithskipurlsync)
        * [`obj current`](#obj-specvariablesdatasourcevariablekindspeccurrent)
          * [`fn withProperties(value)`](#fn-specvariablesdatasourcevariablekindspeccurrentwithproperties)
          * [`fn withPropertiesMixin(value)`](#fn-specvariablesdatasourcevariablekindspeccurrentwithpropertiesmixin)
          * [`fn withSelected(value=true)`](#fn-specvariablesdatasourcevariablekindspeccurrentwithselected)
          * [`fn withText(value)`](#fn-specvariablesdatasourcevariablekindspeccurrentwithtext)
          * [`fn withTextMixin(value)`](#fn-specvariablesdatasourcevariablekindspeccurrentwithtextmixin)
          * [`fn withValue(value)`](#fn-specvariablesdatasourcevariablekindspeccurrentwithvalue)
          * [`fn withValueMixin(value)`](#fn-specvariablesdatasourcevariablekindspeccurrentwithvaluemixin)
        * [`obj origin`](#obj-specvariablesdatasourcevariablekindspecorigin)
          * [`fn withGroup(value)`](#fn-specvariablesdatasourcevariablekindspecoriginwithgroup)
          * [`fn withType()`](#fn-specvariablesdatasourcevariablekindspecoriginwithtype)
    * [`obj GroupByVariableKind`](#obj-specvariablesgroupbyvariablekind)
      * [`fn withDatasource(value)`](#fn-specvariablesgroupbyvariablekindwithdatasource)
      * [`fn withDatasourceMixin(value)`](#fn-specvariablesgroupbyvariablekindwithdatasourcemixin)
      * [`fn withGroup(value)`](#fn-specvariablesgroupbyvariablekindwithgroup)
      * [`fn withKind()`](#fn-specvariablesgroupbyvariablekindwithkind)
      * [`fn withLabels(value)`](#fn-specvariablesgroupbyvariablekindwithlabels)
      * [`fn withLabelsMixin(value)`](#fn-specvariablesgroupbyvariablekindwithlabelsmixin)
      * [`fn withSpec(value)`](#fn-specvariablesgroupbyvariablekindwithspec)
      * [`fn withSpecMixin(value)`](#fn-specvariablesgroupbyvariablekindwithspecmixin)
      * [`obj datasource`](#obj-specvariablesgroupbyvariablekinddatasource)
        * [`fn withName(value)`](#fn-specvariablesgroupbyvariablekinddatasourcewithname)
      * [`obj spec`](#obj-specvariablesgroupbyvariablekindspec)
        * [`fn withCurrent(value={"text": "","value": ""})`](#fn-specvariablesgroupbyvariablekindspecwithcurrent)
        * [`fn withCurrentMixin(value={"text": "","value": ""})`](#fn-specvariablesgroupbyvariablekindspecwithcurrentmixin)
        * [`fn withDefaultValue(value)`](#fn-specvariablesgroupbyvariablekindspecwithdefaultvalue)
        * [`fn withDefaultValueMixin(value)`](#fn-specvariablesgroupbyvariablekindspecwithdefaultvaluemixin)
        * [`fn withDescription(value)`](#fn-specvariablesgroupbyvariablekindspecwithdescription)
        * [`fn withHide(value="dontHide")`](#fn-specvariablesgroupbyvariablekindspecwithhide)
        * [`fn withLabel(value)`](#fn-specvariablesgroupbyvariablekindspecwithlabel)
        * [`fn withMulti(value=true)`](#fn-specvariablesgroupbyvariablekindspecwithmulti)
        * [`fn withName(value="")`](#fn-specvariablesgroupbyvariablekindspecwithname)
        * [`fn withOptions(value)`](#fn-specvariablesgroupbyvariablekindspecwithoptions)
        * [`fn withOptionsMixin(value)`](#fn-specvariablesgroupbyvariablekindspecwithoptionsmixin)
        * [`fn withOrigin(value)`](#fn-specvariablesgroupbyvariablekindspecwithorigin)
        * [`fn withOriginMixin(value)`](#fn-specvariablesgroupbyvariablekindspecwithoriginmixin)
        * [`fn withSkipUrlSync(value=true)`](#fn-specvariablesgroupbyvariablekindspecwithskipurlsync)
        * [`obj current`](#obj-specvariablesgroupbyvariablekindspeccurrent)
          * [`fn withProperties(value)`](#fn-specvariablesgroupbyvariablekindspeccurrentwithproperties)
          * [`fn withPropertiesMixin(value)`](#fn-specvariablesgroupbyvariablekindspeccurrentwithpropertiesmixin)
          * [`fn withSelected(value=true)`](#fn-specvariablesgroupbyvariablekindspeccurrentwithselected)
          * [`fn withText(value)`](#fn-specvariablesgroupbyvariablekindspeccurrentwithtext)
          * [`fn withTextMixin(value)`](#fn-specvariablesgroupbyvariablekindspeccurrentwithtextmixin)
          * [`fn withValue(value)`](#fn-specvariablesgroupbyvariablekindspeccurrentwithvalue)
          * [`fn withValueMixin(value)`](#fn-specvariablesgroupbyvariablekindspeccurrentwithvaluemixin)
        * [`obj defaultValue`](#obj-specvariablesgroupbyvariablekindspecdefaultvalue)
          * [`fn withProperties(value)`](#fn-specvariablesgroupbyvariablekindspecdefaultvaluewithproperties)
          * [`fn withPropertiesMixin(value)`](#fn-specvariablesgroupbyvariablekindspecdefaultvaluewithpropertiesmixin)
          * [`fn withSelected(value=true)`](#fn-specvariablesgroupbyvariablekindspecdefaultvaluewithselected)
          * [`fn withText(value)`](#fn-specvariablesgroupbyvariablekindspecdefaultvaluewithtext)
          * [`fn withTextMixin(value)`](#fn-specvariablesgroupbyvariablekindspecdefaultvaluewithtextmixin)
          * [`fn withValue(value)`](#fn-specvariablesgroupbyvariablekindspecdefaultvaluewithvalue)
          * [`fn withValueMixin(value)`](#fn-specvariablesgroupbyvariablekindspecdefaultvaluewithvaluemixin)
        * [`obj origin`](#obj-specvariablesgroupbyvariablekindspecorigin)
          * [`fn withGroup(value)`](#fn-specvariablesgroupbyvariablekindspecoriginwithgroup)
          * [`fn withType()`](#fn-specvariablesgroupbyvariablekindspecoriginwithtype)
    * [`obj IntervalVariableKind`](#obj-specvariablesintervalvariablekind)
      * [`fn withKind()`](#fn-specvariablesintervalvariablekindwithkind)
      * [`fn withSpec(value)`](#fn-specvariablesintervalvariablekindwithspec)
      * [`fn withSpecMixin(value)`](#fn-specvariablesintervalvariablekindwithspecmixin)
      * [`obj spec`](#obj-specvariablesintervalvariablekindspec)
        * [`fn withAuto(value=true)`](#fn-specvariablesintervalvariablekindspecwithauto)
        * [`fn withAutoCount(value=0)`](#fn-specvariablesintervalvariablekindspecwithautocount)
        * [`fn withAutoMin(value="")`](#fn-specvariablesintervalvariablekindspecwithautomin)
        * [`fn withCurrent(value={"text": "","value": ""})`](#fn-specvariablesintervalvariablekindspecwithcurrent)
        * [`fn withCurrentMixin(value={"text": "","value": ""})`](#fn-specvariablesintervalvariablekindspecwithcurrentmixin)
        * [`fn withDescription(value)`](#fn-specvariablesintervalvariablekindspecwithdescription)
        * [`fn withHide(value="dontHide")`](#fn-specvariablesintervalvariablekindspecwithhide)
        * [`fn withLabel(value)`](#fn-specvariablesintervalvariablekindspecwithlabel)
        * [`fn withName(value="")`](#fn-specvariablesintervalvariablekindspecwithname)
        * [`fn withOptions(value)`](#fn-specvariablesintervalvariablekindspecwithoptions)
        * [`fn withOptionsMixin(value)`](#fn-specvariablesintervalvariablekindspecwithoptionsmixin)
        * [`fn withOrigin(value)`](#fn-specvariablesintervalvariablekindspecwithorigin)
        * [`fn withOriginMixin(value)`](#fn-specvariablesintervalvariablekindspecwithoriginmixin)
        * [`fn withQuery(value="")`](#fn-specvariablesintervalvariablekindspecwithquery)
        * [`fn withRefresh()`](#fn-specvariablesintervalvariablekindspecwithrefresh)
        * [`fn withSkipUrlSync(value=true)`](#fn-specvariablesintervalvariablekindspecwithskipurlsync)
        * [`obj current`](#obj-specvariablesintervalvariablekindspeccurrent)
          * [`fn withProperties(value)`](#fn-specvariablesintervalvariablekindspeccurrentwithproperties)
          * [`fn withPropertiesMixin(value)`](#fn-specvariablesintervalvariablekindspeccurrentwithpropertiesmixin)
          * [`fn withSelected(value=true)`](#fn-specvariablesintervalvariablekindspeccurrentwithselected)
          * [`fn withText(value)`](#fn-specvariablesintervalvariablekindspeccurrentwithtext)
          * [`fn withTextMixin(value)`](#fn-specvariablesintervalvariablekindspeccurrentwithtextmixin)
          * [`fn withValue(value)`](#fn-specvariablesintervalvariablekindspeccurrentwithvalue)
          * [`fn withValueMixin(value)`](#fn-specvariablesintervalvariablekindspeccurrentwithvaluemixin)
        * [`obj origin`](#obj-specvariablesintervalvariablekindspecorigin)
          * [`fn withGroup(value)`](#fn-specvariablesintervalvariablekindspecoriginwithgroup)
          * [`fn withType()`](#fn-specvariablesintervalvariablekindspecoriginwithtype)
    * [`obj QueryVariableKind`](#obj-specvariablesqueryvariablekind)
      * [`fn withKind()`](#fn-specvariablesqueryvariablekindwithkind)
      * [`fn withSpec(value)`](#fn-specvariablesqueryvariablekindwithspec)
      * [`fn withSpecMixin(value)`](#fn-specvariablesqueryvariablekindwithspecmixin)
      * [`obj spec`](#obj-specvariablesqueryvariablekindspec)
        * [`fn withAllValue(value)`](#fn-specvariablesqueryvariablekindspecwithallvalue)
        * [`fn withAllowCustomValue(value=true)`](#fn-specvariablesqueryvariablekindspecwithallowcustomvalue)
        * [`fn withCurrent(value={"text": "","value": ""})`](#fn-specvariablesqueryvariablekindspecwithcurrent)
        * [`fn withCurrentMixin(value={"text": "","value": ""})`](#fn-specvariablesqueryvariablekindspecwithcurrentmixin)
        * [`fn withDefinition(value)`](#fn-specvariablesqueryvariablekindspecwithdefinition)
        * [`fn withDescription(value)`](#fn-specvariablesqueryvariablekindspecwithdescription)
        * [`fn withHide(value="dontHide")`](#fn-specvariablesqueryvariablekindspecwithhide)
        * [`fn withIncludeAll(value=true)`](#fn-specvariablesqueryvariablekindspecwithincludeall)
        * [`fn withLabel(value)`](#fn-specvariablesqueryvariablekindspecwithlabel)
        * [`fn withMulti(value=true)`](#fn-specvariablesqueryvariablekindspecwithmulti)
        * [`fn withName(value="")`](#fn-specvariablesqueryvariablekindspecwithname)
        * [`fn withOptions(value)`](#fn-specvariablesqueryvariablekindspecwithoptions)
        * [`fn withOptionsMixin(value)`](#fn-specvariablesqueryvariablekindspecwithoptionsmixin)
        * [`fn withOrigin(value)`](#fn-specvariablesqueryvariablekindspecwithorigin)
        * [`fn withOriginMixin(value)`](#fn-specvariablesqueryvariablekindspecwithoriginmixin)
        * [`fn withPlaceholder(value)`](#fn-specvariablesqueryvariablekindspecwithplaceholder)
        * [`fn withQuery(value)`](#fn-specvariablesqueryvariablekindspecwithquery)
        * [`fn withQueryMixin(value)`](#fn-specvariablesqueryvariablekindspecwithquerymixin)
        * [`fn withRefresh(value="never")`](#fn-specvariablesqueryvariablekindspecwithrefresh)
        * [`fn withRegex(value="")`](#fn-specvariablesqueryvariablekindspecwithregex)
        * [`fn withRegexApplyTo(value="value")`](#fn-specvariablesqueryvariablekindspecwithregexapplyto)
        * [`fn withSkipUrlSync(value=true)`](#fn-specvariablesqueryvariablekindspecwithskipurlsync)
        * [`fn withSort(value)`](#fn-specvariablesqueryvariablekindspecwithsort)
        * [`fn withStaticOptions(value)`](#fn-specvariablesqueryvariablekindspecwithstaticoptions)
        * [`fn withStaticOptionsMixin(value)`](#fn-specvariablesqueryvariablekindspecwithstaticoptionsmixin)
        * [`fn withStaticOptionsOrder(value)`](#fn-specvariablesqueryvariablekindspecwithstaticoptionsorder)
        * [`obj current`](#obj-specvariablesqueryvariablekindspeccurrent)
          * [`fn withProperties(value)`](#fn-specvariablesqueryvariablekindspeccurrentwithproperties)
          * [`fn withPropertiesMixin(value)`](#fn-specvariablesqueryvariablekindspeccurrentwithpropertiesmixin)
          * [`fn withSelected(value=true)`](#fn-specvariablesqueryvariablekindspeccurrentwithselected)
          * [`fn withText(value)`](#fn-specvariablesqueryvariablekindspeccurrentwithtext)
          * [`fn withTextMixin(value)`](#fn-specvariablesqueryvariablekindspeccurrentwithtextmixin)
          * [`fn withValue(value)`](#fn-specvariablesqueryvariablekindspeccurrentwithvalue)
          * [`fn withValueMixin(value)`](#fn-specvariablesqueryvariablekindspeccurrentwithvaluemixin)
        * [`obj origin`](#obj-specvariablesqueryvariablekindspecorigin)
          * [`fn withGroup(value)`](#fn-specvariablesqueryvariablekindspecoriginwithgroup)
          * [`fn withType()`](#fn-specvariablesqueryvariablekindspecoriginwithtype)
        * [`obj query`](#obj-specvariablesqueryvariablekindspecquery)
          * [`fn withDatasource(value)`](#fn-specvariablesqueryvariablekindspecquerywithdatasource)
          * [`fn withDatasourceMixin(value)`](#fn-specvariablesqueryvariablekindspecquerywithdatasourcemixin)
          * [`fn withGroup(value)`](#fn-specvariablesqueryvariablekindspecquerywithgroup)
          * [`fn withKind()`](#fn-specvariablesqueryvariablekindspecquerywithkind)
          * [`fn withLabels(value)`](#fn-specvariablesqueryvariablekindspecquerywithlabels)
          * [`fn withLabelsMixin(value)`](#fn-specvariablesqueryvariablekindspecquerywithlabelsmixin)
          * [`fn withSpec(value)`](#fn-specvariablesqueryvariablekindspecquerywithspec)
          * [`fn withSpecMixin(value)`](#fn-specvariablesqueryvariablekindspecquerywithspecmixin)
          * [`fn withVersion(value="v0")`](#fn-specvariablesqueryvariablekindspecquerywithversion)
          * [`obj datasource`](#obj-specvariablesqueryvariablekindspecquerydatasource)
            * [`fn withName(value)`](#fn-specvariablesqueryvariablekindspecquerydatasourcewithname)
    * [`obj SwitchVariableKind`](#obj-specvariablesswitchvariablekind)
      * [`fn withKind()`](#fn-specvariablesswitchvariablekindwithkind)
      * [`fn withSpec(value)`](#fn-specvariablesswitchvariablekindwithspec)
      * [`fn withSpecMixin(value)`](#fn-specvariablesswitchvariablekindwithspecmixin)
      * [`obj spec`](#obj-specvariablesswitchvariablekindspec)
        * [`fn withCurrent(value="false")`](#fn-specvariablesswitchvariablekindspecwithcurrent)
        * [`fn withDescription(value)`](#fn-specvariablesswitchvariablekindspecwithdescription)
        * [`fn withDisabledValue(value="false")`](#fn-specvariablesswitchvariablekindspecwithdisabledvalue)
        * [`fn withEnabledValue(value="true")`](#fn-specvariablesswitchvariablekindspecwithenabledvalue)
        * [`fn withHide(value="dontHide")`](#fn-specvariablesswitchvariablekindspecwithhide)
        * [`fn withLabel(value)`](#fn-specvariablesswitchvariablekindspecwithlabel)
        * [`fn withName(value="")`](#fn-specvariablesswitchvariablekindspecwithname)
        * [`fn withOrigin(value)`](#fn-specvariablesswitchvariablekindspecwithorigin)
        * [`fn withOriginMixin(value)`](#fn-specvariablesswitchvariablekindspecwithoriginmixin)
        * [`fn withSkipUrlSync(value=true)`](#fn-specvariablesswitchvariablekindspecwithskipurlsync)
        * [`obj origin`](#obj-specvariablesswitchvariablekindspecorigin)
          * [`fn withGroup(value)`](#fn-specvariablesswitchvariablekindspecoriginwithgroup)
          * [`fn withType()`](#fn-specvariablesswitchvariablekindspecoriginwithtype)
    * [`obj TextVariableKind`](#obj-specvariablestextvariablekind)
      * [`fn withKind()`](#fn-specvariablestextvariablekindwithkind)
      * [`fn withSpec(value)`](#fn-specvariablestextvariablekindwithspec)
      * [`fn withSpecMixin(value)`](#fn-specvariablestextvariablekindwithspecmixin)
      * [`obj spec`](#obj-specvariablestextvariablekindspec)
        * [`fn withCurrent(value={"text": "","value": ""})`](#fn-specvariablestextvariablekindspecwithcurrent)
        * [`fn withCurrentMixin(value={"text": "","value": ""})`](#fn-specvariablestextvariablekindspecwithcurrentmixin)
        * [`fn withDescription(value)`](#fn-specvariablestextvariablekindspecwithdescription)
        * [`fn withHide(value="dontHide")`](#fn-specvariablestextvariablekindspecwithhide)
        * [`fn withLabel(value)`](#fn-specvariablestextvariablekindspecwithlabel)
        * [`fn withName(value="")`](#fn-specvariablestextvariablekindspecwithname)
        * [`fn withOrigin(value)`](#fn-specvariablestextvariablekindspecwithorigin)
        * [`fn withOriginMixin(value)`](#fn-specvariablestextvariablekindspecwithoriginmixin)
        * [`fn withQuery(value="")`](#fn-specvariablestextvariablekindspecwithquery)
        * [`fn withSkipUrlSync(value=true)`](#fn-specvariablestextvariablekindspecwithskipurlsync)
        * [`obj current`](#obj-specvariablestextvariablekindspeccurrent)
          * [`fn withProperties(value)`](#fn-specvariablestextvariablekindspeccurrentwithproperties)
          * [`fn withPropertiesMixin(value)`](#fn-specvariablestextvariablekindspeccurrentwithpropertiesmixin)
          * [`fn withSelected(value=true)`](#fn-specvariablestextvariablekindspeccurrentwithselected)
          * [`fn withText(value)`](#fn-specvariablestextvariablekindspeccurrentwithtext)
          * [`fn withTextMixin(value)`](#fn-specvariablestextvariablekindspeccurrentwithtextmixin)
          * [`fn withValue(value)`](#fn-specvariablestextvariablekindspeccurrentwithvalue)
          * [`fn withValueMixin(value)`](#fn-specvariablestextvariablekindspeccurrentwithvaluemixin)
        * [`obj origin`](#obj-specvariablestextvariablekindspecorigin)
          * [`fn withGroup(value)`](#fn-specvariablestextvariablekindspecoriginwithgroup)
          * [`fn withType()`](#fn-specvariablestextvariablekindspecoriginwithtype)

## Fields

### fn new

```jsonnet
new(name, title)
```

PARAMETERS:

* **name** (`string`)
* **title** (`string`)

Creates a new dashboard.Dashboard resource.
### fn withApiVersion

```jsonnet
withApiVersion()
```


set the resource's apiVersion
### fn withKind

```jsonnet
withKind()
```


set the resource's kind
### fn withMetadata

```jsonnet
withMetadata(value)
```

PARAMETERS:

* **value** (`object`)


### fn withMetadataMixin

```jsonnet
withMetadataMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withSpec

```jsonnet
withSpec(value)
```

PARAMETERS:

* **value** (`object`)


### fn withSpecMixin

```jsonnet
withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


### obj metadata


#### fn metadata.withAnnotations

```jsonnet
metadata.withAnnotations(value)
```

PARAMETERS:

* **value** (`object`)


#### fn metadata.withAnnotationsMixin

```jsonnet
metadata.withAnnotationsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn metadata.withCreationTimestamp

```jsonnet
metadata.withCreationTimestamp(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withDeletionTimestamp

```jsonnet
metadata.withDeletionTimestamp(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withGeneration

```jsonnet
metadata.withGeneration(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn metadata.withLabels

```jsonnet
metadata.withLabels(value)
```

PARAMETERS:

* **value** (`object`)


#### fn metadata.withLabelsMixin

```jsonnet
metadata.withLabelsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn metadata.withName

```jsonnet
metadata.withName(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withNamespace

```jsonnet
metadata.withNamespace(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withResourceVersion

```jsonnet
metadata.withResourceVersion(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withUid

```jsonnet
metadata.withUid(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withUpdateTimestamp

```jsonnet
metadata.withUpdateTimestamp(value)
```

PARAMETERS:

* **value** (`string`)


### obj spec


#### fn spec.withAnnotations

```jsonnet
spec.withAnnotations(value)
```

PARAMETERS:

* **value** (`array`)


#### fn spec.withAnnotationsMixin

```jsonnet
spec.withAnnotationsMixin(value)
```

PARAMETERS:

* **value** (`array`)


#### fn spec.withCursorSync

```jsonnet
spec.withCursorSync(value="Off")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"Off"`
   - valid values: `"Crosshair"`, `"Tooltip"`, `"Off"`

"Off" for no shared crosshair or tooltip (default).
"Crosshair" for shared crosshair.
"Tooltip" for shared crosshair AND shared tooltip.
#### fn spec.withDescription

```jsonnet
spec.withDescription(value)
```

PARAMETERS:

* **value** (`string`)

Description of dashboard.
#### fn spec.withEditable

```jsonnet
spec.withEditable(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether a dashboard is editable or not.
#### fn spec.withElements

```jsonnet
spec.withElements(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withElementsMixin

```jsonnet
spec.withElementsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withLayout

```jsonnet
spec.withLayout(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withLayoutMixin

```jsonnet
spec.withLayoutMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withLinks

```jsonnet
spec.withLinks(value)
```

PARAMETERS:

* **value** (`array`)

Links with references to other dashboards or external websites.
#### fn spec.withLinksMixin

```jsonnet
spec.withLinksMixin(value)
```

PARAMETERS:

* **value** (`array`)

Links with references to other dashboards or external websites.
#### fn spec.withLiveNow

```jsonnet
spec.withLiveNow(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

When set to true, the dashboard will redraw panels at an interval matching the pixel width.
This will keep data "moving left" regardless of the query refresh rate. This setting helps
avoid dashboards presenting stale live data.
#### fn spec.withPreferences

```jsonnet
spec.withPreferences(value)
```

PARAMETERS:

* **value** (`object`)

Dashboard specific preferences (applied per dashboard = all users using the dashboard)
#### fn spec.withPreferencesMixin

```jsonnet
spec.withPreferencesMixin(value)
```

PARAMETERS:

* **value** (`object`)

Dashboard specific preferences (applied per dashboard = all users using the dashboard)
#### fn spec.withPreload

```jsonnet
spec.withPreload(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

When set to true, the dashboard will load all panels in the dashboard when it's loaded.
#### fn spec.withRevision

```jsonnet
spec.withRevision(value)
```

PARAMETERS:

* **value** (`integer`)

Plugins only. The version of the dashboard installed together with the plugin.
This is used to determine if the dashboard should be updated when the plugin is updated.
#### fn spec.withTags

```jsonnet
spec.withTags(value)
```

PARAMETERS:

* **value** (`array`)

Tags associated with dashboard.
#### fn spec.withTagsMixin

```jsonnet
spec.withTagsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Tags associated with dashboard.
#### fn spec.withTimeSettings

```jsonnet
spec.withTimeSettings(value)
```

PARAMETERS:

* **value** (`object`)

Time configuration
It defines the default time config for the time picker, the refresh picker for the specific dashboard.
#### fn spec.withTimeSettingsMixin

```jsonnet
spec.withTimeSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Time configuration
It defines the default time config for the time picker, the refresh picker for the specific dashboard.
#### fn spec.withTitle

```jsonnet
spec.withTitle(value)
```

PARAMETERS:

* **value** (`string`)

Title of dashboard.
#### fn spec.withVariables

```jsonnet
spec.withVariables(value)
```

PARAMETERS:

* **value** (`array`)

Configured template variables.
#### fn spec.withVariablesMixin

```jsonnet
spec.withVariablesMixin(value)
```

PARAMETERS:

* **value** (`array`)

Configured template variables.
#### obj spec.layout


##### fn spec.layout.withAutoGridLayoutKind

```jsonnet
spec.layout.withAutoGridLayoutKind(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.layout.withAutoGridLayoutKindMixin

```jsonnet
spec.layout.withAutoGridLayoutKindMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.layout.withGridLayoutKind

```jsonnet
spec.layout.withGridLayoutKind(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.layout.withGridLayoutKindMixin

```jsonnet
spec.layout.withGridLayoutKindMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.layout.withRowsLayoutKind

```jsonnet
spec.layout.withRowsLayoutKind(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.layout.withRowsLayoutKindMixin

```jsonnet
spec.layout.withRowsLayoutKindMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.layout.withTabsLayoutKind

```jsonnet
spec.layout.withTabsLayoutKind(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.layout.withTabsLayoutKindMixin

```jsonnet
spec.layout.withTabsLayoutKindMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### obj spec.layout.AutoGridLayoutKind


###### fn spec.layout.AutoGridLayoutKind.withKind

```jsonnet
spec.layout.AutoGridLayoutKind.withKind()
```



###### fn spec.layout.AutoGridLayoutKind.withSpec

```jsonnet
spec.layout.AutoGridLayoutKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.layout.AutoGridLayoutKind.withSpecMixin

```jsonnet
spec.layout.AutoGridLayoutKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### obj spec.layout.AutoGridLayoutKind.spec


####### fn spec.layout.AutoGridLayoutKind.spec.withColumnWidth

```jsonnet
spec.layout.AutoGridLayoutKind.spec.withColumnWidth(value)
```

PARAMETERS:

* **value** (`number`)


####### fn spec.layout.AutoGridLayoutKind.spec.withColumnWidthMode

```jsonnet
spec.layout.AutoGridLayoutKind.spec.withColumnWidthMode(value="standard")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"standard"`
   - valid values: `"narrow"`, `"standard"`, `"wide"`, `"custom"`


####### fn spec.layout.AutoGridLayoutKind.spec.withFillScreen

```jsonnet
spec.layout.AutoGridLayoutKind.spec.withFillScreen(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.layout.AutoGridLayoutKind.spec.withItems

```jsonnet
spec.layout.AutoGridLayoutKind.spec.withItems(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.layout.AutoGridLayoutKind.spec.withItemsMixin

```jsonnet
spec.layout.AutoGridLayoutKind.spec.withItemsMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.layout.AutoGridLayoutKind.spec.withMaxColumnCount

```jsonnet
spec.layout.AutoGridLayoutKind.spec.withMaxColumnCount(value=3)
```

PARAMETERS:

* **value** (`number`)
   - default value: `3`


####### fn spec.layout.AutoGridLayoutKind.spec.withRowHeight

```jsonnet
spec.layout.AutoGridLayoutKind.spec.withRowHeight(value)
```

PARAMETERS:

* **value** (`number`)


####### fn spec.layout.AutoGridLayoutKind.spec.withRowHeightMode

```jsonnet
spec.layout.AutoGridLayoutKind.spec.withRowHeightMode(value="standard")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"standard"`
   - valid values: `"short"`, `"standard"`, `"tall"`, `"custom"`


##### obj spec.layout.GridLayoutKind


###### fn spec.layout.GridLayoutKind.withKind

```jsonnet
spec.layout.GridLayoutKind.withKind()
```



###### fn spec.layout.GridLayoutKind.withSpec

```jsonnet
spec.layout.GridLayoutKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.layout.GridLayoutKind.withSpecMixin

```jsonnet
spec.layout.GridLayoutKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### obj spec.layout.GridLayoutKind.spec


####### fn spec.layout.GridLayoutKind.spec.withItems

```jsonnet
spec.layout.GridLayoutKind.spec.withItems(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.layout.GridLayoutKind.spec.withItemsMixin

```jsonnet
spec.layout.GridLayoutKind.spec.withItemsMixin(value)
```

PARAMETERS:

* **value** (`array`)


##### obj spec.layout.RowsLayoutKind


###### fn spec.layout.RowsLayoutKind.withKind

```jsonnet
spec.layout.RowsLayoutKind.withKind()
```



###### fn spec.layout.RowsLayoutKind.withSpec

```jsonnet
spec.layout.RowsLayoutKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.layout.RowsLayoutKind.withSpecMixin

```jsonnet
spec.layout.RowsLayoutKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### obj spec.layout.RowsLayoutKind.spec


####### fn spec.layout.RowsLayoutKind.spec.withRows

```jsonnet
spec.layout.RowsLayoutKind.spec.withRows(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.layout.RowsLayoutKind.spec.withRowsMixin

```jsonnet
spec.layout.RowsLayoutKind.spec.withRowsMixin(value)
```

PARAMETERS:

* **value** (`array`)


##### obj spec.layout.TabsLayoutKind


###### fn spec.layout.TabsLayoutKind.withKind

```jsonnet
spec.layout.TabsLayoutKind.withKind()
```



###### fn spec.layout.TabsLayoutKind.withSpec

```jsonnet
spec.layout.TabsLayoutKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.layout.TabsLayoutKind.withSpecMixin

```jsonnet
spec.layout.TabsLayoutKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### obj spec.layout.TabsLayoutKind.spec


####### fn spec.layout.TabsLayoutKind.spec.withTabs

```jsonnet
spec.layout.TabsLayoutKind.spec.withTabs(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.layout.TabsLayoutKind.spec.withTabsMixin

```jsonnet
spec.layout.TabsLayoutKind.spec.withTabsMixin(value)
```

PARAMETERS:

* **value** (`array`)


#### obj spec.preferences


##### fn spec.preferences.withLayout

```jsonnet
spec.preferences.withLayout(value)
```

PARAMETERS:

* **value** (`object`)

default layout template to be used when new containers are created
##### fn spec.preferences.withLayoutMixin

```jsonnet
spec.preferences.withLayoutMixin(value)
```

PARAMETERS:

* **value** (`object`)

default layout template to be used when new containers are created
##### obj spec.preferences.layout


###### fn spec.preferences.layout.withAutoGridLayoutKind

```jsonnet
spec.preferences.layout.withAutoGridLayoutKind(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.preferences.layout.withAutoGridLayoutKindMixin

```jsonnet
spec.preferences.layout.withAutoGridLayoutKindMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.preferences.layout.withGridLayoutKind

```jsonnet
spec.preferences.layout.withGridLayoutKind(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.preferences.layout.withGridLayoutKindMixin

```jsonnet
spec.preferences.layout.withGridLayoutKindMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### obj spec.preferences.layout.AutoGridLayoutKind


####### fn spec.preferences.layout.AutoGridLayoutKind.withKind

```jsonnet
spec.preferences.layout.AutoGridLayoutKind.withKind()
```



####### fn spec.preferences.layout.AutoGridLayoutKind.withSpec

```jsonnet
spec.preferences.layout.AutoGridLayoutKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)


####### fn spec.preferences.layout.AutoGridLayoutKind.withSpecMixin

```jsonnet
spec.preferences.layout.AutoGridLayoutKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


####### obj spec.preferences.layout.AutoGridLayoutKind.spec


######## fn spec.preferences.layout.AutoGridLayoutKind.spec.withColumnWidth

```jsonnet
spec.preferences.layout.AutoGridLayoutKind.spec.withColumnWidth(value)
```

PARAMETERS:

* **value** (`number`)


######## fn spec.preferences.layout.AutoGridLayoutKind.spec.withColumnWidthMode

```jsonnet
spec.preferences.layout.AutoGridLayoutKind.spec.withColumnWidthMode(value="standard")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"standard"`
   - valid values: `"narrow"`, `"standard"`, `"wide"`, `"custom"`


######## fn spec.preferences.layout.AutoGridLayoutKind.spec.withFillScreen

```jsonnet
spec.preferences.layout.AutoGridLayoutKind.spec.withFillScreen(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


######## fn spec.preferences.layout.AutoGridLayoutKind.spec.withItems

```jsonnet
spec.preferences.layout.AutoGridLayoutKind.spec.withItems(value)
```

PARAMETERS:

* **value** (`array`)


######## fn spec.preferences.layout.AutoGridLayoutKind.spec.withItemsMixin

```jsonnet
spec.preferences.layout.AutoGridLayoutKind.spec.withItemsMixin(value)
```

PARAMETERS:

* **value** (`array`)


######## fn spec.preferences.layout.AutoGridLayoutKind.spec.withMaxColumnCount

```jsonnet
spec.preferences.layout.AutoGridLayoutKind.spec.withMaxColumnCount(value=3)
```

PARAMETERS:

* **value** (`number`)
   - default value: `3`


######## fn spec.preferences.layout.AutoGridLayoutKind.spec.withRowHeight

```jsonnet
spec.preferences.layout.AutoGridLayoutKind.spec.withRowHeight(value)
```

PARAMETERS:

* **value** (`number`)


######## fn spec.preferences.layout.AutoGridLayoutKind.spec.withRowHeightMode

```jsonnet
spec.preferences.layout.AutoGridLayoutKind.spec.withRowHeightMode(value="standard")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"standard"`
   - valid values: `"short"`, `"standard"`, `"tall"`, `"custom"`


###### obj spec.preferences.layout.GridLayoutKind


####### fn spec.preferences.layout.GridLayoutKind.withKind

```jsonnet
spec.preferences.layout.GridLayoutKind.withKind()
```



####### fn spec.preferences.layout.GridLayoutKind.withSpec

```jsonnet
spec.preferences.layout.GridLayoutKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)


####### fn spec.preferences.layout.GridLayoutKind.withSpecMixin

```jsonnet
spec.preferences.layout.GridLayoutKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


####### obj spec.preferences.layout.GridLayoutKind.spec


######## fn spec.preferences.layout.GridLayoutKind.spec.withItems

```jsonnet
spec.preferences.layout.GridLayoutKind.spec.withItems(value)
```

PARAMETERS:

* **value** (`array`)


######## fn spec.preferences.layout.GridLayoutKind.spec.withItemsMixin

```jsonnet
spec.preferences.layout.GridLayoutKind.spec.withItemsMixin(value)
```

PARAMETERS:

* **value** (`array`)


#### obj spec.timeSettings


##### fn spec.timeSettings.withAutoRefresh

```jsonnet
spec.timeSettings.withAutoRefresh(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`

Refresh rate of dashboard. Represented via interval string, e.g. "5s", "1m", "1h", "1d".
v1: refresh
##### fn spec.timeSettings.withAutoRefreshIntervals

```jsonnet
spec.timeSettings.withAutoRefreshIntervals(value=["5s","10s","30s","1m","5m","15m","30m","1h","2h","1d"])
```

PARAMETERS:

* **value** (`array`)
   - default value: `["5s","10s","30s","1m","5m","15m","30m","1h","2h","1d"]`

Interval options available in the refresh picker dropdown.
v1: timepicker.refresh_intervals
##### fn spec.timeSettings.withAutoRefreshIntervalsMixin

```jsonnet
spec.timeSettings.withAutoRefreshIntervalsMixin(value=["5s","10s","30s","1m","5m","15m","30m","1h","2h","1d"])
```

PARAMETERS:

* **value** (`array`)
   - default value: `["5s","10s","30s","1m","5m","15m","30m","1h","2h","1d"]`

Interval options available in the refresh picker dropdown.
v1: timepicker.refresh_intervals
##### fn spec.timeSettings.withFiscalYearStartMonth

```jsonnet
spec.timeSettings.withFiscalYearStartMonth(value=0)
```

PARAMETERS:

* **value** (`integer`)
   - default value: `0`

The month that the fiscal year starts on. 0 = January, 11 = December
##### fn spec.timeSettings.withFrom

```jsonnet
spec.timeSettings.withFrom(value="now-6h")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"now-6h"`

Start time range for dashboard.
Accepted values are relative time strings like "now-6h" or absolute time strings like "2020-07-10T08:00:00.000Z".
##### fn spec.timeSettings.withHideTimepicker

```jsonnet
spec.timeSettings.withHideTimepicker(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether timepicker is visible or not.
v1: timepicker.hidden
##### fn spec.timeSettings.withNowDelay

```jsonnet
spec.timeSettings.withNowDelay(value)
```

PARAMETERS:

* **value** (`string`)

Override the now time by entering a time delay. Use this option to accommodate known delays in data aggregation to avoid null values.
v1: timepicker.nowDelay
##### fn spec.timeSettings.withQuickRanges

```jsonnet
spec.timeSettings.withQuickRanges(value)
```

PARAMETERS:

* **value** (`array`)

Selectable options available in the time picker dropdown. Has no effect on provisioned dashboard.
v1: timepicker.quick_ranges , not exposed in the UI
##### fn spec.timeSettings.withQuickRangesMixin

```jsonnet
spec.timeSettings.withQuickRangesMixin(value)
```

PARAMETERS:

* **value** (`array`)

Selectable options available in the time picker dropdown. Has no effect on provisioned dashboard.
v1: timepicker.quick_ranges , not exposed in the UI
##### fn spec.timeSettings.withTimezone

```jsonnet
spec.timeSettings.withTimezone(value="browser")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"browser"`

Timezone of dashboard. Accepted values are IANA TZDB zone ID or "browser" or "utc".
##### fn spec.timeSettings.withTo

```jsonnet
spec.timeSettings.withTo(value="now")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"now"`

End time range for dashboard.
Accepted values are relative time strings like "now-6h" or absolute time strings like "2020-07-10T08:00:00.000Z".
##### fn spec.timeSettings.withWeekStart

```jsonnet
spec.timeSettings.withWeekStart(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"saturday"`, `"monday"`, `"sunday"`

Day when the week starts. Expressed by the name of the day in lowercase, e.g. "monday".
#### obj spec.variables


##### obj spec.variables.AdhocVariableKind


###### fn spec.variables.AdhocVariableKind.withDatasource

```jsonnet
spec.variables.AdhocVariableKind.withDatasource(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.variables.AdhocVariableKind.withDatasourceMixin

```jsonnet
spec.variables.AdhocVariableKind.withDatasourceMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.variables.AdhocVariableKind.withGroup

```jsonnet
spec.variables.AdhocVariableKind.withGroup(value)
```

PARAMETERS:

* **value** (`string`)


###### fn spec.variables.AdhocVariableKind.withKind

```jsonnet
spec.variables.AdhocVariableKind.withKind()
```



###### fn spec.variables.AdhocVariableKind.withLabels

```jsonnet
spec.variables.AdhocVariableKind.withLabels(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.variables.AdhocVariableKind.withLabelsMixin

```jsonnet
spec.variables.AdhocVariableKind.withLabelsMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.variables.AdhocVariableKind.withSpec

```jsonnet
spec.variables.AdhocVariableKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)

Adhoc variable specification
###### fn spec.variables.AdhocVariableKind.withSpecMixin

```jsonnet
spec.variables.AdhocVariableKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)

Adhoc variable specification
###### obj spec.variables.AdhocVariableKind.datasource


####### fn spec.variables.AdhocVariableKind.datasource.withName

```jsonnet
spec.variables.AdhocVariableKind.datasource.withName(value)
```

PARAMETERS:

* **value** (`string`)


###### obj spec.variables.AdhocVariableKind.spec


####### fn spec.variables.AdhocVariableKind.spec.withAllowCustomValue

```jsonnet
spec.variables.AdhocVariableKind.spec.withAllowCustomValue(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.AdhocVariableKind.spec.withBaseFilters

```jsonnet
spec.variables.AdhocVariableKind.spec.withBaseFilters(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.AdhocVariableKind.spec.withBaseFiltersMixin

```jsonnet
spec.variables.AdhocVariableKind.spec.withBaseFiltersMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.AdhocVariableKind.spec.withDefaultKeys

```jsonnet
spec.variables.AdhocVariableKind.spec.withDefaultKeys(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.AdhocVariableKind.spec.withDefaultKeysMixin

```jsonnet
spec.variables.AdhocVariableKind.spec.withDefaultKeysMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.AdhocVariableKind.spec.withDescription

```jsonnet
spec.variables.AdhocVariableKind.spec.withDescription(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.AdhocVariableKind.spec.withEnableGroupBy

```jsonnet
spec.variables.AdhocVariableKind.spec.withEnableGroupBy(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether the group-by operator is enabled in the ad hoc filter combobox.
####### fn spec.variables.AdhocVariableKind.spec.withFilters

```jsonnet
spec.variables.AdhocVariableKind.spec.withFilters(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.AdhocVariableKind.spec.withFiltersMixin

```jsonnet
spec.variables.AdhocVariableKind.spec.withFiltersMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.AdhocVariableKind.spec.withHide

```jsonnet
spec.variables.AdhocVariableKind.spec.withHide(value="dontHide")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"dontHide"`
   - valid values: `"dontHide"`, `"hideLabel"`, `"hideVariable"`, `"inControlsMenu"`

Determine if the variable shows on dashboard
Accepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).
####### fn spec.variables.AdhocVariableKind.spec.withLabel

```jsonnet
spec.variables.AdhocVariableKind.spec.withLabel(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.AdhocVariableKind.spec.withName

```jsonnet
spec.variables.AdhocVariableKind.spec.withName(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.AdhocVariableKind.spec.withOrigin

```jsonnet
spec.variables.AdhocVariableKind.spec.withOrigin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.AdhocVariableKind.spec.withOriginMixin

```jsonnet
spec.variables.AdhocVariableKind.spec.withOriginMixin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.AdhocVariableKind.spec.withSkipUrlSync

```jsonnet
spec.variables.AdhocVariableKind.spec.withSkipUrlSync(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### obj spec.variables.AdhocVariableKind.spec.origin


######## fn spec.variables.AdhocVariableKind.spec.origin.withGroup

```jsonnet
spec.variables.AdhocVariableKind.spec.origin.withGroup(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
######## fn spec.variables.AdhocVariableKind.spec.origin.withType

```jsonnet
spec.variables.AdhocVariableKind.spec.origin.withType()
```



##### obj spec.variables.ConstantVariableKind


###### fn spec.variables.ConstantVariableKind.withKind

```jsonnet
spec.variables.ConstantVariableKind.withKind()
```



###### fn spec.variables.ConstantVariableKind.withSpec

```jsonnet
spec.variables.ConstantVariableKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)

Constant variable specification
###### fn spec.variables.ConstantVariableKind.withSpecMixin

```jsonnet
spec.variables.ConstantVariableKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)

Constant variable specification
###### obj spec.variables.ConstantVariableKind.spec


####### fn spec.variables.ConstantVariableKind.spec.withCurrent

```jsonnet
spec.variables.ConstantVariableKind.spec.withCurrent(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.ConstantVariableKind.spec.withCurrentMixin

```jsonnet
spec.variables.ConstantVariableKind.spec.withCurrentMixin(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.ConstantVariableKind.spec.withDescription

```jsonnet
spec.variables.ConstantVariableKind.spec.withDescription(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.ConstantVariableKind.spec.withHide

```jsonnet
spec.variables.ConstantVariableKind.spec.withHide(value="dontHide")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"dontHide"`
   - valid values: `"dontHide"`, `"hideLabel"`, `"hideVariable"`, `"inControlsMenu"`

Determine if the variable shows on dashboard
Accepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).
####### fn spec.variables.ConstantVariableKind.spec.withLabel

```jsonnet
spec.variables.ConstantVariableKind.spec.withLabel(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.ConstantVariableKind.spec.withName

```jsonnet
spec.variables.ConstantVariableKind.spec.withName(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.ConstantVariableKind.spec.withOrigin

```jsonnet
spec.variables.ConstantVariableKind.spec.withOrigin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.ConstantVariableKind.spec.withOriginMixin

```jsonnet
spec.variables.ConstantVariableKind.spec.withOriginMixin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.ConstantVariableKind.spec.withQuery

```jsonnet
spec.variables.ConstantVariableKind.spec.withQuery(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.ConstantVariableKind.spec.withSkipUrlSync

```jsonnet
spec.variables.ConstantVariableKind.spec.withSkipUrlSync(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### obj spec.variables.ConstantVariableKind.spec.current


######## fn spec.variables.ConstantVariableKind.spec.current.withProperties

```jsonnet
spec.variables.ConstantVariableKind.spec.current.withProperties(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.ConstantVariableKind.spec.current.withPropertiesMixin

```jsonnet
spec.variables.ConstantVariableKind.spec.current.withPropertiesMixin(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.ConstantVariableKind.spec.current.withSelected

```jsonnet
spec.variables.ConstantVariableKind.spec.current.withSelected(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether the option is selected or not
######## fn spec.variables.ConstantVariableKind.spec.current.withText

```jsonnet
spec.variables.ConstantVariableKind.spec.current.withText(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.ConstantVariableKind.spec.current.withTextMixin

```jsonnet
spec.variables.ConstantVariableKind.spec.current.withTextMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.ConstantVariableKind.spec.current.withValue

```jsonnet
spec.variables.ConstantVariableKind.spec.current.withValue(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
######## fn spec.variables.ConstantVariableKind.spec.current.withValueMixin

```jsonnet
spec.variables.ConstantVariableKind.spec.current.withValueMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
####### obj spec.variables.ConstantVariableKind.spec.origin


######## fn spec.variables.ConstantVariableKind.spec.origin.withGroup

```jsonnet
spec.variables.ConstantVariableKind.spec.origin.withGroup(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
######## fn spec.variables.ConstantVariableKind.spec.origin.withType

```jsonnet
spec.variables.ConstantVariableKind.spec.origin.withType()
```



##### obj spec.variables.CustomVariableKind


###### fn spec.variables.CustomVariableKind.withKind

```jsonnet
spec.variables.CustomVariableKind.withKind()
```



###### fn spec.variables.CustomVariableKind.withSpec

```jsonnet
spec.variables.CustomVariableKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)

Custom variable specification
###### fn spec.variables.CustomVariableKind.withSpecMixin

```jsonnet
spec.variables.CustomVariableKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)

Custom variable specification
###### obj spec.variables.CustomVariableKind.spec


####### fn spec.variables.CustomVariableKind.spec.withAllValue

```jsonnet
spec.variables.CustomVariableKind.spec.withAllValue(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.CustomVariableKind.spec.withAllowCustomValue

```jsonnet
spec.variables.CustomVariableKind.spec.withAllowCustomValue(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.CustomVariableKind.spec.withCurrent

```jsonnet
spec.variables.CustomVariableKind.spec.withCurrent(value)
```

PARAMETERS:

* **value** (`object`)

Variable option specification
####### fn spec.variables.CustomVariableKind.spec.withCurrentMixin

```jsonnet
spec.variables.CustomVariableKind.spec.withCurrentMixin(value)
```

PARAMETERS:

* **value** (`object`)

Variable option specification
####### fn spec.variables.CustomVariableKind.spec.withDescription

```jsonnet
spec.variables.CustomVariableKind.spec.withDescription(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.CustomVariableKind.spec.withHide

```jsonnet
spec.variables.CustomVariableKind.spec.withHide(value="dontHide")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"dontHide"`
   - valid values: `"dontHide"`, `"hideLabel"`, `"hideVariable"`, `"inControlsMenu"`

Determine if the variable shows on dashboard
Accepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).
####### fn spec.variables.CustomVariableKind.spec.withIncludeAll

```jsonnet
spec.variables.CustomVariableKind.spec.withIncludeAll(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.CustomVariableKind.spec.withLabel

```jsonnet
spec.variables.CustomVariableKind.spec.withLabel(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.CustomVariableKind.spec.withMulti

```jsonnet
spec.variables.CustomVariableKind.spec.withMulti(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.CustomVariableKind.spec.withName

```jsonnet
spec.variables.CustomVariableKind.spec.withName(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.CustomVariableKind.spec.withOptions

```jsonnet
spec.variables.CustomVariableKind.spec.withOptions(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.CustomVariableKind.spec.withOptionsMixin

```jsonnet
spec.variables.CustomVariableKind.spec.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.CustomVariableKind.spec.withOrigin

```jsonnet
spec.variables.CustomVariableKind.spec.withOrigin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.CustomVariableKind.spec.withOriginMixin

```jsonnet
spec.variables.CustomVariableKind.spec.withOriginMixin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.CustomVariableKind.spec.withQuery

```jsonnet
spec.variables.CustomVariableKind.spec.withQuery(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.CustomVariableKind.spec.withSkipUrlSync

```jsonnet
spec.variables.CustomVariableKind.spec.withSkipUrlSync(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.CustomVariableKind.spec.withValuesFormat

```jsonnet
spec.variables.CustomVariableKind.spec.withValuesFormat(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"csv"`, `"json"`


####### obj spec.variables.CustomVariableKind.spec.current


######## fn spec.variables.CustomVariableKind.spec.current.withProperties

```jsonnet
spec.variables.CustomVariableKind.spec.current.withProperties(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.CustomVariableKind.spec.current.withPropertiesMixin

```jsonnet
spec.variables.CustomVariableKind.spec.current.withPropertiesMixin(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.CustomVariableKind.spec.current.withSelected

```jsonnet
spec.variables.CustomVariableKind.spec.current.withSelected(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether the option is selected or not
######## fn spec.variables.CustomVariableKind.spec.current.withText

```jsonnet
spec.variables.CustomVariableKind.spec.current.withText(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.CustomVariableKind.spec.current.withTextMixin

```jsonnet
spec.variables.CustomVariableKind.spec.current.withTextMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.CustomVariableKind.spec.current.withValue

```jsonnet
spec.variables.CustomVariableKind.spec.current.withValue(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
######## fn spec.variables.CustomVariableKind.spec.current.withValueMixin

```jsonnet
spec.variables.CustomVariableKind.spec.current.withValueMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
####### obj spec.variables.CustomVariableKind.spec.origin


######## fn spec.variables.CustomVariableKind.spec.origin.withGroup

```jsonnet
spec.variables.CustomVariableKind.spec.origin.withGroup(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
######## fn spec.variables.CustomVariableKind.spec.origin.withType

```jsonnet
spec.variables.CustomVariableKind.spec.origin.withType()
```



##### obj spec.variables.DatasourceVariableKind


###### fn spec.variables.DatasourceVariableKind.withKind

```jsonnet
spec.variables.DatasourceVariableKind.withKind()
```



###### fn spec.variables.DatasourceVariableKind.withSpec

```jsonnet
spec.variables.DatasourceVariableKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)

Datasource variable specification
###### fn spec.variables.DatasourceVariableKind.withSpecMixin

```jsonnet
spec.variables.DatasourceVariableKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)

Datasource variable specification
###### obj spec.variables.DatasourceVariableKind.spec


####### fn spec.variables.DatasourceVariableKind.spec.withAllValue

```jsonnet
spec.variables.DatasourceVariableKind.spec.withAllValue(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.DatasourceVariableKind.spec.withAllowCustomValue

```jsonnet
spec.variables.DatasourceVariableKind.spec.withAllowCustomValue(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.DatasourceVariableKind.spec.withCurrent

```jsonnet
spec.variables.DatasourceVariableKind.spec.withCurrent(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.DatasourceVariableKind.spec.withCurrentMixin

```jsonnet
spec.variables.DatasourceVariableKind.spec.withCurrentMixin(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.DatasourceVariableKind.spec.withDescription

```jsonnet
spec.variables.DatasourceVariableKind.spec.withDescription(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.DatasourceVariableKind.spec.withHide

```jsonnet
spec.variables.DatasourceVariableKind.spec.withHide(value="dontHide")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"dontHide"`
   - valid values: `"dontHide"`, `"hideLabel"`, `"hideVariable"`, `"inControlsMenu"`

Determine if the variable shows on dashboard
Accepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).
####### fn spec.variables.DatasourceVariableKind.spec.withIncludeAll

```jsonnet
spec.variables.DatasourceVariableKind.spec.withIncludeAll(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.DatasourceVariableKind.spec.withLabel

```jsonnet
spec.variables.DatasourceVariableKind.spec.withLabel(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.DatasourceVariableKind.spec.withMulti

```jsonnet
spec.variables.DatasourceVariableKind.spec.withMulti(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.DatasourceVariableKind.spec.withName

```jsonnet
spec.variables.DatasourceVariableKind.spec.withName(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.DatasourceVariableKind.spec.withOptions

```jsonnet
spec.variables.DatasourceVariableKind.spec.withOptions(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.DatasourceVariableKind.spec.withOptionsMixin

```jsonnet
spec.variables.DatasourceVariableKind.spec.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.DatasourceVariableKind.spec.withOrigin

```jsonnet
spec.variables.DatasourceVariableKind.spec.withOrigin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.DatasourceVariableKind.spec.withOriginMixin

```jsonnet
spec.variables.DatasourceVariableKind.spec.withOriginMixin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.DatasourceVariableKind.spec.withPluginId

```jsonnet
spec.variables.DatasourceVariableKind.spec.withPluginId(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.DatasourceVariableKind.spec.withRefresh

```jsonnet
spec.variables.DatasourceVariableKind.spec.withRefresh(value="never")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"never"`
   - valid values: `"never"`, `"onDashboardLoad"`, `"onTimeRangeChanged"`

Options to config when to refresh a variable
`never`: Never refresh the variable
`onDashboardLoad`: Queries the data source every time the dashboard loads.
`onTimeRangeChanged`: Queries the data source when the dashboard time range changes.
####### fn spec.variables.DatasourceVariableKind.spec.withRegex

```jsonnet
spec.variables.DatasourceVariableKind.spec.withRegex(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.DatasourceVariableKind.spec.withSkipUrlSync

```jsonnet
spec.variables.DatasourceVariableKind.spec.withSkipUrlSync(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### obj spec.variables.DatasourceVariableKind.spec.current


######## fn spec.variables.DatasourceVariableKind.spec.current.withProperties

```jsonnet
spec.variables.DatasourceVariableKind.spec.current.withProperties(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.DatasourceVariableKind.spec.current.withPropertiesMixin

```jsonnet
spec.variables.DatasourceVariableKind.spec.current.withPropertiesMixin(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.DatasourceVariableKind.spec.current.withSelected

```jsonnet
spec.variables.DatasourceVariableKind.spec.current.withSelected(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether the option is selected or not
######## fn spec.variables.DatasourceVariableKind.spec.current.withText

```jsonnet
spec.variables.DatasourceVariableKind.spec.current.withText(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.DatasourceVariableKind.spec.current.withTextMixin

```jsonnet
spec.variables.DatasourceVariableKind.spec.current.withTextMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.DatasourceVariableKind.spec.current.withValue

```jsonnet
spec.variables.DatasourceVariableKind.spec.current.withValue(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
######## fn spec.variables.DatasourceVariableKind.spec.current.withValueMixin

```jsonnet
spec.variables.DatasourceVariableKind.spec.current.withValueMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
####### obj spec.variables.DatasourceVariableKind.spec.origin


######## fn spec.variables.DatasourceVariableKind.spec.origin.withGroup

```jsonnet
spec.variables.DatasourceVariableKind.spec.origin.withGroup(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
######## fn spec.variables.DatasourceVariableKind.spec.origin.withType

```jsonnet
spec.variables.DatasourceVariableKind.spec.origin.withType()
```



##### obj spec.variables.GroupByVariableKind


###### fn spec.variables.GroupByVariableKind.withDatasource

```jsonnet
spec.variables.GroupByVariableKind.withDatasource(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.variables.GroupByVariableKind.withDatasourceMixin

```jsonnet
spec.variables.GroupByVariableKind.withDatasourceMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.variables.GroupByVariableKind.withGroup

```jsonnet
spec.variables.GroupByVariableKind.withGroup(value)
```

PARAMETERS:

* **value** (`string`)


###### fn spec.variables.GroupByVariableKind.withKind

```jsonnet
spec.variables.GroupByVariableKind.withKind()
```



###### fn spec.variables.GroupByVariableKind.withLabels

```jsonnet
spec.variables.GroupByVariableKind.withLabels(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.variables.GroupByVariableKind.withLabelsMixin

```jsonnet
spec.variables.GroupByVariableKind.withLabelsMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.variables.GroupByVariableKind.withSpec

```jsonnet
spec.variables.GroupByVariableKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)

GroupBy variable specification
###### fn spec.variables.GroupByVariableKind.withSpecMixin

```jsonnet
spec.variables.GroupByVariableKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)

GroupBy variable specification
###### obj spec.variables.GroupByVariableKind.datasource


####### fn spec.variables.GroupByVariableKind.datasource.withName

```jsonnet
spec.variables.GroupByVariableKind.datasource.withName(value)
```

PARAMETERS:

* **value** (`string`)


###### obj spec.variables.GroupByVariableKind.spec


####### fn spec.variables.GroupByVariableKind.spec.withCurrent

```jsonnet
spec.variables.GroupByVariableKind.spec.withCurrent(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.GroupByVariableKind.spec.withCurrentMixin

```jsonnet
spec.variables.GroupByVariableKind.spec.withCurrentMixin(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.GroupByVariableKind.spec.withDefaultValue

```jsonnet
spec.variables.GroupByVariableKind.spec.withDefaultValue(value)
```

PARAMETERS:

* **value** (`object`)

Variable option specification
####### fn spec.variables.GroupByVariableKind.spec.withDefaultValueMixin

```jsonnet
spec.variables.GroupByVariableKind.spec.withDefaultValueMixin(value)
```

PARAMETERS:

* **value** (`object`)

Variable option specification
####### fn spec.variables.GroupByVariableKind.spec.withDescription

```jsonnet
spec.variables.GroupByVariableKind.spec.withDescription(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.GroupByVariableKind.spec.withHide

```jsonnet
spec.variables.GroupByVariableKind.spec.withHide(value="dontHide")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"dontHide"`
   - valid values: `"dontHide"`, `"hideLabel"`, `"hideVariable"`, `"inControlsMenu"`

Determine if the variable shows on dashboard
Accepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).
####### fn spec.variables.GroupByVariableKind.spec.withLabel

```jsonnet
spec.variables.GroupByVariableKind.spec.withLabel(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.GroupByVariableKind.spec.withMulti

```jsonnet
spec.variables.GroupByVariableKind.spec.withMulti(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.GroupByVariableKind.spec.withName

```jsonnet
spec.variables.GroupByVariableKind.spec.withName(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.GroupByVariableKind.spec.withOptions

```jsonnet
spec.variables.GroupByVariableKind.spec.withOptions(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.GroupByVariableKind.spec.withOptionsMixin

```jsonnet
spec.variables.GroupByVariableKind.spec.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.GroupByVariableKind.spec.withOrigin

```jsonnet
spec.variables.GroupByVariableKind.spec.withOrigin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.GroupByVariableKind.spec.withOriginMixin

```jsonnet
spec.variables.GroupByVariableKind.spec.withOriginMixin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.GroupByVariableKind.spec.withSkipUrlSync

```jsonnet
spec.variables.GroupByVariableKind.spec.withSkipUrlSync(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### obj spec.variables.GroupByVariableKind.spec.current


######## fn spec.variables.GroupByVariableKind.spec.current.withProperties

```jsonnet
spec.variables.GroupByVariableKind.spec.current.withProperties(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.GroupByVariableKind.spec.current.withPropertiesMixin

```jsonnet
spec.variables.GroupByVariableKind.spec.current.withPropertiesMixin(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.GroupByVariableKind.spec.current.withSelected

```jsonnet
spec.variables.GroupByVariableKind.spec.current.withSelected(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether the option is selected or not
######## fn spec.variables.GroupByVariableKind.spec.current.withText

```jsonnet
spec.variables.GroupByVariableKind.spec.current.withText(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.GroupByVariableKind.spec.current.withTextMixin

```jsonnet
spec.variables.GroupByVariableKind.spec.current.withTextMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.GroupByVariableKind.spec.current.withValue

```jsonnet
spec.variables.GroupByVariableKind.spec.current.withValue(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
######## fn spec.variables.GroupByVariableKind.spec.current.withValueMixin

```jsonnet
spec.variables.GroupByVariableKind.spec.current.withValueMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
####### obj spec.variables.GroupByVariableKind.spec.defaultValue


######## fn spec.variables.GroupByVariableKind.spec.defaultValue.withProperties

```jsonnet
spec.variables.GroupByVariableKind.spec.defaultValue.withProperties(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.GroupByVariableKind.spec.defaultValue.withPropertiesMixin

```jsonnet
spec.variables.GroupByVariableKind.spec.defaultValue.withPropertiesMixin(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.GroupByVariableKind.spec.defaultValue.withSelected

```jsonnet
spec.variables.GroupByVariableKind.spec.defaultValue.withSelected(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether the option is selected or not
######## fn spec.variables.GroupByVariableKind.spec.defaultValue.withText

```jsonnet
spec.variables.GroupByVariableKind.spec.defaultValue.withText(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.GroupByVariableKind.spec.defaultValue.withTextMixin

```jsonnet
spec.variables.GroupByVariableKind.spec.defaultValue.withTextMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.GroupByVariableKind.spec.defaultValue.withValue

```jsonnet
spec.variables.GroupByVariableKind.spec.defaultValue.withValue(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
######## fn spec.variables.GroupByVariableKind.spec.defaultValue.withValueMixin

```jsonnet
spec.variables.GroupByVariableKind.spec.defaultValue.withValueMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
####### obj spec.variables.GroupByVariableKind.spec.origin


######## fn spec.variables.GroupByVariableKind.spec.origin.withGroup

```jsonnet
spec.variables.GroupByVariableKind.spec.origin.withGroup(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
######## fn spec.variables.GroupByVariableKind.spec.origin.withType

```jsonnet
spec.variables.GroupByVariableKind.spec.origin.withType()
```



##### obj spec.variables.IntervalVariableKind


###### fn spec.variables.IntervalVariableKind.withKind

```jsonnet
spec.variables.IntervalVariableKind.withKind()
```



###### fn spec.variables.IntervalVariableKind.withSpec

```jsonnet
spec.variables.IntervalVariableKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)

Interval variable specification
###### fn spec.variables.IntervalVariableKind.withSpecMixin

```jsonnet
spec.variables.IntervalVariableKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)

Interval variable specification
###### obj spec.variables.IntervalVariableKind.spec


####### fn spec.variables.IntervalVariableKind.spec.withAuto

```jsonnet
spec.variables.IntervalVariableKind.spec.withAuto(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.IntervalVariableKind.spec.withAutoCount

```jsonnet
spec.variables.IntervalVariableKind.spec.withAutoCount(value=0)
```

PARAMETERS:

* **value** (`integer`)
   - default value: `0`


####### fn spec.variables.IntervalVariableKind.spec.withAutoMin

```jsonnet
spec.variables.IntervalVariableKind.spec.withAutoMin(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.IntervalVariableKind.spec.withCurrent

```jsonnet
spec.variables.IntervalVariableKind.spec.withCurrent(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.IntervalVariableKind.spec.withCurrentMixin

```jsonnet
spec.variables.IntervalVariableKind.spec.withCurrentMixin(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.IntervalVariableKind.spec.withDescription

```jsonnet
spec.variables.IntervalVariableKind.spec.withDescription(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.IntervalVariableKind.spec.withHide

```jsonnet
spec.variables.IntervalVariableKind.spec.withHide(value="dontHide")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"dontHide"`
   - valid values: `"dontHide"`, `"hideLabel"`, `"hideVariable"`, `"inControlsMenu"`

Determine if the variable shows on dashboard
Accepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).
####### fn spec.variables.IntervalVariableKind.spec.withLabel

```jsonnet
spec.variables.IntervalVariableKind.spec.withLabel(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.IntervalVariableKind.spec.withName

```jsonnet
spec.variables.IntervalVariableKind.spec.withName(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.IntervalVariableKind.spec.withOptions

```jsonnet
spec.variables.IntervalVariableKind.spec.withOptions(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.IntervalVariableKind.spec.withOptionsMixin

```jsonnet
spec.variables.IntervalVariableKind.spec.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.IntervalVariableKind.spec.withOrigin

```jsonnet
spec.variables.IntervalVariableKind.spec.withOrigin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.IntervalVariableKind.spec.withOriginMixin

```jsonnet
spec.variables.IntervalVariableKind.spec.withOriginMixin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.IntervalVariableKind.spec.withQuery

```jsonnet
spec.variables.IntervalVariableKind.spec.withQuery(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.IntervalVariableKind.spec.withRefresh

```jsonnet
spec.variables.IntervalVariableKind.spec.withRefresh()
```



####### fn spec.variables.IntervalVariableKind.spec.withSkipUrlSync

```jsonnet
spec.variables.IntervalVariableKind.spec.withSkipUrlSync(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### obj spec.variables.IntervalVariableKind.spec.current


######## fn spec.variables.IntervalVariableKind.spec.current.withProperties

```jsonnet
spec.variables.IntervalVariableKind.spec.current.withProperties(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.IntervalVariableKind.spec.current.withPropertiesMixin

```jsonnet
spec.variables.IntervalVariableKind.spec.current.withPropertiesMixin(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.IntervalVariableKind.spec.current.withSelected

```jsonnet
spec.variables.IntervalVariableKind.spec.current.withSelected(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether the option is selected or not
######## fn spec.variables.IntervalVariableKind.spec.current.withText

```jsonnet
spec.variables.IntervalVariableKind.spec.current.withText(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.IntervalVariableKind.spec.current.withTextMixin

```jsonnet
spec.variables.IntervalVariableKind.spec.current.withTextMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.IntervalVariableKind.spec.current.withValue

```jsonnet
spec.variables.IntervalVariableKind.spec.current.withValue(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
######## fn spec.variables.IntervalVariableKind.spec.current.withValueMixin

```jsonnet
spec.variables.IntervalVariableKind.spec.current.withValueMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
####### obj spec.variables.IntervalVariableKind.spec.origin


######## fn spec.variables.IntervalVariableKind.spec.origin.withGroup

```jsonnet
spec.variables.IntervalVariableKind.spec.origin.withGroup(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
######## fn spec.variables.IntervalVariableKind.spec.origin.withType

```jsonnet
spec.variables.IntervalVariableKind.spec.origin.withType()
```



##### obj spec.variables.QueryVariableKind


###### fn spec.variables.QueryVariableKind.withKind

```jsonnet
spec.variables.QueryVariableKind.withKind()
```



###### fn spec.variables.QueryVariableKind.withSpec

```jsonnet
spec.variables.QueryVariableKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)

Query variable specification
###### fn spec.variables.QueryVariableKind.withSpecMixin

```jsonnet
spec.variables.QueryVariableKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)

Query variable specification
###### obj spec.variables.QueryVariableKind.spec


####### fn spec.variables.QueryVariableKind.spec.withAllValue

```jsonnet
spec.variables.QueryVariableKind.spec.withAllValue(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.QueryVariableKind.spec.withAllowCustomValue

```jsonnet
spec.variables.QueryVariableKind.spec.withAllowCustomValue(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.QueryVariableKind.spec.withCurrent

```jsonnet
spec.variables.QueryVariableKind.spec.withCurrent(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.QueryVariableKind.spec.withCurrentMixin

```jsonnet
spec.variables.QueryVariableKind.spec.withCurrentMixin(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.QueryVariableKind.spec.withDefinition

```jsonnet
spec.variables.QueryVariableKind.spec.withDefinition(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.QueryVariableKind.spec.withDescription

```jsonnet
spec.variables.QueryVariableKind.spec.withDescription(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.QueryVariableKind.spec.withHide

```jsonnet
spec.variables.QueryVariableKind.spec.withHide(value="dontHide")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"dontHide"`
   - valid values: `"dontHide"`, `"hideLabel"`, `"hideVariable"`, `"inControlsMenu"`

Determine if the variable shows on dashboard
Accepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).
####### fn spec.variables.QueryVariableKind.spec.withIncludeAll

```jsonnet
spec.variables.QueryVariableKind.spec.withIncludeAll(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.QueryVariableKind.spec.withLabel

```jsonnet
spec.variables.QueryVariableKind.spec.withLabel(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.QueryVariableKind.spec.withMulti

```jsonnet
spec.variables.QueryVariableKind.spec.withMulti(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.QueryVariableKind.spec.withName

```jsonnet
spec.variables.QueryVariableKind.spec.withName(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.QueryVariableKind.spec.withOptions

```jsonnet
spec.variables.QueryVariableKind.spec.withOptions(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.QueryVariableKind.spec.withOptionsMixin

```jsonnet
spec.variables.QueryVariableKind.spec.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.QueryVariableKind.spec.withOrigin

```jsonnet
spec.variables.QueryVariableKind.spec.withOrigin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.QueryVariableKind.spec.withOriginMixin

```jsonnet
spec.variables.QueryVariableKind.spec.withOriginMixin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.QueryVariableKind.spec.withPlaceholder

```jsonnet
spec.variables.QueryVariableKind.spec.withPlaceholder(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.QueryVariableKind.spec.withQuery

```jsonnet
spec.variables.QueryVariableKind.spec.withQuery(value)
```

PARAMETERS:

* **value** (`object`)


####### fn spec.variables.QueryVariableKind.spec.withQueryMixin

```jsonnet
spec.variables.QueryVariableKind.spec.withQueryMixin(value)
```

PARAMETERS:

* **value** (`object`)


####### fn spec.variables.QueryVariableKind.spec.withRefresh

```jsonnet
spec.variables.QueryVariableKind.spec.withRefresh(value="never")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"never"`
   - valid values: `"never"`, `"onDashboardLoad"`, `"onTimeRangeChanged"`

Options to config when to refresh a variable
`never`: Never refresh the variable
`onDashboardLoad`: Queries the data source every time the dashboard loads.
`onTimeRangeChanged`: Queries the data source when the dashboard time range changes.
####### fn spec.variables.QueryVariableKind.spec.withRegex

```jsonnet
spec.variables.QueryVariableKind.spec.withRegex(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.QueryVariableKind.spec.withRegexApplyTo

```jsonnet
spec.variables.QueryVariableKind.spec.withRegexApplyTo(value="value")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"value"`
   - valid values: `"value"`, `"text"`

Determine whether regex applies to variable value or display text
Accepted values are `value` (apply to value used in queries) or `text` (apply to display text shown to users)
####### fn spec.variables.QueryVariableKind.spec.withSkipUrlSync

```jsonnet
spec.variables.QueryVariableKind.spec.withSkipUrlSync(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn spec.variables.QueryVariableKind.spec.withSort

```jsonnet
spec.variables.QueryVariableKind.spec.withSort(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"disabled"`, `"alphabeticalAsc"`, `"alphabeticalDesc"`, `"numericalAsc"`, `"numericalDesc"`, `"alphabeticalCaseInsensitiveAsc"`, `"alphabeticalCaseInsensitiveDesc"`, `"naturalAsc"`, `"naturalDesc"`

Sort variable options
Accepted values are:
`disabled`: No sorting
`alphabeticalAsc`: Alphabetical ASC
`alphabeticalDesc`: Alphabetical DESC
`numericalAsc`: Numerical ASC
`numericalDesc`: Numerical DESC
`alphabeticalCaseInsensitiveAsc`: Alphabetical Case Insensitive ASC
`alphabeticalCaseInsensitiveDesc`: Alphabetical Case Insensitive DESC
`naturalAsc`: Natural ASC
`naturalDesc`: Natural DESC
VariableSort enum with default value
####### fn spec.variables.QueryVariableKind.spec.withStaticOptions

```jsonnet
spec.variables.QueryVariableKind.spec.withStaticOptions(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.QueryVariableKind.spec.withStaticOptionsMixin

```jsonnet
spec.variables.QueryVariableKind.spec.withStaticOptionsMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn spec.variables.QueryVariableKind.spec.withStaticOptionsOrder

```jsonnet
spec.variables.QueryVariableKind.spec.withStaticOptionsOrder(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"before"`, `"after"`, `"sorted"`


####### obj spec.variables.QueryVariableKind.spec.current


######## fn spec.variables.QueryVariableKind.spec.current.withProperties

```jsonnet
spec.variables.QueryVariableKind.spec.current.withProperties(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.QueryVariableKind.spec.current.withPropertiesMixin

```jsonnet
spec.variables.QueryVariableKind.spec.current.withPropertiesMixin(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.QueryVariableKind.spec.current.withSelected

```jsonnet
spec.variables.QueryVariableKind.spec.current.withSelected(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether the option is selected or not
######## fn spec.variables.QueryVariableKind.spec.current.withText

```jsonnet
spec.variables.QueryVariableKind.spec.current.withText(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.QueryVariableKind.spec.current.withTextMixin

```jsonnet
spec.variables.QueryVariableKind.spec.current.withTextMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.QueryVariableKind.spec.current.withValue

```jsonnet
spec.variables.QueryVariableKind.spec.current.withValue(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
######## fn spec.variables.QueryVariableKind.spec.current.withValueMixin

```jsonnet
spec.variables.QueryVariableKind.spec.current.withValueMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
####### obj spec.variables.QueryVariableKind.spec.origin


######## fn spec.variables.QueryVariableKind.spec.origin.withGroup

```jsonnet
spec.variables.QueryVariableKind.spec.origin.withGroup(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
######## fn spec.variables.QueryVariableKind.spec.origin.withType

```jsonnet
spec.variables.QueryVariableKind.spec.origin.withType()
```



####### obj spec.variables.QueryVariableKind.spec.query


######## fn spec.variables.QueryVariableKind.spec.query.withDatasource

```jsonnet
spec.variables.QueryVariableKind.spec.query.withDatasource(value)
```

PARAMETERS:

* **value** (`object`)

New type for datasource reference
Not creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.
######## fn spec.variables.QueryVariableKind.spec.query.withDatasourceMixin

```jsonnet
spec.variables.QueryVariableKind.spec.query.withDatasourceMixin(value)
```

PARAMETERS:

* **value** (`object`)

New type for datasource reference
Not creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.
######## fn spec.variables.QueryVariableKind.spec.query.withGroup

```jsonnet
spec.variables.QueryVariableKind.spec.query.withGroup(value)
```

PARAMETERS:

* **value** (`string`)


######## fn spec.variables.QueryVariableKind.spec.query.withKind

```jsonnet
spec.variables.QueryVariableKind.spec.query.withKind()
```



######## fn spec.variables.QueryVariableKind.spec.query.withLabels

```jsonnet
spec.variables.QueryVariableKind.spec.query.withLabels(value)
```

PARAMETERS:

* **value** (`object`)


######## fn spec.variables.QueryVariableKind.spec.query.withLabelsMixin

```jsonnet
spec.variables.QueryVariableKind.spec.query.withLabelsMixin(value)
```

PARAMETERS:

* **value** (`object`)


######## fn spec.variables.QueryVariableKind.spec.query.withSpec

```jsonnet
spec.variables.QueryVariableKind.spec.query.withSpec(value)
```

PARAMETERS:

* **value** (`object`)


######## fn spec.variables.QueryVariableKind.spec.query.withSpecMixin

```jsonnet
spec.variables.QueryVariableKind.spec.query.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


######## fn spec.variables.QueryVariableKind.spec.query.withVersion

```jsonnet
spec.variables.QueryVariableKind.spec.query.withVersion(value="v0")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"v0"`


######## obj spec.variables.QueryVariableKind.spec.query.datasource


######### fn spec.variables.QueryVariableKind.spec.query.datasource.withName

```jsonnet
spec.variables.QueryVariableKind.spec.query.datasource.withName(value)
```

PARAMETERS:

* **value** (`string`)


##### obj spec.variables.SwitchVariableKind


###### fn spec.variables.SwitchVariableKind.withKind

```jsonnet
spec.variables.SwitchVariableKind.withKind()
```



###### fn spec.variables.SwitchVariableKind.withSpec

```jsonnet
spec.variables.SwitchVariableKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)


###### fn spec.variables.SwitchVariableKind.withSpecMixin

```jsonnet
spec.variables.SwitchVariableKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### obj spec.variables.SwitchVariableKind.spec


####### fn spec.variables.SwitchVariableKind.spec.withCurrent

```jsonnet
spec.variables.SwitchVariableKind.spec.withCurrent(value="false")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"false"`


####### fn spec.variables.SwitchVariableKind.spec.withDescription

```jsonnet
spec.variables.SwitchVariableKind.spec.withDescription(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.SwitchVariableKind.spec.withDisabledValue

```jsonnet
spec.variables.SwitchVariableKind.spec.withDisabledValue(value="false")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"false"`


####### fn spec.variables.SwitchVariableKind.spec.withEnabledValue

```jsonnet
spec.variables.SwitchVariableKind.spec.withEnabledValue(value="true")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"true"`


####### fn spec.variables.SwitchVariableKind.spec.withHide

```jsonnet
spec.variables.SwitchVariableKind.spec.withHide(value="dontHide")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"dontHide"`
   - valid values: `"dontHide"`, `"hideLabel"`, `"hideVariable"`, `"inControlsMenu"`

Determine if the variable shows on dashboard
Accepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).
####### fn spec.variables.SwitchVariableKind.spec.withLabel

```jsonnet
spec.variables.SwitchVariableKind.spec.withLabel(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.SwitchVariableKind.spec.withName

```jsonnet
spec.variables.SwitchVariableKind.spec.withName(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.SwitchVariableKind.spec.withOrigin

```jsonnet
spec.variables.SwitchVariableKind.spec.withOrigin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.SwitchVariableKind.spec.withOriginMixin

```jsonnet
spec.variables.SwitchVariableKind.spec.withOriginMixin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.SwitchVariableKind.spec.withSkipUrlSync

```jsonnet
spec.variables.SwitchVariableKind.spec.withSkipUrlSync(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### obj spec.variables.SwitchVariableKind.spec.origin


######## fn spec.variables.SwitchVariableKind.spec.origin.withGroup

```jsonnet
spec.variables.SwitchVariableKind.spec.origin.withGroup(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
######## fn spec.variables.SwitchVariableKind.spec.origin.withType

```jsonnet
spec.variables.SwitchVariableKind.spec.origin.withType()
```



##### obj spec.variables.TextVariableKind


###### fn spec.variables.TextVariableKind.withKind

```jsonnet
spec.variables.TextVariableKind.withKind()
```



###### fn spec.variables.TextVariableKind.withSpec

```jsonnet
spec.variables.TextVariableKind.withSpec(value)
```

PARAMETERS:

* **value** (`object`)

Text variable specification
###### fn spec.variables.TextVariableKind.withSpecMixin

```jsonnet
spec.variables.TextVariableKind.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)

Text variable specification
###### obj spec.variables.TextVariableKind.spec


####### fn spec.variables.TextVariableKind.spec.withCurrent

```jsonnet
spec.variables.TextVariableKind.spec.withCurrent(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.TextVariableKind.spec.withCurrentMixin

```jsonnet
spec.variables.TextVariableKind.spec.withCurrentMixin(value={"text": "","value": ""})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"text": "","value": ""}`

Variable option specification
####### fn spec.variables.TextVariableKind.spec.withDescription

```jsonnet
spec.variables.TextVariableKind.spec.withDescription(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.TextVariableKind.spec.withHide

```jsonnet
spec.variables.TextVariableKind.spec.withHide(value="dontHide")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"dontHide"`
   - valid values: `"dontHide"`, `"hideLabel"`, `"hideVariable"`, `"inControlsMenu"`

Determine if the variable shows on dashboard
Accepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).
####### fn spec.variables.TextVariableKind.spec.withLabel

```jsonnet
spec.variables.TextVariableKind.spec.withLabel(value)
```

PARAMETERS:

* **value** (`string`)


####### fn spec.variables.TextVariableKind.spec.withName

```jsonnet
spec.variables.TextVariableKind.spec.withName(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.TextVariableKind.spec.withOrigin

```jsonnet
spec.variables.TextVariableKind.spec.withOrigin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.TextVariableKind.spec.withOriginMixin

```jsonnet
spec.variables.TextVariableKind.spec.withOriginMixin(value)
```

PARAMETERS:

* **value** (`object`)

Source information for controls (e.g. variables or links)
####### fn spec.variables.TextVariableKind.spec.withQuery

```jsonnet
spec.variables.TextVariableKind.spec.withQuery(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


####### fn spec.variables.TextVariableKind.spec.withSkipUrlSync

```jsonnet
spec.variables.TextVariableKind.spec.withSkipUrlSync(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### obj spec.variables.TextVariableKind.spec.current


######## fn spec.variables.TextVariableKind.spec.current.withProperties

```jsonnet
spec.variables.TextVariableKind.spec.current.withProperties(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.TextVariableKind.spec.current.withPropertiesMixin

```jsonnet
spec.variables.TextVariableKind.spec.current.withPropertiesMixin(value)
```

PARAMETERS:

* **value** (`object`)

Additional properties for multi-props variables
######## fn spec.variables.TextVariableKind.spec.current.withSelected

```jsonnet
spec.variables.TextVariableKind.spec.current.withSelected(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether the option is selected or not
######## fn spec.variables.TextVariableKind.spec.current.withText

```jsonnet
spec.variables.TextVariableKind.spec.current.withText(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.TextVariableKind.spec.current.withTextMixin

```jsonnet
spec.variables.TextVariableKind.spec.current.withTextMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Text to be displayed for the option
######## fn spec.variables.TextVariableKind.spec.current.withValue

```jsonnet
spec.variables.TextVariableKind.spec.current.withValue(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
######## fn spec.variables.TextVariableKind.spec.current.withValueMixin

```jsonnet
spec.variables.TextVariableKind.spec.current.withValueMixin(value)
```

PARAMETERS:

* **value** (`array`,`string`)

Value of the option
####### obj spec.variables.TextVariableKind.spec.origin


######## fn spec.variables.TextVariableKind.spec.origin.withGroup

```jsonnet
spec.variables.TextVariableKind.spec.origin.withGroup(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
######## fn spec.variables.TextVariableKind.spec.origin.withType

```jsonnet
spec.variables.TextVariableKind.spec.origin.withType()
```


