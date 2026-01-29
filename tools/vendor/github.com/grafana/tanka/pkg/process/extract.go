package process

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/objx"
	"gopkg.in/yaml.v3"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
)

// Extract scans the raw Jsonnet evaluation result (JSON tree) for objects that
// look like Kubernetes objects and extracts those into a flat map, indexed by
// their path in the original JSON tree
func Extract(raw interface{}) (map[string]manifest.Manifest, error) {
	extracted := make(map[string]manifest.Manifest)
	if err := walkJSON(raw, extracted, nil); err != nil {
		return nil, err
	}
	return extracted, nil
}

// walkJSON recurses into either a map or list, returning a list of all objects that look
// like kubernetes resources. We support resources at an arbitrary level of nesting, and
// return an error if a node is not walkable.
//
// Handling the different types is quite gross, so we split this method into a generic
// walkJSON, and then walkObj/walkList to handle the two different types of collection we
// support.
func walkJSON(ptr interface{}, extracted map[string]manifest.Manifest, path trace) error {
	// check for known types
	switch v := ptr.(type) {
	case map[string]interface{}:
		return walkObj(v, extracted, path)
	case []interface{}:
		return walkList(v, extracted, path)
	}

	log.Debug().Msgf("recursion ended on key %q of type %T which does not belong to a valid Kubernetes object", path.Name(), ptr)
	return ErrorPrimitiveReached{
		path:      path.Base(),
		key:       path.Name(),
		primitive: ptr,
	}
}

func walkList(list []interface{}, extracted map[string]manifest.Manifest, path trace) error {
	for idx, value := range list {
		err := walkJSON(value, extracted, append(path, fmt.Sprintf("[%d]", idx)))
		if err != nil {
			return err
		}
	}
	return nil
}

func walkObj(obj objx.Map, extracted map[string]manifest.Manifest, path trace) error {
	obj = obj.Exclude([]string{"__ksonnet"}) // remove our private ksonnet field

	// This looks like a kubernetes manifest, so make one and return it
	ok, manifestErr := isKubernetesManifest(obj)
	if ok {
		m, err := manifest.NewFromObj(obj)
		var e *manifest.SchemaError
		if errors.As(err, &e) {
			e.Name = path.Full()
			return e
		}

		extracted[path.Full()] = m
		return nil
	}

	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		path := append(path, key)
		if obj[key] == nil { // result from false if condition in Jsonnet
			continue
		}
		err := walkJSON(obj[key], extracted, path)
		if err != nil {
			if err, ok := err.(ErrorPrimitiveReached); ok {
				return err.WithContainingObj(obj, manifestErr)
			}

			return err
		}
	}

	return nil
}

type trace []string

func (t trace) Full() string {
	return "." + strings.Join(t, ".")
}

func (t trace) Base() string {
	if len(t) > 0 {
		t = t[:len(t)-1]
	}
	return "." + strings.Join(t, ".")
}

func (t trace) Name() string {
	if len(t) > 0 {
		return t[len(t)-1]
	}

	return ""
}

// ErrorPrimitiveReached occurs when walkJSON reaches the end of nested dicts without finding a valid Kubernetes manifest
type ErrorPrimitiveReached struct {
	path, key        string
	primitive        interface{}
	containingObj    objx.Map
	containingObjErr error
}

func (e ErrorPrimitiveReached) WithContainingObj(obj objx.Map, err error) ErrorPrimitiveReached {
	if e.containingObj == nil {
		e.containingObj = obj
		e.containingObjErr = err
	}
	return e
}

func (e ErrorPrimitiveReached) Error() string {
	errMessage := fmt.Sprintf(`found invalid Kubernetes object (at %s): %s`, e.path, e.containingObjErr)

	container, err := yaml.Marshal(e.containingObj)
	if err != nil {
		log.Error().Msgf("failed to marshal invalid Kubernetes object: %s", err)
	} else {
		errMessage += "\n\n" + string(container)
	}

	return errMessage
}

// isKubernetesManifest attempts to infer whether the given object is a valid
// kubernetes resource by verifying the presence of apiVersion and kind. These
// two fields are required for kubernetes to accept any resource.
// The error return value indicates the reason why the object is not a valid
func isKubernetesManifest(obj objx.Map) (ok bool, err error) {
	checkAttribute := func(key string) error {
		v := obj.Get(key)
		if v.IsNil() {
			return fmt.Errorf("missing attribute %q", key)
		}
		if !v.IsStr() {
			return fmt.Errorf("attribute %q is not a string, it is a %s", key, reflect.TypeOf(v.Data()))
		}
		if v.Str() == "" {
			return fmt.Errorf("attribute %q is empty", key)
		}
		return nil
	}

	if err := checkAttribute("apiVersion"); err != nil {
		return false, err
	}
	if err := checkAttribute("kind"); err != nil {
		return false, err
	}

	return true, nil
}
