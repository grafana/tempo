package native

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/tanka/pkg/helm"
	"github.com/grafana/tanka/pkg/kustomize"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
)

// Funcs returns a slice of native Go functions that shall be available
// from Jsonnet using `std.nativeFunc`
func Funcs() []*jsonnet.NativeFunction {
	return []*jsonnet.NativeFunction{
		// Parse serialized data into dicts
		parseJSON(),
		parseYAML(),

		// Convert serializations
		manifestJSONFromJSON(),
		manifestYAMLFromJSON(),

		// Regular expressions
		escapeStringRegex(),
		regexMatch(),
		regexSubst(),

		// Hash functions
		hashSha256(),

		helm.NativeFunc(helm.ExecHelm{}),
		kustomize.NativeFunc(kustomize.ExecKustomize{}),
	}
}

// parseJSON wraps `json.Unmarshal` to convert a json string into a dict
func parseJSON() *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "parseJson",
		Params: ast.Identifiers{"json"},
		Func: func(dataString []interface{}) (res interface{}, err error) {
			data := []byte(dataString[0].(string))
			err = json.Unmarshal(data, &res)
			return
		},
	}
}

func hashSha256() *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "sha256",
		Params: ast.Identifiers{"str"},
		Func: func(dataString []interface{}) (interface{}, error) {
			h := sha256.New()
			h.Write([]byte(dataString[0].(string)))
			return fmt.Sprintf("%x", h.Sum(nil)), nil
		},
	}
}

// parseYAML wraps `yaml.Unmarshal` to convert a string of yaml document(s) into a (set of) dicts
func parseYAML() *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "parseYaml",
		Params: ast.Identifiers{"yaml"},
		Func: func(dataString []interface{}) (interface{}, error) {
			data := []byte(dataString[0].(string))
			ret := []interface{}{}

			d := yaml.NewDecoder(bytes.NewReader(data))
			for {
				var doc, jsonDoc interface{}
				if err := d.Decode(&doc); err != nil {
					if err == io.EOF {
						break
					}
					return nil, errors.Wrap(err, "parsing yaml")
				}

				jsonRaw, err := json.Marshal(doc)
				if err != nil {
					return nil, errors.Wrap(err, "converting yaml to json")
				}

				if err := json.Unmarshal(jsonRaw, &jsonDoc); err != nil {
					return nil, errors.Wrap(err, "converting yaml to json")
				}

				ret = append(ret, jsonDoc)
			}

			return ret, nil
		},
	}
}

// manifestJSONFromJSON reserializes JSON which allows to change the indentation.
func manifestJSONFromJSON() *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "manifestJsonFromJson",
		Params: ast.Identifiers{"json", "indent"},
		Func: func(data []interface{}) (interface{}, error) {
			indent := int(data[1].(float64))
			dataBytes := []byte(data[0].(string))
			dataBytes = bytes.TrimSpace(dataBytes)
			buf := bytes.Buffer{}
			if err := json.Indent(&buf, dataBytes, "", strings.Repeat(" ", indent)); err != nil {
				return "", err
			}
			buf.WriteString("\n")
			return buf.String(), nil
		},
	}
}

// manifestYamlFromJSON serializes a JSON string as a YAML document
func manifestYAMLFromJSON() *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "manifestYamlFromJson",
		Params: ast.Identifiers{"json"},
		Func: func(data []interface{}) (interface{}, error) {
			var input interface{}
			dataBytes := []byte(data[0].(string))
			if err := json.Unmarshal(dataBytes, &input); err != nil {
				return "", err
			}
			output, err := yaml.Marshal(input)
			return string(output), err
		},
	}
}

// escapeStringRegex escapes all regular expression metacharacters
// and returns a regular expression that matches the literal text.
func escapeStringRegex() *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "escapeStringRegex",
		Params: ast.Identifiers{"str"},
		Func: func(s []interface{}) (interface{}, error) {
			return regexp.QuoteMeta(s[0].(string)), nil
		},
	}
}

// regexMatch returns whether the given string is matched by the given re2 regular expression.
func regexMatch() *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "regexMatch",
		Params: ast.Identifiers{"regex", "string"},
		Func: func(s []interface{}) (interface{}, error) {
			return regexp.MatchString(s[0].(string), s[1].(string))
		},
	}
}

// regexSubst replaces all matches of the re2 regular expression with another string.
func regexSubst() *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "regexSubst",
		Params: ast.Identifiers{"regex", "src", "repl"},
		Func: func(data []interface{}) (interface{}, error) {
			regex, src, repl := data[0].(string), data[1].(string), data[2].(string)

			r, err := regexp.Compile(regex)
			if err != nil {
				return "", err
			}
			return r.ReplaceAllString(src, repl), nil
		},
	}
}
