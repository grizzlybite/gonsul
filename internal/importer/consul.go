package importer

import (
	"strconv"

	"github.com/grizzlybite/gonsul/internal/entities"
	"github.com/grizzlybite/gonsul/internal/util"

	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const consulTxnLimit = 64
const maximumPayloadSize = 500000 // max size is actually 512kb

// processConsulTransaction ...
func (i *importer) processConsulTransaction(ctx context.Context, transactions []entities.ConsulTxn, batchNumber int) error {
	batch := strconv.Itoa(batchNumber)

	// Encode our transaction into a JSON payload
	jsonPayload, err := json.Marshal(transactions)
	if err != nil {
		return util.NewGonsulError(fmt.Errorf("Marshal: %w in Batch %s", err, batch), util.ErrorFailedJsonEncode)
	}

	// Create our URL
	consulUrl := i.config.GetConsulURL() + "/v1/txn"

	// build our request
	i.logger.PrintDebug("CONSUL: creating PUT request for Batch " + batch)
	req, err := http.NewRequestWithContext(ctx, "PUT", consulUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return util.NewGonsulError(fmt.Errorf("NewRequestPUT: %w in Batch %s", err, batch), util.ErrorFailedConsulConnection)
	}

	// Set ACL token
	if i.config.GetConsulACL() != "" {
		req.Header.Set("X-Consul-Token", i.config.GetConsulACL())
	}

	// Send the request via a client
	// Do sends an HTTP request and
	// returns an HTTP response
	i.logger.PrintDebug("CONSUL: calling PUT request for Batch " + batch)
	resp, err := i.client.Do(req)
	if err != nil {
		message := util.RedactSensitive(err.Error(), consulUrl, i.config.GetConsulACL())
		return util.NewGonsulError(fmt.Errorf("DoPUT: %s for Batch %s", message, batch), util.ErrorFailedConsulConnection)
	}

	// Clean response after function ends
	defer func() {
		if err := resp.Body.Close(); err != nil {
			i.logger.PrintError("Could not close Consul http body")
		}
	}()

	// Read the response body
	i.logger.PrintDebug("CONSUL: reading PUT response from Batch " + batch)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return util.NewGonsulError(fmt.Errorf("ReadPutResponse: %w in Batch %s", err, batch), util.ErrorFailedReadingResponse)
	}

	// Cast response to string
	bodyString := string(bodyBytes)

	if resp.StatusCode != 200 {
		return util.NewGonsulError(
			fmt.Errorf("TransactionError: consul returned %s with %d response bytes in Batch %s", resp.Status, len(bodyString), batch),
			util.ErrorFailedConsulTxn,
		)
	}

	// All good. Output some status for each transaction operation
	for _, txn := range transactions {
		i.logger.PrintInfo("Operation: " + *txn.KV.Verb + " Path: " + *txn.KV.Key + " Batch: " + batch)
	}

	return nil
}
