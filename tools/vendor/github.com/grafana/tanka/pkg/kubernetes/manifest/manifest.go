package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
	"github.com/stretchr/objx"
	yaml "gopkg.in/yaml.v2"
)

// Manifest represents a Kubernetes API object. The fields `apiVersion` and
// `kind` are required, `metadata.name` should be present as well
type Manifest map[string]interface{}

// New creates a new Manifest
func New(raw map[string]interface{}) (Manifest, error) {
	m := Manifest(raw)
	if err := m.Verify(); err != nil {
		return nil, err
	}
	return m, nil
}

// NewFromObj creates a new Manifest from an objx.Map
func NewFromObj(raw objx.Map) (Manifest, error) {
	return New(map[string]interface{}(raw))
}

// String returns the Manifest in yaml representation
func (m Manifest) String() string {
	y, err := yaml.Marshal(m)
	if err != nil {
		// this should never go wrong in normal operations
		panic(errors.Wrap(err, "formatting manifest"))
	}
	return string(y)
}

var (
	ErrInvalidStr = fmt.Errorf("missing or not of string type")
	ErrInvalidMap = fmt.Errorf("missing or not an object")
)

// Verify checks whether the manifest is correctly structured
func (m Manifest) Verify() error {
	o := m2o(m)
	fields := make(map[string]error)

	if !o.Get("kind").IsStr() {
		fields["kind"] = ErrInvalidStr
	}
	if !o.Get("apiVersion").IsStr() {
		fields["apiVersion"] = ErrInvalidStr
	}

	// Lists don't have `metadata`
	if !m.IsList() {
		if !o.Get("metadata").IsMSI() {
			fields["metadata"] = ErrInvalidMap
		}
		if !o.Get("metadata.name").IsStr() && !o.Get("metadata.generateName").IsStr() {
			fields["metadata.name"] = ErrInvalidStr
		}

		if err := verifyMSS(o.Get("metadata.labels").Data()); err != nil {
			fields["metadata.labels"] = err
		}
		if err := verifyMSS(o.Get("metadata.annotations").Data()); err != nil {
			fields["metadata.annotations"] = err
		}
	}

	if len(fields) == 0 {
		return nil
	}

	return &SchemaError{
		Fields:   fields,
		Manifest: m,
	}
}

// verifyMSS checks that ptr is either nil or a string map
func verifyMSS(ptr interface{}) error {
	if ptr == nil {
		return nil
	}

	switch t := ptr.(type) {
	case map[string]string:
		return nil
	case map[string]interface{}:
		for k, v := range t {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("contains non-string field '%s' of type '%T'", k, v)
			}
		}
		return nil
	default:
		return fmt.Errorf("must be object, but got '%T' instead", ptr)
	}
}

// IsList returns whether the manifest is a List type, containing other
// manifests as children. Code based on
// https://github.com/kubernetes/apimachinery/blob/61490fe38e784592212b24b9878306b09be45ab0/pkg/apis/meta/v1/unstructured/unstructured.go#L54
func (m Manifest) IsList() bool {
	items, ok := m["items"]
	if !ok {
		return false
	}
	_, ok = items.([]interface{})
	return ok
}

// Items returns list items if the manifest is of List type
func (m Manifest) Items() (List, error) {
	if !m.IsList() {
		return nil, fmt.Errorf("attempt to unwrap non-list object '%s' of kind '%s'", m.Metadata().Name(), m.Kind())
	}

	// This is safe, IsList() asserts this
	items := m["items"].([]interface{})
	list := make(List, 0, len(items))
	for _, i := range items {
		child, ok := i.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("unwrapped list item is not an object, but '%T'", child)
		}

		m := Manifest(child)
		list = append(list, m)
	}

	return list, nil
}

// Kind returns the kind of the API object
func (m Manifest) Kind() string {
	return m["kind"].(string)
}

// KindName returns kind and metadata.name in the `<kind>/<name>` format
func (m Manifest) KindName() string {
	return fmt.Sprintf("%s/%s",
		m.Kind(),
		m.Metadata().Name(),
	)
}

// APIVersion returns the version of the API this object uses
func (m Manifest) APIVersion() string {
	return m["apiVersion"].(string)
}

