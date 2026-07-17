package importer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grizzlybite/gonsul/internal/config"
	"github.com/grizzlybite/gonsul/internal/entities"
	"github.com/grizzlybite/gonsul/internal/util"
	"github.com/grizzlybite/gonsul/tests/mocks"
)

func TestImporterStartWritesTransactionToConsulAPI(t *testing.T) {
	var gotTransactions []entities.ConsulTxn

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && strings.TrimSuffix(request.URL.Path, "/") == "/v1/kv/base" && request.URL.Query().Get("recurse") == "true":
			response.WriteHeader(http.StatusNotFound)
		case request.Method == http.MethodPut && request.URL.Path == "/v1/txn":
			if err := json.NewDecoder(request.Body).Decode(&gotTransactions); err != nil {
				t.Errorf("decode transaction payload: %v", err)
				response.WriteHeader(http.StatusBadRequest)
				return
			}
			response.WriteHeader(http.StatusOK)
			_, _ = response.Write([]byte("true"))
		default:
			t.Errorf("unexpected Consul API request: %s %s", request.Method, request.URL.String())
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &mocks.IConfig{}
	cfg.On("GetConsulBasePath").Return("base")
	cfg.On("GetConsulURL").Return(server.URL)
	cfg.On("GetConsulACL").Return("")
	cfg.On("DoSecrets").Return(false)
	cfg.On("AllowDeletes").Return("true")
	cfg.On("GetStrategy").Return(config.StrategyOnce)
	cfg.On("WorkingChan").Return(make(chan bool, 1))

	imp := NewImporter(cfg, util.NewLogger(0), server.Client())

	err := imp.Start(context.Background(), map[string]string{
		"base/app/config": "value with \"quotes\"",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(gotTransactions) != 1 {
		t.Fatalf("expected one Consul transaction, got %d", len(gotTransactions))
	}

	txn := gotTransactions[0].KV
	if txn.Verb == nil || *txn.Verb != "set" {
		t.Fatalf("unexpected transaction verb: %#v", txn.Verb)
	}
	if txn.Key == nil || *txn.Key != "base/app/config" {
		t.Fatalf("unexpected transaction key: %#v", txn.Key)
	}
	if txn.Value == nil {
		t.Fatal("expected transaction value")
	}

	decoded, err := base64.StdEncoding.DecodeString(*txn.Value)
	if err != nil {
		t.Fatalf("transaction value is not base64: %v", err)
	}
	if string(decoded) != "value with \"quotes\"" {
		t.Fatalf("unexpected transaction value: %q", string(decoded))
	}

	cfg.AssertExpectations(t)
}
