package importer

import (
	"context"
	"strings"

	"github.com/grizzlybite/gonsul/internal/config"
	"github.com/grizzlybite/gonsul/internal/entities"
	"github.com/grizzlybite/gonsul/internal/util"

	"github.com/cbroglie/mustache"
	"github.com/olekukonko/tablewriter"

	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
)

// createOperationMatrix ...
func (i *importer) createOperationMatrix(liveData map[string]string, localData map[string]string) (entities.OperationMatrix, error) {
	// Set local error variable
	var err error
	// Create our Operations array
	var operations = entities.NewOperationsMatrix()

	// Check for updates or inserts
	localKeys := make([]string, 0, len(localData))
	for localKey := range localData {
		localKeys = append(localKeys, localKey)
	}
	sort.Strings(localKeys)

	for _, localKey := range localKeys {
		localVal := localData[localKey]
		// Make sure we do not have an empty value (Consul KV will not have it)
		if localVal == "" {
			continue
		}

		// Shall we run secret replacement
		if i.config.DoSecrets() {
			localVal, err = mustache.Render(localVal, i.config.GetSecretsMap())
		}
		if err != nil {
			return operations, util.NewGonsulError(fmt.Errorf("MustacheRender: %w", err), util.ErrorFailedMustache)
		}

		// Base64 encode local value
		localValB64 := base64.StdEncoding.EncodeToString([]byte(localVal))

		// Does the current local KV key (path) exists in live?
		if liveVal, ok := liveData[localKey]; ok {
			// it does, is it different value?
			if localValB64 != liveVal {
				// Gentleman we have an update
				operations.AddUpdate(entities.Entry{KVPath: localKey, Value: localValB64})
			}
		} else {
			// Current key does not exist in live data, we have an insert
			operations.AddInsert(entities.Entry{KVPath: localKey, Value: localValB64})
		}
	}

	// Now check for deletes
	// Check for deletes
	liveKeys := make([]string, 0, len(liveData))
	for liveKey := range liveData {
		liveKeys = append(liveKeys, liveKey)
	}
	sort.Strings(liveKeys)

	for _, liveKey := range liveKeys {
		if _, ok := localData[liveKey]; !ok && i.config.AllowDeletes() != "skip" {
			// Not found in local - DELETE
			operations.AddDelete(entities.Entry{KVPath: liveKey, Value: ""})
		}
	}

	return operations, nil
}

// createLiveData ...
func (i *importer) createLiveData(ctx context.Context) (map[string]string, error) {
	// Create some local variables
	var liveData map[string]string

	// Create our URL
	consulBasePath := strings.TrimSuffix(i.config.GetConsulBasePath(), "/")
	fullUrl := path.Join("v1", "kv", consulBasePath)
	hostname := strings.TrimSuffix(i.config.GetConsulURL(), "/")
	consulUrl := hostname + "/" + fullUrl + "/?recurse=true"
	// build our request
	req, err := http.NewRequestWithContext(ctx, "GET", consulUrl, nil)
	if err != nil {
		return nil, util.NewGonsulError(fmt.Errorf("NewRequestGET: %w", err), util.ErrorFailedConsulConnection)
	}

	// Set ACL token (if given)
	if i.config.GetConsulACL() != "" {
		req.Header.Set("X-Consul-Token", i.config.GetConsulACL())
	}

	// Send the request via a client, Do sends an HTTP request and returns an HTTP response
	resp, err := i.client.Do(req)
	if err != nil {
		message := util.RedactSensitive(err.Error(), consulUrl, i.config.GetConsulACL())
		return nil, util.NewGonsulError(fmt.Errorf("DoGET: %s", message), util.ErrorFailedConsulConnection)
	}

	// Clean response after function ends
	defer func() {
		if err := resp.Body.Close(); err != nil {
			i.logger.PrintError("Could not close Consul http body")
		}
	}()

	// Invalid response, path is empty then, fresh import
	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		return nil, util.NewGonsulError(fmt.Errorf("Invalid response from consul: %s", resp.Status), util.ErrorFailedConsulConnection)
	}

	// Read response from HTTP Response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, util.NewGonsulError(fmt.Errorf("ReadGetResponse: %w", err), util.ErrorFailedReadingResponse)
	}
	// Create a structure for our response, basically an array of
	// Consul results because we're doing a recurse call
	var bodyStruct []entities.ConsulResult
	// Convert response to a string and then parse it to our struct
	bodyString := string(bodyBytes)
	err = json.Unmarshal([]byte(bodyString), &bodyStruct)
	if err != nil {
		return nil, util.NewGonsulError(fmt.Errorf("Unmarshal: %w", err), util.ErrorFailedJsonDecode)
	}

	// All good so far, instantiate our map
	liveData = map[string]string{}

	// Loop each entry on our Consul response
	for _, v := range bodyStruct {
		// Add to our map
		liveData[v.Key] = v.Value
	}

	return liveData, nil
}