// Metadata returns the metadata of this object
func (m Manifest) Metadata() Metadata {
	if m["metadata"] == nil {
		m["metadata"] = make(map[string]interface{})
	}
	return Metadata(m["metadata"].(map[string]interface{}))
}

// UnmarshalJSON validates the Manifest during json parsing
func (m *Manifest) UnmarshalJSON(data []byte) error {
	type tmp Manifest
	var t tmp
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	*m = Manifest(t)
	return m.Verify()
}

// UnmarshalYAML validates the Manifest during yaml parsing
func (m *Manifest) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tmp map[string]interface{}
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, m)
}

// Metadata is the metadata object from the Manifest
type Metadata map[string]interface{}

// Name of the manifest
func (m Metadata) Name() string {
	name, ok := m["name"]
	if !ok {
		return ""
	}
	return name.(string)
}

func (m Metadata) GenerateName() string {
	generateName, ok := m["generateName"]
	if !ok {
		return ""
	}
	return generateName.(string)
}

// HasNamespace returns whether the manifest has a namespace set
func (m Metadata) HasNamespace() bool {
	return m2o(m).Get("namespace").IsStr()
}

// Namespace of the manifest
func (m Metadata) Namespace() string {
	if !m.HasNamespace() {
		return ""
	}
	return m["namespace"].(string)
}

func (m Metadata) UID() string {
	uid, ok := m["uid"].(string)
	if !ok {
		return ""
	}
	return uid
}

// Labels of the manifest
func (m Metadata) Labels() map[string]interface{} {
	return safeMSI(m, "labels")
}

// Annotations of the manifest
func (m Metadata) Annotations() map[string]interface{} {
	return safeMSI(m, "annotations")
}

// Managed fields of the manifest
func (m Metadata) ManagedFields() []interface{} {
	items, ok := m["managedFields"]
	if !ok {
		return make([]interface{}, 0)
	}
	list, ok := items.([]interface{})
	if !ok {
		return make([]interface{}, 0)
	}
	return list
}

func safeMSI(m map[string]interface{}, key string) map[string]interface{} {
	switch t := m[key].(type) {
	case map[string]interface{}:
		return t
	default:
		m[key] = make(map[string]interface{})
		return m[key].(map[string]interface{})
	}
}

// List of individual Manifests
type List []Manifest

// String returns the List as a yaml stream. In case of an error, it is
// returned as a string instead.
func (m List) String() string {
	buf := bytes.Buffer{}
	enc := yaml.NewEncoder(&buf)

	for _, d := range m {
		if err := enc.Encode(d); err != nil {
			// This should never happen in normal operations
			panic(errors.Wrap(err, "formatting manifests"))
		}
	}

	return buf.String()
}

func (m List) Namespaces() []string {
	namespaces := map[string]struct{}{}
	for _, manifest := range m {
		if namespace := manifest.Metadata().Namespace(); namespace != "" {
			namespaces[namespace] = struct{}{}
		}
	}
	keys := []string{}
	for k := range namespaces {
		keys = append(keys, k)
	}
	return keys
}

func m2o(m interface{}) objx.Map {
	switch mm := m.(type) {
	case Metadata:
		return objx.New(map[string]interface{}(mm))
	case Manifest:
		return objx.New(map[string]interface{}(mm))
	}
	return nil
}

// DefaultNameFormat to use when no nameFormat is supplied
const DefaultNameFormat = `{{ print .kind "_" .metadata.name | snakecase }}`

func ListAsMap(list List, nameFormat string) (map[string]interface{}, error) {
	if nameFormat == "" {
		nameFormat = DefaultNameFormat
	}

	tmpl, err := template.New("").
		Funcs(sprig.TxtFuncMap()).
		Parse(nameFormat)
	if err != nil {
		return nil, fmt.Errorf("parsing name format: %w", err)
	}

	out := make(map[string]interface{})
	for _, m := range list {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, m); err != nil {
			return nil, err
		}
		name := buf.String()

		if _, ok := out[name]; ok {
			return nil, ErrorDuplicateName{name: name, format: nameFormat}
		}
		out[name] = map[string]interface{}(m)
	}

	return out, nil
}
