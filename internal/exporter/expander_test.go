package exporter

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

type expanderTestConfig struct{}

func (expanderTestConfig) IsCloning() bool                  { return false }
func (expanderTestConfig) GetLogLevel() int                 { return 0 }
func (expanderTestConfig) GetStrategy() string              { return "" }
func (expanderTestConfig) GetRepoURL() string               { return "" }
func (expanderTestConfig) GetRepoSSHKey() string            { return "" }
func (expanderTestConfig) GetRepoSSHUser() string           { return "" }
func (expanderTestConfig) GetRepoBranch() string            { return "" }
func (expanderTestConfig) GetRepoRemoteName() string        { return "" }
func (expanderTestConfig) GetRepoBasePath() string          { return "" }
func (expanderTestConfig) GetRepoRootDir() string           { return "" }
func (expanderTestConfig) GetConsulURL() string             { return "" }
func (expanderTestConfig) GetConsulACL() string             { return "" }
func (expanderTestConfig) GetConsulBasePath() string        { return "" }
func (expanderTestConfig) ShouldExpandJSON() bool           { return true }
func (expanderTestConfig) ShouldExpandYAML() bool           { return true }
func (expanderTestConfig) DoSecrets() bool                  { return false }
func (expanderTestConfig) GetSecretsMap() map[string]string { return nil }
func (expanderTestConfig) AllowDeletes() string             { return "" }
func (expanderTestConfig) GetPollInterval() int             { return 0 }
func (expanderTestConfig) WorkingChan() chan bool           { return nil }
func (expanderTestConfig) GetValidExtensions() []string     { return nil }
func (expanderTestConfig) KeepFileExt() bool                { return false }
func (expanderTestConfig) GetTimeout() int                  { return 0 }
func (expanderTestConfig) GetHookAddr() string              { return "" }
func (expanderTestConfig) GetDryRunOutput() string          { return "" }
func (expanderTestConfig) IsShowVersion() bool              { return false }

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestExpandYAMLSerializesStringArrayAsJSON(t *testing.T) {
	exp := &exporter{config: expanderTestConfig{}}
	localData := map[string]string{}

	requireNoError(t, exp.expandYAML("config", `whitelist:
  enabled: true
  ranges:
    - "U+E000..U+F8FF"
    - "U+F0000..U+FFFFF"
    - "U+100000..U+10FFFF"
`, localData))

	assertJSONArray(t, localData["config/whitelist/ranges"], []interface{}{
		"U+E000..U+F8FF",
		"U+F0000..U+FFFFF",
		"U+100000..U+10FFFF",
	})

	if localData["config/whitelist/enabled"] != "true" {
		t.Fatalf("expected scalar bool to be preserved as true, got %q", localData["config/whitelist/enabled"])
	}
}

func TestExpandJSONSerializesStringArrayAsJSON(t *testing.T) {
	exp := &exporter{config: expanderTestConfig{}}
	localData := map[string]string{}

	requireNoError(t, exp.expandJSON("config", `{
  "whitelist": {
    "ranges": [
      "U+E000..U+F8FF",
      "U+F0000..U+FFFFF",
      "U+100000..U+10FFFF"
    ]
  }
}`, localData))

	assertJSONArray(t, localData["config/whitelist/ranges"], []interface{}{
		"U+E000..U+F8FF",
		"U+F0000..U+FFFFF",
		"U+100000..U+10FFFF",
	})
}

func TestExpandDocumentsStoreNullScalarAsJSONNull(t *testing.T) {
	exp := &exporter{config: expanderTestConfig{}}

	yamlData := map[string]string{}
	requireNoError(t, exp.expandYAML("config", "value: null\n", yamlData))
	if yamlData["config/value"] != "null" {
		t.Fatalf("expected YAML null scalar to be stored as null, got %q", yamlData["config/value"])
	}

	jsonData := map[string]string{}
	requireNoError(t, exp.expandJSON("config", `{"value": null}`, jsonData))
	if jsonData["config/value"] != "null" {
		t.Fatalf("expected JSON null scalar to be stored as null, got %q", jsonData["config/value"])
	}
}