// printOperations ...
func (i *importer) printOperations(matrix entities.OperationMatrix, printWhat string) error {
	// Add a new line before the table
	fmt.Println()
	// Let's make sure there are any operation
	if matrix.GetTotalOps() > 0 {
		// Instantiate our table and set table header
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("", "BATCH", "OP INDEX", "OPERATION NAME", "CONSUL VERB", "PATH")

		// Initialize the batch counter
		batch := 1
		opIndex := 0

		var transactions []entities.ConsulTxn
		var newTransactions []entities.ConsulTxn

		// Loop each operation and add to table
		for _, op := range matrix.GetOperations() {

			if printWhat == entities.OperationAll || printWhat == op.GetType() {
				var TxnKV entities.ConsulTxnKV
				var warning string

				// Generate the actual payload to calculate its length.
				verb := op.GetVerb()
				path := op.GetPath()
				if op.GetType() == entities.OperationDelete {
					warning = "!!"
					TxnKV = entities.ConsulTxnKV{Verb: &verb, Key: &path}
				} else {
					warning = ""
					val := op.GetValue()
					TxnKV = entities.ConsulTxnKV{Verb: &verb, Key: &path, Value: &val}
				}

				// Add the next transaction and check payload length.
				newTransactions = transactions
				newTransactions = append(newTransactions, entities.ConsulTxn{KV: TxnKV})
				newPayloadSize, err := i.getTransactionsPayloadSize(&newTransactions)
				if err != nil {
					return err
				}

				// If the next transaction brings us over the maximum payload size,
				// or the maximum transaction per batch limit is reached, start a new batch
				if newPayloadSize > maximumPayloadSize || len(transactions) == consulTxnLimit {
					// reset transactions and add the next transaction
					transactions = []entities.ConsulTxn{}
					// start a new batch counter
					opIndex = 0
					batch++
				}

				transactions = append(transactions, entities.ConsulTxn{KV: TxnKV})

				if err := table.Append([]string{warning, strconv.Itoa(batch), strconv.Itoa(opIndex), op.GetType(), op.GetVerb(), op.GetPath()}); err != nil {
					return util.NewGonsulError(fmt.Errorf("render operations table row: %w", err), util.ErrorFailedJsonEncode)
				}

				opIndex++
			}
		}
		// Outputs ASCII table
		if err := table.Render(); err != nil {
			return util.NewGonsulError(fmt.Errorf("render operations table: %w", err), util.ErrorFailedJsonEncode)
		}
	} else {
		i.logger.PrintInfo("No operations to process, all synced")
	}

	return nil
}

func (i *importer) printDryRun(matrix entities.OperationMatrix) error {
	switch i.config.GetDryRunOutput() {
	case config.DryRunOutputTable:
		return i.printOperations(matrix, entities.OperationAll)
	case config.DryRunOutputJSON:
		return writeDryRunJSON(os.Stdout, matrix)
	}

	return writeDryRunSummary(os.Stdout, matrix)
}

func formatDryRunSummary(matrix entities.OperationMatrix) string {
	if matrix.GetTotalOps() == 0 {
		return "DRYRUN: no operations to process, all synced"
	}

	return fmt.Sprintf(
		"DRYRUN: %d operations: %d inserts, %d updates, %d deletes",
		matrix.GetTotalOps(),
		matrix.GetTotalInserts(),
		matrix.GetTotalUpdates(),
		matrix.GetTotalDeletes(),
	)
}

func writeDryRunSummary(writer io.Writer, matrix entities.OperationMatrix) error {
	if _, err := fmt.Fprintln(writer, formatDryRunSummary(matrix)); err != nil {
		return util.NewGonsulError(fmt.Errorf("write dry-run summary: %w", err), util.ErrorFailedJsonEncode)
	}

	return nil
}

type dryRunReport struct {
	Total      int               `json:"total"`
	Inserts    int               `json:"inserts"`
	Updates    int               `json:"updates"`
	Deletes    int               `json:"deletes"`
	Operations []dryRunOperation `json:"operations"`
}

type dryRunOperation struct {
	Type string `json:"type"`
	Verb string `json:"verb"`
	Path string `json:"path"`
}

func writeDryRunJSON(writer io.Writer, matrix entities.OperationMatrix) error {
	report := dryRunReport{
		Total:      matrix.GetTotalOps(),
		Inserts:    matrix.GetTotalInserts(),
		Updates:    matrix.GetTotalUpdates(),
		Deletes:    matrix.GetTotalDeletes(),
		Operations: make([]dryRunOperation, 0, matrix.GetTotalOps()),
	}

	for _, op := range matrix.GetOperations() {
		report.Operations = append(report.Operations, dryRunOperation{
			Type: op.GetType(),
			Verb: op.GetVerb(),
			Path: op.GetPath(),
		})
	}

	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(report); err != nil {
		return util.NewGonsulError(fmt.Errorf("encode dry-run JSON: %w", err), util.ErrorFailedJsonEncode)
	}

	return nil
}

// setDeletesToLogger ...
func (i *importer) setDeletesToLogger(matrix entities.OperationMatrix) {
	// Let's make sure there are any operation
	if matrix.GetTotalOps() > 0 {
		// Loop each operation and add to table
		for _, op := range matrix.GetOperations() {
			if op.GetType() == entities.OperationDelete {
				i.logger.AddMessage(op.GetPath())
			}
		}
	}
}

// Get the payload size for a slice of transactions
func (i *importer) getTransactionsPayloadSize(transactions *[]entities.ConsulTxn) (int, error) {
	payload, err := json.Marshal(&transactions)
	if err != nil {
		return 0, util.NewGonsulError(fmt.Errorf("Marshal: %w", err), util.ErrorFailedJsonEncode)
	}

	return len(string(payload)), nil
}
