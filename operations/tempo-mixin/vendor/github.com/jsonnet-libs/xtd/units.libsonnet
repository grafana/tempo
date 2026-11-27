local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';

{
  '#': d.pkg(
    name='units',
    url='github.com/jsonnet-libs/xtd/units.libsonnet',
    help='`units` implements helper functions for converting units.',
  ),

  '#parseKubernetesCPU':: d.fn(
    |||
      `parseKubernetesCPU` parses a Kubernetes CPU string/number into a number of cores.
       The function assumes the input is in a correct Kubernetes format, i.e., an integer, a float,
       a string representation of an integer or a float, or a string containing a number ending with 'm'
       representing a number of millicores.
    |||,
    [
      d.arg('input', d.T.any),
    ]
  ),
  parseKubernetesCPU(input):
    if std.isString(input) && std.endsWith(input, 'm') then std.parseJson(std.rstripChars(input, 'm')) / 1000
    else if std.isString(input) then std.parseJson(input)
    else if std.isNumber(input) then input
    else 0,

  '#siToBytes':: d.fn(
    |||
      `siToBytes` converts Kubernetes byte units to bytes.
      Only works for limited set of SI prefixes: Ki, Mi, Gi, Ti.
    |||,
    [
      d.arg('str', d.T.string),
    ]
  ),
  siToBytes(str):
    local siToBytesDecimal(str) =
      if std.endsWith(str, 'Ki') then
        std.parseJson(std.rstripChars(str, 'Ki')) * std.pow(2, 10)
      else if std.endsWith(str, 'Mi') then
        std.parseJson(std.rstripChars(str, 'Mi')) * std.pow(2, 20)
      else if std.endsWith(str, 'Gi') then
        std.parseJson(std.rstripChars(str, 'Gi')) * std.pow(2, 30)
      else if std.endsWith(str, 'Ti') then
        std.parseJson(std.rstripChars(str, 'Ti')) * std.pow(2, 40)
      else
        std.parseJson(str);
    std.floor(siToBytesDecimal(str)),

  '#parseDuration':: d.fn(
    |||
      `parseDuration` parses a duration string and returns the number of seconds.
      Handles milliseconds (ms), seconds (s), minutes (m), hours (h), and combined formats like "4m30s" or "1h30m".
    |||,
    [
      d.arg('duration', d.T.string),
    ]
  ),
  parseDuration(duration):
    local replaced = std.strReplace(duration, 'ms', 'n');  // replace ms with n to avoid confusion with minutes

    local splitH = std.split(replaced, 'h');
    local hours = if std.length(splitH) > 1 then std.parseJson(splitH[0]) else 0;
    local withoutHours = if std.length(splitH) > 1 then splitH[1] else replaced;

    local splitM = std.split(withoutHours, 'm');
    local minutes = if std.length(splitM) > 1 then std.parseJson(splitM[0]) else 0;
    local withoutMinutes = if std.length(splitM) > 1 then splitM[1] else withoutHours;

    local splitS = std.split(withoutMinutes, 's');
    local seconds = if std.length(splitS) > 1 then std.parseJson(splitS[0]) else 0;
    local withoutSeconds = if std.length(splitS) > 1 then splitS[1] else withoutMinutes;

    local splitN = std.split(withoutSeconds, 'n');
    local milliseconds = if std.length(splitN) > 1 then std.parseJson(splitN[0]) else 0;

    hours * 3600 + minutes * 60 + seconds + milliseconds / 1000,

  '#formatDuration':: d.fn(
    |||
      `formatDuration` formats a number of seconds into a human-readable duration string.
      Returns the duration in the smallest appropriate unit (s, m, h, or combined formats like "4m30s").
    |||,
    [
      d.arg('seconds', d.T.number),
    ]
  ),
  formatDuration(seconds):
    (if std.floor(seconds / 3600) > 0 then '%dh' % (seconds / 3600) else '') +
    (if std.floor((seconds % 3600) / 60) > 0 then '%dm' % ((seconds % 3600) / 60) else '') +
    (if (seconds % 60) > 1 then '%ds' % (seconds % 60) else '') +
    (if seconds % 1 > 0 then '%dms' % ((seconds % 1) * 1000) else ''),
}