func TestExpandYAMLStoresNumericValuesFromIssue35(t *testing.T) {
	exp := &exporter{config: expanderTestConfig{}}
	localData := map[string]string{}

	requireNoError(t, exp.expandYAML("deployments/environments/production", `list1: value1,value2
port1: 7777
url1: https://some.url
url2: other.url.com
url3: yet-another.url.com
list2: value3,value4
ip1: 10.2.3.5
port2: 36482
`, localData))

	want := map[string]string{
		"deployments/environments/production/list1": "value1,value2",
		"deployments/environments/production/port1": "7777",
		"deployments/environments/production/url1":  "https://some.url",
		"deployments/environments/production/url2":  "other.url.com",
		"deployments/environments/production/url3":  "yet-another.url.com",
		"deployments/environments/production/list2": "value3,value4",
		"deployments/environments/production/ip1":   "10.2.3.5",
		"deployments/environments/production/port2": "36482",
	}

	if !reflect.DeepEqual(localData, want) {
		t.Fatalf("unexpected expanded YAML data\nexpected: %#v\nactual:   %#v", want, localData)
	}
}

func TestParseFileExpandsYMLFiles(t *testing.T) {
	exp := &exporter{config: expanderTestConfig{}}
	localData := map[string]string{}

	requireNoError(t, exp.parseFile("config.yml", "name: service\nvalues:\n  - one\n  - two\n", localData))

	if localData["config/name"] != "service" {
		t.Fatalf("expected .yml scalar to be expanded, got %q", localData["config/name"])
	}
	assertJSONArray(t, localData["config/values"], []interface{}{"one", "two"})
	if _, exists := localData["config"]; exists {
		t.Fatalf("expected .yml file not to be stored as a single blob")
	}
}

func TestExpandYAMLPreservesBlockScalarQuotesAndBackslashes(t *testing.T) {
	exp := &exporter{config: expanderTestConfig{}}
	localData := map[string]string{}

	requireNoError(t, exp.expandYAML("config", `skip-cert-issuers: |
  E=ca@sertum.ru,1.2.643.100.1=1116673008539,1.2.643.100.4=6673240328,C=RU,ST=66 Свердловская область,L=Екатеринбург,STREET=ул. Малопрудная\, стр. 5\, офис 715,OU=Удостоверяющий центр,O=ООО \"Сертум-Про\",CN=ООО \"Сертум-Про\"
  E=ca@sertum.ru,1.2.643.100.1=1116673008539,1.2.643.100.4=6673240328,C=RU,ST=66 Свердловская область,L=Екатеринбург,STREET=ул. Малопрудная\, стр. 5\, офис 715,OU=Удостоверяющий центр,O=ООО "Сертум-Про",CN=ООО "Сертум-Про"
`, localData))

	want := "E=ca@sertum.ru,1.2.643.100.1=1116673008539,1.2.643.100.4=6673240328,C=RU,ST=66 Свердловская область,L=Екатеринбург,STREET=ул. Малопрудная\\, стр. 5\\, офис 715,OU=Удостоверяющий центр,O=ООО \\\"Сертум-Про\\\",CN=ООО \\\"Сертум-Про\\\"\n" +
		"E=ca@sertum.ru,1.2.643.100.1=1116673008539,1.2.643.100.4=6673240328,C=RU,ST=66 Свердловская область,L=Екатеринбург,STREET=ул. Малопрудная\\, стр. 5\\, офис 715,OU=Удостоверяющий центр,O=ООО \"Сертум-Про\",CN=ООО \"Сертум-Про\"\n"

	if localData["config/skip-cert-issuers"] != want {
		t.Fatalf("block scalar changed\nexpected: %q\nactual:   %q", want, localData["config/skip-cert-issuers"])
	}
}

func TestValidateDocumentsRejectRootArrays(t *testing.T) {
	exp := &exporter{config: expanderTestConfig{}}

	assertValidationError(t, func() error {
		_, err := exp.validateYAML("config", "- one\n- two\n")
		return err
	}, "root document must be an object")

	assertValidationError(t, func() error {
		_, err := exp.validateJSON("config", `["one", "two"]`)
		return err
	}, "root document must be an object")
}

