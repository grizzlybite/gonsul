package importer

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grizzlybite/gonsul/internal/config"
	"github.com/grizzlybite/gonsul/internal/entities"
	"github.com/grizzlybite/gonsul/internal/util"
)

// IImporter ...
type IImporter interface {
	Start(ctx context.Context, localData map[string]string) error
}

// importer ...
type importer struct {
	config config.IConfig
	logger util.ILogger
	client *http.Client
}

// NewImporter
func NewImporter(config config.IConfig, logger util.ILogger, client *http.Client) IImporter {
	return &importer{config: config, logger: logger, client: client}
}

// Start ...
func (i *importer) Start(ctx context.Context, localData map[string]string) error {

	// Create some local variables
	var ops entities.OperationMatrix
	var liveData map[string]string

	// Populate our Consul live data
	liveData, err := i.createLiveData(ctx)
	if err != nil {
		return err
	}

	// Create our operations Matrix
	ops, err = i.createOperationMatrix(liveData, localData)
	if err != nil {
		return err
	}

	// Check if it's a dry run
	if i.config.GetStrategy() == config.StrategyDry {
		return i.printDryRun(ops)
	}

	// Print operation table
	if err := i.printOperations(ops, entities.OperationAll); err != nil {
		return err
	}

	// Process our operations matrix
	if err := i.processOperations(ctx, ops); err != nil {
		return err
	}

	// Print result summary
	i.logger.PrintInfo(fmt.Sprintf("Finished: %d Inserts, %d Updates %d Deletes", ops.GetTotalInserts(), ops.GetTotalUpdates(), ops.GetTotalDeletes()))
	return nil
}

func (i *importer) processOperations(ctx context.Context, matrix entities.OperationMatrix) error {
	// Did we got any deletes and are we allowed to delete them?
	if i.config.AllowDeletes() == "false" && matrix.HasDeletes() {
		// We're not supposed to trigger Consul deletes, output report and exit with error
		i.logger.PrintError("We're stopping as there are deletes and Gonsul is running without delete permission")
		i.logger.PrintError("Below is all the Consul KV paths that would be deleted")

		// Print matrix (or set in logger messages if in hook mode) and exit
		if i.config.GetStrategy() == config.StrategyHook {
			i.setDeletesToLogger(matrix)
		} else {
			if err := i.printOperations(matrix, entities.OperationDelete); err != nil {
				return err
			}
		}
		return util.NewGonsulError(fmt.Errorf("deletes are not allowed"), util.ErrorDeleteNotAllowed)
	}

	// Initialize the batch counter
	batch := 1

	var transactions []entities.ConsulTxn
	var newTransactions []entities.ConsulTxn

	// Fill our channel to indicate a non interruptible work (It stops here if interruption in progress)
	i.config.WorkingChan() <- true

	// Loop each operation
	for _, op := range matrix.GetOperations() {
		// We need to get the values to use pointers for our structure
		// so we can clearly identify nil values, as in https://willnorris.com/2014/05/go-rest-apis-and-pointers
		verb := op.GetVerb()
		path := op.GetPath()

		var TxnKV entities.ConsulTxnKV

		if op.GetType() == entities.OperationDelete {
			TxnKV = entities.ConsulTxnKV{Verb: &verb, Key: &path}
		} else {
			val := op.GetValue()
			TxnKV = entities.ConsulTxnKV{Verb: &verb, Key: &path, Value: &val}
		}

		// Add the next transaction and check payload length.
		newTransactions = transactions
		newTransactions = append(transactions, entities.ConsulTxn{KV: TxnKV})
		newPayloadSize, err := i.getTransactionsPayloadSize(&newTransactions)
		if err != nil {
			return err
		}

		if newPayloadSize > maximumPayloadSize || len(transactions) == consulTxnLimit {
			if err := i.processConsulTransaction(ctx, transactions, batch); err != nil {
				return err
			}
			transactions = []entities.ConsulTxn{}

			batch++
		}

		transactions = append(transactions, entities.ConsulTxn{KV: TxnKV})
	}

	// Do we have transactions to process
	if len(transactions) > 0 {
		if err := i.processConsulTransaction(ctx, transactions, batch); err != nil {
			return err
		}
	}

	// Consume our channel, to re-allow application interruption
	<-i.config.WorkingChan()
	return nil
}
