package importer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/grizzlybite/gonsul/internal/entities"
	"github.com/grizzlybite/gonsul/tests/mocks"
)

func TestCreateOperationMatrixUsesSortedLocalKeys(t *testing.T) {
	cfg := &mocks.IConfig{}
	cfg.On("DoSecrets").Return(false)
	imp := &importer{config: cfg}

	matrix, err := imp.createOperationMatrix(map[string]string{}, map[string]string{
		"zeta":  "last",
		"alpha": "first",
		"gamma": "middle",
	})
	if err != nil {
		t.Fatal(err)
	}

	operations := matrix.GetOperations()
	got := []string{operations[0].GetPath(), operations[1].GetPath(), operations[2].GetPath()}
	want := []string{"alpha", "gamma", "zeta"}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected operation order: got %#v, want %#v", got, want)
		}
	}
}

func TestCreateOperationMatrixUsesSortedLiveKeysForDeletes(t *testing.T) {
	cfg := &mocks.IConfig{}
	cfg.On("AllowDeletes").Return("true")
	imp := &importer{config: cfg}

	matrix, err := imp.createOperationMatrix(map[string]string{
		"zeta":  "last",
		"alpha": "first",
		"gamma": "middle",
	}, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	operations := matrix.GetOperations()
	got := []string{operations[0].GetPath(), operations[1].GetPath(), operations[2].GetPath()}
	want := []string{"alpha", "gamma", "zeta"}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected delete order: got %#v, want %#v", got, want)
		}
	}
}

func TestCreateOperationMatrixPreservesQuotedValues(t *testing.T) {
	cfg := &mocks.IConfig{}
	cfg.On("DoSecrets").Return(false)
	imp := &importer{config: cfg}

	value := "O=ООО \\\"Сертум-Про\\\",CN=ООО \\\"Сертум-Про\\\"\n" +
		"O=ООО \"Сертум-Про\",CN=ООО \"Сертум-Про\"\n" +
		"STREET=ул. Малопрудная\\, стр. 5\\, офис 715"

	matrix, err := imp.createOperationMatrix(map[string]string{}, map[string]string{
		"skip-cert-issuers": value,
	})
	if err != nil {
		t.Fatal(err)
	}

	operations := matrix.GetOperations()
	if len(operations) != 1 {
		t.Fatalf("expected one operation, got %d", len(operations))
	}

	decoded, err := base64.StdEncoding.DecodeString(operations[0].GetValue())
	if err != nil {
		t.Fatalf("operation value is not valid base64: %v", err)
	}

	if string(decoded) != value {
		t.Fatalf("quoted value changed\nexpected: %q\nactual:   %q", value, string(decoded))
	}
}

func TestFormatDryRunSummary(t *testing.T) {
	matrix := entities.NewOperationsMatrix()
	matrix.AddInsert(entities.Entry{KVPath: "new"})
	matrix.AddUpdate(entities.Entry{KVPath: "changed"})
	matrix.AddDelete(entities.Entry{KVPath: "old"})

	got := formatDryRunSummary(matrix)
	want := "DRYRUN: 3 operations: 1 inserts, 1 updates, 1 deletes"
	if got != want {
		t.Fatalf("formatDryRunSummary() = %q, want %q", got, want)
	}
}

func TestFormatDryRunSummaryNoOperations(t *testing.T) {
	got := formatDryRunSummary(entities.NewOperationsMatrix())
	want := "DRYRUN: no operations to process, all synced"
	if got != want {
		t.Fatalf("formatDryRunSummary() = %q, want %q", got, want)
	}
}

func TestWriteDryRunSummary(t *testing.T) {
	matrix := entities.NewOperationsMatrix()
	matrix.AddInsert(entities.Entry{KVPath: "new"})

	var buffer bytes.Buffer
	if err := writeDryRunSummary(&buffer, matrix); err != nil {
		t.Fatalf("writeDryRunSummary() error = %v", err)
	}

	want := "DRYRUN: 1 operations: 1 inserts, 0 updates, 0 deletes\n"
	if buffer.String() != want {
		t.Fatalf("writeDryRunSummary() = %q, want %q", buffer.String(), want)
	}
}

func TestWriteDryRunJSON(t *testing.T) {
	matrix := entities.NewOperationsMatrix()
	matrix.AddInsert(entities.Entry{KVPath: "new", Value: "secret"})
	matrix.AddDelete(entities.Entry{KVPath: "old"})

	var buffer bytes.Buffer
	if err := writeDryRunJSON(&buffer, matrix); err != nil {
		t.Fatalf("writeDryRunJSON() error = %v", err)
	}

	var got struct {
		Total      int `json:"total"`
		Inserts    int `json:"inserts"`
		Updates    int `json:"updates"`
		Deletes    int `json:"deletes"`
		Operations []struct {
			Type  string `json:"type"`
			Verb  string `json:"verb"`
			Path  string `json:"path"`
			Value string `json:"value"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(buffer.Bytes(), &got); err != nil {
		t.Fatalf("dry-run output is not valid JSON: %v", err)
	}

	if got.Total != 2 || got.Inserts != 1 || got.Updates != 0 || got.Deletes != 1 {
		t.Fatalf("unexpected dry-run counters: %#v", got)
	}
	if got.Operations[0].Type != entities.OperationInsert || got.Operations[0].Verb != "set" || got.Operations[0].Path != "new" {
		t.Fatalf("unexpected first operation: %#v", got.Operations[0])
	}
	if got.Operations[0].Value != "" {
		t.Fatalf("dry-run JSON should not expose operation values, got %q", got.Operations[0].Value)
	}
}