func TestExpandYAMLSerializesArrayTypesAsJSON(t *testing.T) {
	exp := &exporter{config: expanderTestConfig{}}
	localData := map[string]string{}

	requireNoError(t, exp.expandYAML("config", `strings:
  - one
  - two
numbers:
  - 1
  - 2
  - 3.5
booleans:
  - true
  - false
mixed:
  - one
  - 2
  - true
  - null
nested:
  - name: first
    enabled: true
  - name: second
    enabled: false
arrays:
  - [one, two]
  - [three, four]
empty: []
values:
  - "hello world"
  - "a,b"
  - 'say "hello"'
  - 'C:\temp\file'
  - "\u0441\u0442\u0440\u043e\u043a\u0430 \u043d\u0430 \u0440\u0443\u0441\u0441\u043a\u043e\u043c"
  - ""
parent:
  child:
    list:
      - nested
    scalar: keep-me
`, localData))

	assertJSONArray(t, localData["config/strings"], []interface{}{"one", "two"})
	assertJSONArray(t, localData["config/numbers"], []interface{}{float64(1), float64(2), 3.5})
	assertJSONArray(t, localData["config/booleans"], []interface{}{true, false})
	assertJSONArray(t, localData["config/mixed"], []interface{}{"one", float64(2), true, nil})
	assertJSONArray(t, localData["config/nested"], []interface{}{
		map[string]interface{}{"name": "first", "enabled": true},
		map[string]interface{}{"name": "second", "enabled": false},
	})
	assertJSONArray(t, localData["config/arrays"], []interface{}{
		[]interface{}{"one", "two"},
		[]interface{}{"three", "four"},
	})
	assertJSONArray(t, localData["config/empty"], []interface{}{})
	assertJSONArray(t, localData["config/values"], []interface{}{
		"hello world",
		"a,b",
		`say "hello"`,
		`C:\temp\file`,
		"\u0441\u0442\u0440\u043e\u043a\u0430 \u043d\u0430 \u0440\u0443\u0441\u0441\u043a\u043e\u043c",
		"",
	})
	assertJSONArray(t, localData["config/parent/child/list"], []interface{}{"nested"})

	if localData["config/parent/child/scalar"] != "keep-me" {
		t.Fatalf("expected nested scalar to be preserved, got %q", localData["config/parent/child/scalar"])
	}
}

func TestExpandDocumentFlattensAndSerializesLeaves(t *testing.T) {
	exp := &exporter{config: expanderTestConfig{}}
	localData := map[string]string{}

	document := map[string]interface{}{
		"parent": map[string]interface{}{
			"scalar": "keep-me",
			"array":  []interface{}{"one", "two"},
		},
	}

	requireNoError(t, exp.expandDocument("config", document, localData))

	if localData["config/parent/scalar"] != "keep-me" {
		t.Fatalf("expected scalar leaf to be preserved, got %q", localData["config/parent/scalar"])
	}
	assertJSONArray(t, localData["config/parent/array"], []interface{}{"one", "two"})
}

func TestSerializeCollectionReturnsMarshalError(t *testing.T) {
	_, err := serializeCollection([]interface{}{func() {}})
	if err == nil {
		t.Fatal("expected serializeCollection to return an error for unsupported JSON values")
	}
}

func TestSerializeValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "string", value: "hello", want: "hello"},
		{name: "bool", value: true, want: "true"},
		{name: "nil", value: nil, want: "null"},
		{name: "float64", value: 1.5, want: "1.5"},
		{name: "float32", value: float32(1.5), want: "1.5"},
		{name: "int", value: 42, want: "42"},
		{name: "int8", value: int8(8), want: "8"},
		{name: "int16", value: int16(16), want: "16"},
		{name: "int32", value: int32(32), want: "32"},
		{name: "int64", value: int64(64), want: "64"},
		{name: "uint", value: uint(42), want: "42"},
		{name: "uint8", value: uint8(8), want: "8"},
		{name: "uint16", value: uint16(16), want: "16"},
		{name: "uint32", value: uint32(32), want: "32"},
		{name: "uint64", value: uint64(64), want: "64"},
		{name: "array", value: []interface{}{"one", "two"}, want: `["one","two"]`},
	}

	for _, test := range tests {
		got, err := serializeValue(test.value)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", test.name, err)
		}
		if got != test.want {
			t.Fatalf("%s: got %q, want %q", test.name, got, test.want)
		}
	}
}

func TestSerializeValueReturnsUnsupportedTypeError(t *testing.T) {
	_, err := serializeValue(struct{}{})
	if err == nil {
		t.Fatal("expected unsupported value type error")
	}
}

func assertValidationError(t *testing.T, fn func() error, want string) {
	t.Helper()

	err := fn()
	if err == nil {
		t.Fatalf("expected validation to fail")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected validation error to contain %q, got %q", want, err.Error())
	}
}

func assertJSONArray(t *testing.T, value string, expected []interface{}) {
	t.Helper()

	var got []interface{}
	if err := json.Unmarshal([]byte(value), &got); err != nil {
		t.Fatalf("expected %q to be a valid JSON array: %v", value, err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected JSON array\nexpected: %#v\nactual:   %#v", expected, got)
	}
}
