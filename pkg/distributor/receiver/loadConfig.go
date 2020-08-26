package receiver

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/receiver"
)

// extracted from otel collector : https://github.com/open-telemetry/opentelemetry-collector/blob/master/config/config.go

// These are errors that can be returned by Load(). Note that error codes are not part
// of Load()'s public API, they are for internal unit testing only.
type configErrorCode int

const (
	_ configErrorCode = iota // skip 0, start errors codes from 1.
	errInvalidTypeAndNameKey
	errUnknownReceiverType
	errDuplicateReceiverName
	errMissingReceivers
	errUnmarshalErrorOnReceiver
)

// typeAndNameSeparator is the separator that is used between type and name in type/name composite keys.
const typeAndNameSeparator = "/"

type configError struct {
	msg  string          // human readable error message.
	code configErrorCode // internal error code.
}

func (e *configError) Error() string {
	return e.msg
}

func loadReceivers(v *viper.Viper, keyMap map[string]interface{}, factories map[string]receiver.Factory) (configmodels.Receivers, error) {

	// Currently there is no default receiver enabled. The configuration must specify at least one receiver to enable
	// functionality.
	if len(keyMap) == 0 {
		return nil, &configError{
			code: errMissingReceivers,
			msg:  "no receivers specified in config",
		}
	}

	// Prepare resulting map
	receivers := make(configmodels.Receivers)

	// Iterate over input map and create a config for each.
	for key := range keyMap {
		// Decode the key into type and fullName components.
		typeStr, fullName, err := decodeTypeAndName(key)
		if err != nil || typeStr == "" {
			return nil, &configError{
				code: errInvalidTypeAndNameKey,
				msg:  fmt.Sprintf("invalid key %q: %s", key, err.Error()),
			}
		}

		// Find receiver factory based on "type" that we read from config source
		factory := factories[typeStr]
		if factory == nil {
			return nil, &configError{
				code: errUnknownReceiverType,
				msg:  fmt.Sprintf("unknown receiver type %q", typeStr),
			}
		}

		// Create the default config for this receiver.
		receiverCfg := factory.CreateDefaultConfig()
		receiverCfg.SetType(typeStr)
		receiverCfg.SetName(fullName)

		// Unmarshal only the subconfig for this exporter.
		sv := getConfigSection(v, key)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		customUnmarshaler := factory.CustomUnmarshaler()
		if customUnmarshaler != nil {
			// This configuration requires a custom unmarshaler, use it.
			err = customUnmarshaler(v, key, sv, receiverCfg)
		} else {
			err = sv.UnmarshalExact(receiverCfg)
		}

		if err != nil {
			return nil, &configError{
				code: errUnmarshalErrorOnReceiver,
				msg:  fmt.Sprintf("error reading settings for receiver type %q: %v", typeStr, err),
			}
		}

		if receivers[fullName] != nil {
			return nil, &configError{
				code: errDuplicateReceiverName,
				msg:  fmt.Sprintf("duplicate receiver name %q", fullName),
			}
		}
		receivers[fullName] = receiverCfg
	}

	return receivers, nil
}

// getConfigSection returns a sub-config from the viper config that has the corresponding given key.
// It also expands all the string values.
func getConfigSection(v *viper.Viper, key string) *viper.Viper {
	// Unmarshal only the subconfig for this processor.
	sv := v.Sub(key)
	if sv == nil {
		// When the config for this key is empty Sub returns nil. In order to avoid nil checks
		// just return an empty config.
		return viper.New()
	}
	// Before unmarshaling first expand all environment variables.
	return expandEnvConfig(sv)
}

// expandEnvConfig creates a new viper config with expanded values for all the values (simple, list or map value).
// It does not expand the keys.
// Need to copy everything because of a bug in Viper: Set a value "map[string]interface{}" where a key has a ".",
// then AllSettings will return the previous value not the newly set one.
func expandEnvConfig(v *viper.Viper) *viper.Viper {
	newCfg := make(map[string]interface{})
	for k, val := range v.AllSettings() {
		newCfg[k] = expandStringValues(val)
	}
	newVip := viper.New()
	newVip.MergeConfigMap(newCfg) //nolint:errcheck
	return newVip
}

func expandStringValues(value interface{}) interface{} {
	switch v := value.(type) {
	default:
		return v
	case string:
		return os.ExpandEnv(v)
	case []interface{}:
		// Viper treats all the slices as []interface{} (at least in what the otelcol tests).
		nslice := make([]interface{}, 0, len(v))
		for _, vint := range v {
			nslice = append(nslice, expandStringValues(vint))
		}
		return nslice
	case map[string]interface{}:
		nmap := make(map[string]interface{}, len(v))
		// Viper treats all the maps as [string]interface{} (at least in what the otelcol tests).
		for k, vint := range v {
			nmap[k] = expandStringValues(vint)
		}
		return nmap
	}
}

// decodeTypeAndName decodes a key in type[/name] format into type and fullName.
// fullName is the key normalized such that type and name components have spaces trimmed.
// The "type" part must be present, the forward slash and "name" are optional.
func decodeTypeAndName(key string) (typeStr, fullName string, err error) {
	items := strings.SplitN(key, typeAndNameSeparator, 2)

	if len(items) >= 1 {
		typeStr = strings.TrimSpace(items[0])
	}

	if len(items) < 1 || typeStr == "" {
		err = errors.New("type/name key must have the type part")
		return
	}

	var nameSuffix string
	if len(items) > 1 {
		// "name" part is present.
		nameSuffix = strings.TrimSpace(items[1])
		if nameSuffix == "" {
			err = errors.New("name part must be specified after " + typeAndNameSeparator + " in type/name key")
			return
		}
	} else {
		nameSuffix = ""
	}

	// Create normalized fullName.
	if nameSuffix == "" {
		fullName = typeStr
	} else {
		fullName = typeStr + typeAndNameSeparator + nameSuffix
	}

	err = nil
	return
}
