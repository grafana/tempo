# metrics



## Subpackages

* [MetricAggregationWithSettings.BucketScript.pipelineVariables](MetricAggregationWithSettings/BucketScript/pipelineVariables.md)
* [PipelineMetricAggregation.BucketScript.pipelineVariables](PipelineMetricAggregation/BucketScript/pipelineVariables.md)

## Index

* [`obj Count`](#obj-count)
  * [`fn withHide(value=true)`](#fn-countwithhide)
  * [`fn withId(value)`](#fn-countwithid)
  * [`fn withType(value)`](#fn-countwithtype)
  * [`fn withTypeMixin(value)`](#fn-countwithtypemixin)
  * [`obj type`](#obj-counttype)
    * [`fn withPipelineMetricAggregationType(value)`](#fn-counttypewithpipelinemetricaggregationtype)
* [`obj MetricAggregationWithSettings`](#obj-metricaggregationwithsettings)
  * [`obj Average`](#obj-metricaggregationwithsettingsaverage)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingsaveragewithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsaveragewithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsaveragewithid)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsaveragewithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsaveragewithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsaveragewithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsaveragewithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsaveragesettings)
      * [`fn withMissing(value)`](#fn-metricaggregationwithsettingsaveragesettingswithmissing)
      * [`fn withScript(value)`](#fn-metricaggregationwithsettingsaveragesettingswithscript)
      * [`fn withScriptMixin(value)`](#fn-metricaggregationwithsettingsaveragesettingswithscriptmixin)
      * [`obj script`](#obj-metricaggregationwithsettingsaveragesettingsscript)
        * [`fn withInline(value)`](#fn-metricaggregationwithsettingsaveragesettingsscriptwithinline)
    * [`obj type`](#obj-metricaggregationwithsettingsaveragetype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsaveragetypewithpipelinemetricaggregationtype)
  * [`obj BucketScript`](#obj-metricaggregationwithsettingsbucketscript)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsbucketscriptwithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsbucketscriptwithid)
    * [`fn withPipelineVariables(value)`](#fn-metricaggregationwithsettingsbucketscriptwithpipelinevariables)
    * [`fn withPipelineVariablesMixin(value)`](#fn-metricaggregationwithsettingsbucketscriptwithpipelinevariablesmixin)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsbucketscriptwithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsbucketscriptwithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsbucketscriptwithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsbucketscriptwithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsbucketscriptsettings)
      * [`fn withScript(value)`](#fn-metricaggregationwithsettingsbucketscriptsettingswithscript)
      * [`fn withScriptMixin(value)`](#fn-metricaggregationwithsettingsbucketscriptsettingswithscriptmixin)
      * [`obj script`](#obj-metricaggregationwithsettingsbucketscriptsettingsscript)
        * [`fn withInline(value)`](#fn-metricaggregationwithsettingsbucketscriptsettingsscriptwithinline)
    * [`obj type`](#obj-metricaggregationwithsettingsbucketscripttype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsbucketscripttypewithpipelinemetricaggregationtype)
  * [`obj CumulativeSum`](#obj-metricaggregationwithsettingscumulativesum)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingscumulativesumwithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingscumulativesumwithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingscumulativesumwithid)
    * [`fn withPipelineAgg(value)`](#fn-metricaggregationwithsettingscumulativesumwithpipelineagg)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingscumulativesumwithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingscumulativesumwithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingscumulativesumwithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingscumulativesumwithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingscumulativesumsettings)
      * [`fn withFormat(value)`](#fn-metricaggregationwithsettingscumulativesumsettingswithformat)
    * [`obj type`](#obj-metricaggregationwithsettingscumulativesumtype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingscumulativesumtypewithpipelinemetricaggregationtype)
  * [`obj Derivative`](#obj-metricaggregationwithsettingsderivative)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingsderivativewithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsderivativewithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsderivativewithid)
    * [`fn withPipelineAgg(value)`](#fn-metricaggregationwithsettingsderivativewithpipelineagg)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsderivativewithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsderivativewithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsderivativewithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsderivativewithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsderivativesettings)
      * [`fn withUnit(value)`](#fn-metricaggregationwithsettingsderivativesettingswithunit)
    * [`obj type`](#obj-metricaggregationwithsettingsderivativetype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsderivativetypewithpipelinemetricaggregationtype)
  * [`obj ExtendedStats`](#obj-metricaggregationwithsettingsextendedstats)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingsextendedstatswithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsextendedstatswithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsextendedstatswithid)
    * [`fn withMeta(value)`](#fn-metricaggregationwithsettingsextendedstatswithmeta)
    * [`fn withMetaMixin(value)`](#fn-metricaggregationwithsettingsextendedstatswithmetamixin)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsextendedstatswithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsextendedstatswithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsextendedstatswithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsextendedstatswithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsextendedstatssettings)
      * [`fn withMissing(value)`](#fn-metricaggregationwithsettingsextendedstatssettingswithmissing)
      * [`fn withScript(value)`](#fn-metricaggregationwithsettingsextendedstatssettingswithscript)
      * [`fn withScriptMixin(value)`](#fn-metricaggregationwithsettingsextendedstatssettingswithscriptmixin)
      * [`fn withSigma(value)`](#fn-metricaggregationwithsettingsextendedstatssettingswithsigma)
      * [`obj script`](#obj-metricaggregationwithsettingsextendedstatssettingsscript)
        * [`fn withInline(value)`](#fn-metricaggregationwithsettingsextendedstatssettingsscriptwithinline)
    * [`obj type`](#obj-metricaggregationwithsettingsextendedstatstype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsextendedstatstypewithpipelinemetricaggregationtype)
  * [`obj Logs`](#obj-metricaggregationwithsettingslogs)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingslogswithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingslogswithid)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingslogswithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingslogswithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingslogswithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingslogswithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingslogssettings)
      * [`fn withLimit(value)`](#fn-metricaggregationwithsettingslogssettingswithlimit)
    * [`obj type`](#obj-metricaggregationwithsettingslogstype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingslogstypewithpipelinemetricaggregationtype)
  * [`obj Max`](#obj-metricaggregationwithsettingsmax)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingsmaxwithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsmaxwithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsmaxwithid)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsmaxwithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsmaxwithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsmaxwithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsmaxwithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsmaxsettings)
      * [`fn withMissing(value)`](#fn-metricaggregationwithsettingsmaxsettingswithmissing)
      * [`fn withScript(value)`](#fn-metricaggregationwithsettingsmaxsettingswithscript)
      * [`fn withScriptMixin(value)`](#fn-metricaggregationwithsettingsmaxsettingswithscriptmixin)
      * [`obj script`](#obj-metricaggregationwithsettingsmaxsettingsscript)
        * [`fn withInline(value)`](#fn-metricaggregationwithsettingsmaxsettingsscriptwithinline)
    * [`obj type`](#obj-metricaggregationwithsettingsmaxtype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsmaxtypewithpipelinemetricaggregationtype)
  * [`obj Min`](#obj-metricaggregationwithsettingsmin)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingsminwithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsminwithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsminwithid)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsminwithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsminwithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsminwithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsminwithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsminsettings)
      * [`fn withMissing(value)`](#fn-metricaggregationwithsettingsminsettingswithmissing)
      * [`fn withScript(value)`](#fn-metricaggregationwithsettingsminsettingswithscript)
      * [`fn withScriptMixin(value)`](#fn-metricaggregationwithsettingsminsettingswithscriptmixin)
      * [`obj script`](#obj-metricaggregationwithsettingsminsettingsscript)
        * [`fn withInline(value)`](#fn-metricaggregationwithsettingsminsettingsscriptwithinline)
    * [`obj type`](#obj-metricaggregationwithsettingsmintype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsmintypewithpipelinemetricaggregationtype)
  * [`obj MovingAverage`](#obj-metricaggregationwithsettingsmovingaverage)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingsmovingaveragewithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsmovingaveragewithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsmovingaveragewithid)
    * [`fn withPipelineAgg(value)`](#fn-metricaggregationwithsettingsmovingaveragewithpipelineagg)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsmovingaveragewithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsmovingaveragewithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsmovingaveragewithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsmovingaveragewithtypemixin)
    * [`obj type`](#obj-metricaggregationwithsettingsmovingaveragetype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsmovingaveragetypewithpipelinemetricaggregationtype)
  * [`obj MovingFunction`](#obj-metricaggregationwithsettingsmovingfunction)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingsmovingfunctionwithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsmovingfunctionwithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsmovingfunctionwithid)
    * [`fn withPipelineAgg(value)`](#fn-metricaggregationwithsettingsmovingfunctionwithpipelineagg)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsmovingfunctionwithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsmovingfunctionwithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsmovingfunctionwithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsmovingfunctionwithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsmovingfunctionsettings)
      * [`fn withScript(value)`](#fn-metricaggregationwithsettingsmovingfunctionsettingswithscript)
      * [`fn withScriptMixin(value)`](#fn-metricaggregationwithsettingsmovingfunctionsettingswithscriptmixin)
      * [`fn withShift(value)`](#fn-metricaggregationwithsettingsmovingfunctionsettingswithshift)
      * [`fn withWindow(value)`](#fn-metricaggregationwithsettingsmovingfunctionsettingswithwindow)
      * [`obj script`](#obj-metricaggregationwithsettingsmovingfunctionsettingsscript)
        * [`fn withInline(value)`](#fn-metricaggregationwithsettingsmovingfunctionsettingsscriptwithinline)
    * [`obj type`](#obj-metricaggregationwithsettingsmovingfunctiontype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsmovingfunctiontypewithpipelinemetricaggregationtype)
  * [`obj Percentiles`](#obj-metricaggregationwithsettingspercentiles)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingspercentileswithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingspercentileswithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingspercentileswithid)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingspercentileswithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingspercentileswithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingspercentileswithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingspercentileswithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingspercentilessettings)
      * [`fn withMissing(value)`](#fn-metricaggregationwithsettingspercentilessettingswithmissing)
      * [`fn withPercents(value)`](#fn-metricaggregationwithsettingspercentilessettingswithpercents)
      * [`fn withPercentsMixin(value)`](#fn-metricaggregationwithsettingspercentilessettingswithpercentsmixin)
      * [`fn withScript(value)`](#fn-metricaggregationwithsettingspercentilessettingswithscript)
      * [`fn withScriptMixin(value)`](#fn-metricaggregationwithsettingspercentilessettingswithscriptmixin)
      * [`obj script`](#obj-metricaggregationwithsettingspercentilessettingsscript)
        * [`fn withInline(value)`](#fn-metricaggregationwithsettingspercentilessettingsscriptwithinline)
    * [`obj type`](#obj-metricaggregationwithsettingspercentilestype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingspercentilestypewithpipelinemetricaggregationtype)
  * [`obj Rate`](#obj-metricaggregationwithsettingsrate)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingsratewithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsratewithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsratewithid)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsratewithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsratewithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsratewithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsratewithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsratesettings)
      * [`fn withMode(value)`](#fn-metricaggregationwithsettingsratesettingswithmode)
      * [`fn withUnit(value)`](#fn-metricaggregationwithsettingsratesettingswithunit)
    * [`obj type`](#obj-metricaggregationwithsettingsratetype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsratetypewithpipelinemetricaggregationtype)
  * [`obj RawData`](#obj-metricaggregationwithsettingsrawdata)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsrawdatawithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsrawdatawithid)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsrawdatawithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsrawdatawithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsrawdatawithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsrawdatawithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsrawdatasettings)
      * [`fn withSize(value)`](#fn-metricaggregationwithsettingsrawdatasettingswithsize)
    * [`obj type`](#obj-metricaggregationwithsettingsrawdatatype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsrawdatatypewithpipelinemetricaggregationtype)
  * [`obj RawDocument`](#obj-metricaggregationwithsettingsrawdocument)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsrawdocumentwithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsrawdocumentwithid)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsrawdocumentwithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsrawdocumentwithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsrawdocumentwithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsrawdocumentwithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsrawdocumentsettings)
      * [`fn withSize(value)`](#fn-metricaggregationwithsettingsrawdocumentsettingswithsize)
    * [`obj type`](#obj-metricaggregationwithsettingsrawdocumenttype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsrawdocumenttypewithpipelinemetricaggregationtype)
  * [`obj SerialDiff`](#obj-metricaggregationwithsettingsserialdiff)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingsserialdiffwithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsserialdiffwithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsserialdiffwithid)
    * [`fn withPipelineAgg(value)`](#fn-metricaggregationwithsettingsserialdiffwithpipelineagg)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsserialdiffwithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsserialdiffwithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsserialdiffwithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsserialdiffwithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsserialdiffsettings)
      * [`fn withLag(value)`](#fn-metricaggregationwithsettingsserialdiffsettingswithlag)
    * [`obj type`](#obj-metricaggregationwithsettingsserialdifftype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsserialdifftypewithpipelinemetricaggregationtype)
  * [`obj Sum`](#obj-metricaggregationwithsettingssum)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingssumwithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingssumwithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingssumwithid)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingssumwithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingssumwithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingssumwithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingssumwithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingssumsettings)
      * [`fn withMissing(value)`](#fn-metricaggregationwithsettingssumsettingswithmissing)
      * [`fn withScript(value)`](#fn-metricaggregationwithsettingssumsettingswithscript)
      * [`fn withScriptMixin(value)`](#fn-metricaggregationwithsettingssumsettingswithscriptmixin)
      * [`obj script`](#obj-metricaggregationwithsettingssumsettingsscript)
        * [`fn withInline(value)`](#fn-metricaggregationwithsettingssumsettingsscriptwithinline)
    * [`obj type`](#obj-metricaggregationwithsettingssumtype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingssumtypewithpipelinemetricaggregationtype)
  * [`obj TopMetrics`](#obj-metricaggregationwithsettingstopmetrics)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingstopmetricswithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingstopmetricswithid)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingstopmetricswithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingstopmetricswithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingstopmetricswithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingstopmetricswithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingstopmetricssettings)
      * [`fn withMetrics(value)`](#fn-metricaggregationwithsettingstopmetricssettingswithmetrics)
      * [`fn withMetricsMixin(value)`](#fn-metricaggregationwithsettingstopmetricssettingswithmetricsmixin)
      * [`fn withOrder(value)`](#fn-metricaggregationwithsettingstopmetricssettingswithorder)
      * [`fn withOrderBy(value)`](#fn-metricaggregationwithsettingstopmetricssettingswithorderby)
    * [`obj type`](#obj-metricaggregationwithsettingstopmetricstype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingstopmetricstypewithpipelinemetricaggregationtype)
  * [`obj UniqueCount`](#obj-metricaggregationwithsettingsuniquecount)
    * [`fn withField(value)`](#fn-metricaggregationwithsettingsuniquecountwithfield)
    * [`fn withHide(value=true)`](#fn-metricaggregationwithsettingsuniquecountwithhide)
    * [`fn withId(value)`](#fn-metricaggregationwithsettingsuniquecountwithid)
    * [`fn withSettings(value)`](#fn-metricaggregationwithsettingsuniquecountwithsettings)
    * [`fn withSettingsMixin(value)`](#fn-metricaggregationwithsettingsuniquecountwithsettingsmixin)
    * [`fn withType(value)`](#fn-metricaggregationwithsettingsuniquecountwithtype)
    * [`fn withTypeMixin(value)`](#fn-metricaggregationwithsettingsuniquecountwithtypemixin)
    * [`obj settings`](#obj-metricaggregationwithsettingsuniquecountsettings)
      * [`fn withMissing(value)`](#fn-metricaggregationwithsettingsuniquecountsettingswithmissing)
      * [`fn withPrecisionThreshold(value)`](#fn-metricaggregationwithsettingsuniquecountsettingswithprecisionthreshold)
    * [`obj type`](#obj-metricaggregationwithsettingsuniquecounttype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-metricaggregationwithsettingsuniquecounttypewithpipelinemetricaggregationtype)
* [`obj PipelineMetricAggregation`](#obj-pipelinemetricaggregation)
  * [`obj BucketScript`](#obj-pipelinemetricaggregationbucketscript)
    * [`fn withHide(value=true)`](#fn-pipelinemetricaggregationbucketscriptwithhide)
    * [`fn withId(value)`](#fn-pipelinemetricaggregationbucketscriptwithid)
    * [`fn withPipelineVariables(value)`](#fn-pipelinemetricaggregationbucketscriptwithpipelinevariables)
    * [`fn withPipelineVariablesMixin(value)`](#fn-pipelinemetricaggregationbucketscriptwithpipelinevariablesmixin)
    * [`fn withSettings(value)`](#fn-pipelinemetricaggregationbucketscriptwithsettings)
    * [`fn withSettingsMixin(value)`](#fn-pipelinemetricaggregationbucketscriptwithsettingsmixin)
    * [`fn withType(value)`](#fn-pipelinemetricaggregationbucketscriptwithtype)
    * [`fn withTypeMixin(value)`](#fn-pipelinemetricaggregationbucketscriptwithtypemixin)
    * [`obj settings`](#obj-pipelinemetricaggregationbucketscriptsettings)
      * [`fn withScript(value)`](#fn-pipelinemetricaggregationbucketscriptsettingswithscript)
      * [`fn withScriptMixin(value)`](#fn-pipelinemetricaggregationbucketscriptsettingswithscriptmixin)
      * [`obj script`](#obj-pipelinemetricaggregationbucketscriptsettingsscript)
        * [`fn withInline(value)`](#fn-pipelinemetricaggregationbucketscriptsettingsscriptwithinline)
    * [`obj type`](#obj-pipelinemetricaggregationbucketscripttype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-pipelinemetricaggregationbucketscripttypewithpipelinemetricaggregationtype)
  * [`obj CumulativeSum`](#obj-pipelinemetricaggregationcumulativesum)
    * [`fn withField(value)`](#fn-pipelinemetricaggregationcumulativesumwithfield)
    * [`fn withHide(value=true)`](#fn-pipelinemetricaggregationcumulativesumwithhide)
    * [`fn withId(value)`](#fn-pipelinemetricaggregationcumulativesumwithid)
    * [`fn withPipelineAgg(value)`](#fn-pipelinemetricaggregationcumulativesumwithpipelineagg)
    * [`fn withSettings(value)`](#fn-pipelinemetricaggregationcumulativesumwithsettings)
    * [`fn withSettingsMixin(value)`](#fn-pipelinemetricaggregationcumulativesumwithsettingsmixin)
    * [`fn withType(value)`](#fn-pipelinemetricaggregationcumulativesumwithtype)
    * [`fn withTypeMixin(value)`](#fn-pipelinemetricaggregationcumulativesumwithtypemixin)
    * [`obj settings`](#obj-pipelinemetricaggregationcumulativesumsettings)
      * [`fn withFormat(value)`](#fn-pipelinemetricaggregationcumulativesumsettingswithformat)
    * [`obj type`](#obj-pipelinemetricaggregationcumulativesumtype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-pipelinemetricaggregationcumulativesumtypewithpipelinemetricaggregationtype)
  * [`obj Derivative`](#obj-pipelinemetricaggregationderivative)
    * [`fn withField(value)`](#fn-pipelinemetricaggregationderivativewithfield)
    * [`fn withHide(value=true)`](#fn-pipelinemetricaggregationderivativewithhide)
    * [`fn withId(value)`](#fn-pipelinemetricaggregationderivativewithid)
    * [`fn withPipelineAgg(value)`](#fn-pipelinemetricaggregationderivativewithpipelineagg)
    * [`fn withSettings(value)`](#fn-pipelinemetricaggregationderivativewithsettings)
    * [`fn withSettingsMixin(value)`](#fn-pipelinemetricaggregationderivativewithsettingsmixin)
    * [`fn withType(value)`](#fn-pipelinemetricaggregationderivativewithtype)
    * [`fn withTypeMixin(value)`](#fn-pipelinemetricaggregationderivativewithtypemixin)
    * [`obj settings`](#obj-pipelinemetricaggregationderivativesettings)
      * [`fn withUnit(value)`](#fn-pipelinemetricaggregationderivativesettingswithunit)
    * [`obj type`](#obj-pipelinemetricaggregationderivativetype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-pipelinemetricaggregationderivativetypewithpipelinemetricaggregationtype)
  * [`obj MovingAverage`](#obj-pipelinemetricaggregationmovingaverage)
    * [`fn withField(value)`](#fn-pipelinemetricaggregationmovingaveragewithfield)
    * [`fn withHide(value=true)`](#fn-pipelinemetricaggregationmovingaveragewithhide)
    * [`fn withId(value)`](#fn-pipelinemetricaggregationmovingaveragewithid)
    * [`fn withPipelineAgg(value)`](#fn-pipelinemetricaggregationmovingaveragewithpipelineagg)
    * [`fn withSettings(value)`](#fn-pipelinemetricaggregationmovingaveragewithsettings)
    * [`fn withSettingsMixin(value)`](#fn-pipelinemetricaggregationmovingaveragewithsettingsmixin)
    * [`fn withType(value)`](#fn-pipelinemetricaggregationmovingaveragewithtype)
    * [`fn withTypeMixin(value)`](#fn-pipelinemetricaggregationmovingaveragewithtypemixin)
    * [`obj type`](#obj-pipelinemetricaggregationmovingaveragetype)
      * [`fn withPipelineMetricAggregationType(value)`](#fn-pipelinemetricaggregationmovingaveragetypewithpipelinemetricaggregationtype)

## Fields

### obj Count


#### fn Count.withHide

```jsonnet
Count.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn Count.withId

```jsonnet
Count.withId(value)
```

PARAMETERS:

* **value** (`string`)


#### fn Count.withType

```jsonnet
Count.withType(value)
```

PARAMETERS:

* **value** (`string`)


#### fn Count.withTypeMixin

```jsonnet
Count.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


#### obj Count.type


##### fn Count.type.withPipelineMetricAggregationType

```jsonnet
Count.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


### obj MetricAggregationWithSettings


#### obj MetricAggregationWithSettings.Average


##### fn MetricAggregationWithSettings.Average.withField

```jsonnet
MetricAggregationWithSettings.Average.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Average.withHide

```jsonnet
MetricAggregationWithSettings.Average.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.Average.withId

```jsonnet
MetricAggregationWithSettings.Average.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Average.withSettings

```jsonnet
MetricAggregationWithSettings.Average.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Average.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.Average.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Average.withType

```jsonnet
MetricAggregationWithSettings.Average.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Average.withTypeMixin

```jsonnet
MetricAggregationWithSettings.Average.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Average.settings


###### fn MetricAggregationWithSettings.Average.settings.withMissing

```jsonnet
MetricAggregationWithSettings.Average.settings.withMissing(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricAggregationWithSettings.Average.settings.withScript

```jsonnet
MetricAggregationWithSettings.Average.settings.withScript(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### fn MetricAggregationWithSettings.Average.settings.withScriptMixin

```jsonnet
MetricAggregationWithSettings.Average.settings.withScriptMixin(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### obj MetricAggregationWithSettings.Average.settings.script


####### fn MetricAggregationWithSettings.Average.settings.script.withInline

```jsonnet
MetricAggregationWithSettings.Average.settings.script.withInline(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Average.type


###### fn MetricAggregationWithSettings.Average.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.Average.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.BucketScript


##### fn MetricAggregationWithSettings.BucketScript.withHide

```jsonnet
MetricAggregationWithSettings.BucketScript.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.BucketScript.withId

```jsonnet
MetricAggregationWithSettings.BucketScript.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.BucketScript.withPipelineVariables

```jsonnet
MetricAggregationWithSettings.BucketScript.withPipelineVariables(value)
```

PARAMETERS:

* **value** (`array`)


##### fn MetricAggregationWithSettings.BucketScript.withPipelineVariablesMixin

```jsonnet
MetricAggregationWithSettings.BucketScript.withPipelineVariablesMixin(value)
```

PARAMETERS:

* **value** (`array`)


##### fn MetricAggregationWithSettings.BucketScript.withSettings

```jsonnet
MetricAggregationWithSettings.BucketScript.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.BucketScript.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.BucketScript.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.BucketScript.withType

```jsonnet
MetricAggregationWithSettings.BucketScript.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.BucketScript.withTypeMixin

```jsonnet
MetricAggregationWithSettings.BucketScript.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.BucketScript.settings


###### fn MetricAggregationWithSettings.BucketScript.settings.withScript

```jsonnet
MetricAggregationWithSettings.BucketScript.settings.withScript(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### fn MetricAggregationWithSettings.BucketScript.settings.withScriptMixin

```jsonnet
MetricAggregationWithSettings.BucketScript.settings.withScriptMixin(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### obj MetricAggregationWithSettings.BucketScript.settings.script


####### fn MetricAggregationWithSettings.BucketScript.settings.script.withInline

```jsonnet
MetricAggregationWithSettings.BucketScript.settings.script.withInline(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.BucketScript.type


###### fn MetricAggregationWithSettings.BucketScript.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.BucketScript.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.CumulativeSum


##### fn MetricAggregationWithSettings.CumulativeSum.withField

```jsonnet
MetricAggregationWithSettings.CumulativeSum.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.CumulativeSum.withHide

```jsonnet
MetricAggregationWithSettings.CumulativeSum.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.CumulativeSum.withId

```jsonnet
MetricAggregationWithSettings.CumulativeSum.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.CumulativeSum.withPipelineAgg

```jsonnet
MetricAggregationWithSettings.CumulativeSum.withPipelineAgg(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.CumulativeSum.withSettings

```jsonnet
MetricAggregationWithSettings.CumulativeSum.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.CumulativeSum.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.CumulativeSum.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.CumulativeSum.withType

```jsonnet
MetricAggregationWithSettings.CumulativeSum.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.CumulativeSum.withTypeMixin

```jsonnet
MetricAggregationWithSettings.CumulativeSum.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.CumulativeSum.settings


###### fn MetricAggregationWithSettings.CumulativeSum.settings.withFormat

```jsonnet
MetricAggregationWithSettings.CumulativeSum.settings.withFormat(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.CumulativeSum.type


###### fn MetricAggregationWithSettings.CumulativeSum.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.CumulativeSum.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.Derivative


##### fn MetricAggregationWithSettings.Derivative.withField

```jsonnet
MetricAggregationWithSettings.Derivative.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Derivative.withHide

```jsonnet
MetricAggregationWithSettings.Derivative.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.Derivative.withId

```jsonnet
MetricAggregationWithSettings.Derivative.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Derivative.withPipelineAgg

```jsonnet
MetricAggregationWithSettings.Derivative.withPipelineAgg(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Derivative.withSettings

```jsonnet
MetricAggregationWithSettings.Derivative.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Derivative.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.Derivative.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Derivative.withType

```jsonnet
MetricAggregationWithSettings.Derivative.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Derivative.withTypeMixin

```jsonnet
MetricAggregationWithSettings.Derivative.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Derivative.settings


###### fn MetricAggregationWithSettings.Derivative.settings.withUnit

```jsonnet
MetricAggregationWithSettings.Derivative.settings.withUnit(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Derivative.type


###### fn MetricAggregationWithSettings.Derivative.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.Derivative.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.ExtendedStats


##### fn MetricAggregationWithSettings.ExtendedStats.withField

```jsonnet
MetricAggregationWithSettings.ExtendedStats.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.ExtendedStats.withHide

```jsonnet
MetricAggregationWithSettings.ExtendedStats.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.ExtendedStats.withId

```jsonnet
MetricAggregationWithSettings.ExtendedStats.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.ExtendedStats.withMeta

```jsonnet
MetricAggregationWithSettings.ExtendedStats.withMeta(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.ExtendedStats.withMetaMixin

```jsonnet
MetricAggregationWithSettings.ExtendedStats.withMetaMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.ExtendedStats.withSettings

```jsonnet
MetricAggregationWithSettings.ExtendedStats.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.ExtendedStats.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.ExtendedStats.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.ExtendedStats.withType

```jsonnet
MetricAggregationWithSettings.ExtendedStats.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.ExtendedStats.withTypeMixin

```jsonnet
MetricAggregationWithSettings.ExtendedStats.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.ExtendedStats.settings


###### fn MetricAggregationWithSettings.ExtendedStats.settings.withMissing

```jsonnet
MetricAggregationWithSettings.ExtendedStats.settings.withMissing(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricAggregationWithSettings.ExtendedStats.settings.withScript

```jsonnet
MetricAggregationWithSettings.ExtendedStats.settings.withScript(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### fn MetricAggregationWithSettings.ExtendedStats.settings.withScriptMixin

```jsonnet
MetricAggregationWithSettings.ExtendedStats.settings.withScriptMixin(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### fn MetricAggregationWithSettings.ExtendedStats.settings.withSigma

```jsonnet
MetricAggregationWithSettings.ExtendedStats.settings.withSigma(value)
```

PARAMETERS:

* **value** (`string`)


###### obj MetricAggregationWithSettings.ExtendedStats.settings.script


####### fn MetricAggregationWithSettings.ExtendedStats.settings.script.withInline

```jsonnet
MetricAggregationWithSettings.ExtendedStats.settings.script.withInline(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.ExtendedStats.type


###### fn MetricAggregationWithSettings.ExtendedStats.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.ExtendedStats.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.Logs


##### fn MetricAggregationWithSettings.Logs.withHide

```jsonnet
MetricAggregationWithSettings.Logs.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.Logs.withId

```jsonnet
MetricAggregationWithSettings.Logs.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Logs.withSettings

```jsonnet
MetricAggregationWithSettings.Logs.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Logs.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.Logs.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Logs.withType

```jsonnet
MetricAggregationWithSettings.Logs.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Logs.withTypeMixin

```jsonnet
MetricAggregationWithSettings.Logs.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Logs.settings


###### fn MetricAggregationWithSettings.Logs.settings.withLimit

```jsonnet
MetricAggregationWithSettings.Logs.settings.withLimit(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Logs.type


###### fn MetricAggregationWithSettings.Logs.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.Logs.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.Max


##### fn MetricAggregationWithSettings.Max.withField

```jsonnet
MetricAggregationWithSettings.Max.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Max.withHide

```jsonnet
MetricAggregationWithSettings.Max.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.Max.withId

```jsonnet
MetricAggregationWithSettings.Max.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Max.withSettings

```jsonnet
MetricAggregationWithSettings.Max.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Max.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.Max.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Max.withType

```jsonnet
MetricAggregationWithSettings.Max.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Max.withTypeMixin

```jsonnet
MetricAggregationWithSettings.Max.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Max.settings


###### fn MetricAggregationWithSettings.Max.settings.withMissing

```jsonnet
MetricAggregationWithSettings.Max.settings.withMissing(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricAggregationWithSettings.Max.settings.withScript

```jsonnet
MetricAggregationWithSettings.Max.settings.withScript(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### fn MetricAggregationWithSettings.Max.settings.withScriptMixin

```jsonnet
MetricAggregationWithSettings.Max.settings.withScriptMixin(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### obj MetricAggregationWithSettings.Max.settings.script


####### fn MetricAggregationWithSettings.Max.settings.script.withInline

```jsonnet
MetricAggregationWithSettings.Max.settings.script.withInline(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Max.type


###### fn MetricAggregationWithSettings.Max.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.Max.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.Min


##### fn MetricAggregationWithSettings.Min.withField

```jsonnet
MetricAggregationWithSettings.Min.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Min.withHide

```jsonnet
MetricAggregationWithSettings.Min.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.Min.withId

```jsonnet
MetricAggregationWithSettings.Min.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Min.withSettings

```jsonnet
MetricAggregationWithSettings.Min.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Min.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.Min.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Min.withType

```jsonnet
MetricAggregationWithSettings.Min.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Min.withTypeMixin

```jsonnet
MetricAggregationWithSettings.Min.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Min.settings


###### fn MetricAggregationWithSettings.Min.settings.withMissing

```jsonnet
MetricAggregationWithSettings.Min.settings.withMissing(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricAggregationWithSettings.Min.settings.withScript

```jsonnet
MetricAggregationWithSettings.Min.settings.withScript(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### fn MetricAggregationWithSettings.Min.settings.withScriptMixin

```jsonnet
MetricAggregationWithSettings.Min.settings.withScriptMixin(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### obj MetricAggregationWithSettings.Min.settings.script


####### fn MetricAggregationWithSettings.Min.settings.script.withInline

```jsonnet
MetricAggregationWithSettings.Min.settings.script.withInline(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Min.type


###### fn MetricAggregationWithSettings.Min.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.Min.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.MovingAverage


##### fn MetricAggregationWithSettings.MovingAverage.withField

```jsonnet
MetricAggregationWithSettings.MovingAverage.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.MovingAverage.withHide

```jsonnet
MetricAggregationWithSettings.MovingAverage.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.MovingAverage.withId

```jsonnet
MetricAggregationWithSettings.MovingAverage.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.MovingAverage.withPipelineAgg

```jsonnet
MetricAggregationWithSettings.MovingAverage.withPipelineAgg(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.MovingAverage.withSettings

```jsonnet
MetricAggregationWithSettings.MovingAverage.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.MovingAverage.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.MovingAverage.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.MovingAverage.withType

```jsonnet
MetricAggregationWithSettings.MovingAverage.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.MovingAverage.withTypeMixin

```jsonnet
MetricAggregationWithSettings.MovingAverage.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.MovingAverage.type


###### fn MetricAggregationWithSettings.MovingAverage.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.MovingAverage.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.MovingFunction


##### fn MetricAggregationWithSettings.MovingFunction.withField

```jsonnet
MetricAggregationWithSettings.MovingFunction.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.MovingFunction.withHide

```jsonnet
MetricAggregationWithSettings.MovingFunction.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.MovingFunction.withId

```jsonnet
MetricAggregationWithSettings.MovingFunction.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.MovingFunction.withPipelineAgg

```jsonnet
MetricAggregationWithSettings.MovingFunction.withPipelineAgg(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.MovingFunction.withSettings

```jsonnet
MetricAggregationWithSettings.MovingFunction.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.MovingFunction.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.MovingFunction.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.MovingFunction.withType

```jsonnet
MetricAggregationWithSettings.MovingFunction.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.MovingFunction.withTypeMixin

```jsonnet
MetricAggregationWithSettings.MovingFunction.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.MovingFunction.settings


###### fn MetricAggregationWithSettings.MovingFunction.settings.withScript

```jsonnet
MetricAggregationWithSettings.MovingFunction.settings.withScript(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### fn MetricAggregationWithSettings.MovingFunction.settings.withScriptMixin

```jsonnet
MetricAggregationWithSettings.MovingFunction.settings.withScriptMixin(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### fn MetricAggregationWithSettings.MovingFunction.settings.withShift

```jsonnet
MetricAggregationWithSettings.MovingFunction.settings.withShift(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricAggregationWithSettings.MovingFunction.settings.withWindow

```jsonnet
MetricAggregationWithSettings.MovingFunction.settings.withWindow(value)
```

PARAMETERS:

* **value** (`string`)


###### obj MetricAggregationWithSettings.MovingFunction.settings.script


####### fn MetricAggregationWithSettings.MovingFunction.settings.script.withInline

```jsonnet
MetricAggregationWithSettings.MovingFunction.settings.script.withInline(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.MovingFunction.type


###### fn MetricAggregationWithSettings.MovingFunction.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.MovingFunction.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.Percentiles


##### fn MetricAggregationWithSettings.Percentiles.withField

```jsonnet
MetricAggregationWithSettings.Percentiles.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Percentiles.withHide

```jsonnet
MetricAggregationWithSettings.Percentiles.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.Percentiles.withId

```jsonnet
MetricAggregationWithSettings.Percentiles.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Percentiles.withSettings

```jsonnet
MetricAggregationWithSettings.Percentiles.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Percentiles.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.Percentiles.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Percentiles.withType

```jsonnet
MetricAggregationWithSettings.Percentiles.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Percentiles.withTypeMixin

```jsonnet
MetricAggregationWithSettings.Percentiles.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Percentiles.settings


###### fn MetricAggregationWithSettings.Percentiles.settings.withMissing

```jsonnet
MetricAggregationWithSettings.Percentiles.settings.withMissing(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricAggregationWithSettings.Percentiles.settings.withPercents

```jsonnet
MetricAggregationWithSettings.Percentiles.settings.withPercents(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricAggregationWithSettings.Percentiles.settings.withPercentsMixin

```jsonnet
MetricAggregationWithSettings.Percentiles.settings.withPercentsMixin(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricAggregationWithSettings.Percentiles.settings.withScript

```jsonnet
MetricAggregationWithSettings.Percentiles.settings.withScript(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### fn MetricAggregationWithSettings.Percentiles.settings.withScriptMixin

```jsonnet
MetricAggregationWithSettings.Percentiles.settings.withScriptMixin(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### obj MetricAggregationWithSettings.Percentiles.settings.script


####### fn MetricAggregationWithSettings.Percentiles.settings.script.withInline

```jsonnet
MetricAggregationWithSettings.Percentiles.settings.script.withInline(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Percentiles.type


###### fn MetricAggregationWithSettings.Percentiles.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.Percentiles.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.Rate


##### fn MetricAggregationWithSettings.Rate.withField

```jsonnet
MetricAggregationWithSettings.Rate.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Rate.withHide

```jsonnet
MetricAggregationWithSettings.Rate.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.Rate.withId

```jsonnet
MetricAggregationWithSettings.Rate.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Rate.withSettings

```jsonnet
MetricAggregationWithSettings.Rate.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Rate.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.Rate.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Rate.withType

```jsonnet
MetricAggregationWithSettings.Rate.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Rate.withTypeMixin

```jsonnet
MetricAggregationWithSettings.Rate.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Rate.settings


###### fn MetricAggregationWithSettings.Rate.settings.withMode

```jsonnet
MetricAggregationWithSettings.Rate.settings.withMode(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricAggregationWithSettings.Rate.settings.withUnit

```jsonnet
MetricAggregationWithSettings.Rate.settings.withUnit(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Rate.type


###### fn MetricAggregationWithSettings.Rate.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.Rate.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.RawData


##### fn MetricAggregationWithSettings.RawData.withHide

```jsonnet
MetricAggregationWithSettings.RawData.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.RawData.withId

```jsonnet
MetricAggregationWithSettings.RawData.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.RawData.withSettings

```jsonnet
MetricAggregationWithSettings.RawData.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.RawData.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.RawData.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.RawData.withType

```jsonnet
MetricAggregationWithSettings.RawData.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.RawData.withTypeMixin

```jsonnet
MetricAggregationWithSettings.RawData.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.RawData.settings


###### fn MetricAggregationWithSettings.RawData.settings.withSize

```jsonnet
MetricAggregationWithSettings.RawData.settings.withSize(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.RawData.type


###### fn MetricAggregationWithSettings.RawData.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.RawData.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.RawDocument


##### fn MetricAggregationWithSettings.RawDocument.withHide

```jsonnet
MetricAggregationWithSettings.RawDocument.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.RawDocument.withId

```jsonnet
MetricAggregationWithSettings.RawDocument.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.RawDocument.withSettings

```jsonnet
MetricAggregationWithSettings.RawDocument.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.RawDocument.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.RawDocument.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.RawDocument.withType

```jsonnet
MetricAggregationWithSettings.RawDocument.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.RawDocument.withTypeMixin

```jsonnet
MetricAggregationWithSettings.RawDocument.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.RawDocument.settings


###### fn MetricAggregationWithSettings.RawDocument.settings.withSize

```jsonnet
MetricAggregationWithSettings.RawDocument.settings.withSize(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.RawDocument.type


###### fn MetricAggregationWithSettings.RawDocument.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.RawDocument.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.SerialDiff


##### fn MetricAggregationWithSettings.SerialDiff.withField

```jsonnet
MetricAggregationWithSettings.SerialDiff.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.SerialDiff.withHide

```jsonnet
MetricAggregationWithSettings.SerialDiff.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.SerialDiff.withId

```jsonnet
MetricAggregationWithSettings.SerialDiff.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.SerialDiff.withPipelineAgg

```jsonnet
MetricAggregationWithSettings.SerialDiff.withPipelineAgg(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.SerialDiff.withSettings

```jsonnet
MetricAggregationWithSettings.SerialDiff.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.SerialDiff.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.SerialDiff.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.SerialDiff.withType

```jsonnet
MetricAggregationWithSettings.SerialDiff.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.SerialDiff.withTypeMixin

```jsonnet
MetricAggregationWithSettings.SerialDiff.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.SerialDiff.settings


###### fn MetricAggregationWithSettings.SerialDiff.settings.withLag

```jsonnet
MetricAggregationWithSettings.SerialDiff.settings.withLag(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.SerialDiff.type


###### fn MetricAggregationWithSettings.SerialDiff.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.SerialDiff.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.Sum


##### fn MetricAggregationWithSettings.Sum.withField

```jsonnet
MetricAggregationWithSettings.Sum.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Sum.withHide

```jsonnet
MetricAggregationWithSettings.Sum.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.Sum.withId

```jsonnet
MetricAggregationWithSettings.Sum.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Sum.withSettings

```jsonnet
MetricAggregationWithSettings.Sum.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Sum.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.Sum.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.Sum.withType

```jsonnet
MetricAggregationWithSettings.Sum.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.Sum.withTypeMixin

```jsonnet
MetricAggregationWithSettings.Sum.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Sum.settings


###### fn MetricAggregationWithSettings.Sum.settings.withMissing

```jsonnet
MetricAggregationWithSettings.Sum.settings.withMissing(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricAggregationWithSettings.Sum.settings.withScript

```jsonnet
MetricAggregationWithSettings.Sum.settings.withScript(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### fn MetricAggregationWithSettings.Sum.settings.withScriptMixin

```jsonnet
MetricAggregationWithSettings.Sum.settings.withScriptMixin(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### obj MetricAggregationWithSettings.Sum.settings.script


####### fn MetricAggregationWithSettings.Sum.settings.script.withInline

```jsonnet
MetricAggregationWithSettings.Sum.settings.script.withInline(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.Sum.type


###### fn MetricAggregationWithSettings.Sum.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.Sum.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.TopMetrics


##### fn MetricAggregationWithSettings.TopMetrics.withHide

```jsonnet
MetricAggregationWithSettings.TopMetrics.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.TopMetrics.withId

```jsonnet
MetricAggregationWithSettings.TopMetrics.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.TopMetrics.withSettings

```jsonnet
MetricAggregationWithSettings.TopMetrics.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.TopMetrics.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.TopMetrics.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.TopMetrics.withType

```jsonnet
MetricAggregationWithSettings.TopMetrics.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.TopMetrics.withTypeMixin

```jsonnet
MetricAggregationWithSettings.TopMetrics.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.TopMetrics.settings


###### fn MetricAggregationWithSettings.TopMetrics.settings.withMetrics

```jsonnet
MetricAggregationWithSettings.TopMetrics.settings.withMetrics(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricAggregationWithSettings.TopMetrics.settings.withMetricsMixin

```jsonnet
MetricAggregationWithSettings.TopMetrics.settings.withMetricsMixin(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricAggregationWithSettings.TopMetrics.settings.withOrder

```jsonnet
MetricAggregationWithSettings.TopMetrics.settings.withOrder(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricAggregationWithSettings.TopMetrics.settings.withOrderBy

```jsonnet
MetricAggregationWithSettings.TopMetrics.settings.withOrderBy(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.TopMetrics.type


###### fn MetricAggregationWithSettings.TopMetrics.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.TopMetrics.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj MetricAggregationWithSettings.UniqueCount


##### fn MetricAggregationWithSettings.UniqueCount.withField

```jsonnet
MetricAggregationWithSettings.UniqueCount.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.UniqueCount.withHide

```jsonnet
MetricAggregationWithSettings.UniqueCount.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn MetricAggregationWithSettings.UniqueCount.withId

```jsonnet
MetricAggregationWithSettings.UniqueCount.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.UniqueCount.withSettings

```jsonnet
MetricAggregationWithSettings.UniqueCount.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.UniqueCount.withSettingsMixin

```jsonnet
MetricAggregationWithSettings.UniqueCount.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn MetricAggregationWithSettings.UniqueCount.withType

```jsonnet
MetricAggregationWithSettings.UniqueCount.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn MetricAggregationWithSettings.UniqueCount.withTypeMixin

```jsonnet
MetricAggregationWithSettings.UniqueCount.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.UniqueCount.settings


###### fn MetricAggregationWithSettings.UniqueCount.settings.withMissing

```jsonnet
MetricAggregationWithSettings.UniqueCount.settings.withMissing(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricAggregationWithSettings.UniqueCount.settings.withPrecisionThreshold

```jsonnet
MetricAggregationWithSettings.UniqueCount.settings.withPrecisionThreshold(value)
```

PARAMETERS:

* **value** (`string`)


##### obj MetricAggregationWithSettings.UniqueCount.type


###### fn MetricAggregationWithSettings.UniqueCount.type.withPipelineMetricAggregationType

```jsonnet
MetricAggregationWithSettings.UniqueCount.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


### obj PipelineMetricAggregation


#### obj PipelineMetricAggregation.BucketScript


##### fn PipelineMetricAggregation.BucketScript.withHide

```jsonnet
PipelineMetricAggregation.BucketScript.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn PipelineMetricAggregation.BucketScript.withId

```jsonnet
PipelineMetricAggregation.BucketScript.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.BucketScript.withPipelineVariables

```jsonnet
PipelineMetricAggregation.BucketScript.withPipelineVariables(value)
```

PARAMETERS:

* **value** (`array`)


##### fn PipelineMetricAggregation.BucketScript.withPipelineVariablesMixin

```jsonnet
PipelineMetricAggregation.BucketScript.withPipelineVariablesMixin(value)
```

PARAMETERS:

* **value** (`array`)


##### fn PipelineMetricAggregation.BucketScript.withSettings

```jsonnet
PipelineMetricAggregation.BucketScript.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn PipelineMetricAggregation.BucketScript.withSettingsMixin

```jsonnet
PipelineMetricAggregation.BucketScript.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn PipelineMetricAggregation.BucketScript.withType

```jsonnet
PipelineMetricAggregation.BucketScript.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.BucketScript.withTypeMixin

```jsonnet
PipelineMetricAggregation.BucketScript.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj PipelineMetricAggregation.BucketScript.settings


###### fn PipelineMetricAggregation.BucketScript.settings.withScript

```jsonnet
PipelineMetricAggregation.BucketScript.settings.withScript(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### fn PipelineMetricAggregation.BucketScript.settings.withScriptMixin

```jsonnet
PipelineMetricAggregation.BucketScript.settings.withScriptMixin(value)
```

PARAMETERS:

* **value** (`object`,`string`)


###### obj PipelineMetricAggregation.BucketScript.settings.script


####### fn PipelineMetricAggregation.BucketScript.settings.script.withInline

```jsonnet
PipelineMetricAggregation.BucketScript.settings.script.withInline(value)
```

PARAMETERS:

* **value** (`string`)


##### obj PipelineMetricAggregation.BucketScript.type


###### fn PipelineMetricAggregation.BucketScript.type.withPipelineMetricAggregationType

```jsonnet
PipelineMetricAggregation.BucketScript.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj PipelineMetricAggregation.CumulativeSum


##### fn PipelineMetricAggregation.CumulativeSum.withField

```jsonnet
PipelineMetricAggregation.CumulativeSum.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.CumulativeSum.withHide

```jsonnet
PipelineMetricAggregation.CumulativeSum.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn PipelineMetricAggregation.CumulativeSum.withId

```jsonnet
PipelineMetricAggregation.CumulativeSum.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.CumulativeSum.withPipelineAgg

```jsonnet
PipelineMetricAggregation.CumulativeSum.withPipelineAgg(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.CumulativeSum.withSettings

```jsonnet
PipelineMetricAggregation.CumulativeSum.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn PipelineMetricAggregation.CumulativeSum.withSettingsMixin

```jsonnet
PipelineMetricAggregation.CumulativeSum.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn PipelineMetricAggregation.CumulativeSum.withType

```jsonnet
PipelineMetricAggregation.CumulativeSum.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.CumulativeSum.withTypeMixin

```jsonnet
PipelineMetricAggregation.CumulativeSum.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj PipelineMetricAggregation.CumulativeSum.settings


###### fn PipelineMetricAggregation.CumulativeSum.settings.withFormat

```jsonnet
PipelineMetricAggregation.CumulativeSum.settings.withFormat(value)
```

PARAMETERS:

* **value** (`string`)


##### obj PipelineMetricAggregation.CumulativeSum.type


###### fn PipelineMetricAggregation.CumulativeSum.type.withPipelineMetricAggregationType

```jsonnet
PipelineMetricAggregation.CumulativeSum.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj PipelineMetricAggregation.Derivative


##### fn PipelineMetricAggregation.Derivative.withField

```jsonnet
PipelineMetricAggregation.Derivative.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.Derivative.withHide

```jsonnet
PipelineMetricAggregation.Derivative.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn PipelineMetricAggregation.Derivative.withId

```jsonnet
PipelineMetricAggregation.Derivative.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.Derivative.withPipelineAgg

```jsonnet
PipelineMetricAggregation.Derivative.withPipelineAgg(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.Derivative.withSettings

```jsonnet
PipelineMetricAggregation.Derivative.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn PipelineMetricAggregation.Derivative.withSettingsMixin

```jsonnet
PipelineMetricAggregation.Derivative.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn PipelineMetricAggregation.Derivative.withType

```jsonnet
PipelineMetricAggregation.Derivative.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.Derivative.withTypeMixin

```jsonnet
PipelineMetricAggregation.Derivative.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj PipelineMetricAggregation.Derivative.settings


###### fn PipelineMetricAggregation.Derivative.settings.withUnit

```jsonnet
PipelineMetricAggregation.Derivative.settings.withUnit(value)
```

PARAMETERS:

* **value** (`string`)


##### obj PipelineMetricAggregation.Derivative.type


###### fn PipelineMetricAggregation.Derivative.type.withPipelineMetricAggregationType

```jsonnet
PipelineMetricAggregation.Derivative.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`


#### obj PipelineMetricAggregation.MovingAverage


##### fn PipelineMetricAggregation.MovingAverage.withField

```jsonnet
PipelineMetricAggregation.MovingAverage.withField(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.MovingAverage.withHide

```jsonnet
PipelineMetricAggregation.MovingAverage.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn PipelineMetricAggregation.MovingAverage.withId

```jsonnet
PipelineMetricAggregation.MovingAverage.withId(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.MovingAverage.withPipelineAgg

```jsonnet
PipelineMetricAggregation.MovingAverage.withPipelineAgg(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.MovingAverage.withSettings

```jsonnet
PipelineMetricAggregation.MovingAverage.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


##### fn PipelineMetricAggregation.MovingAverage.withSettingsMixin

```jsonnet
PipelineMetricAggregation.MovingAverage.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn PipelineMetricAggregation.MovingAverage.withType

```jsonnet
PipelineMetricAggregation.MovingAverage.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn PipelineMetricAggregation.MovingAverage.withTypeMixin

```jsonnet
PipelineMetricAggregation.MovingAverage.withTypeMixin(value)
```

PARAMETERS:

* **value** (`string`)


##### obj PipelineMetricAggregation.MovingAverage.type


###### fn PipelineMetricAggregation.MovingAverage.type.withPipelineMetricAggregationType

```jsonnet
PipelineMetricAggregation.MovingAverage.type.withPipelineMetricAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"moving_avg"`, `"moving_fn"`, `"derivative"`, `"serial_diff"`, `"cumulative_sum"`, `"bucket_script"`

