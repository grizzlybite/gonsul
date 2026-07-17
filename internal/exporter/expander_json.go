package exporter

import (
	"encoding/json"
	"fmt"

	"github.com/grizzlybite/gonsul/internal/util"
)

func (e *exporter) validateJSON(path string, data string) (map[string]interface{}, error) {
	var document interface{}

	err := json.Unmarshal([]byte(data), &document)

	// Decoded JSON ok?
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON file: %s with Message: %s", path, err.Error())
	}

	documentMap, ok := document.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error parsing JSON file: %s with Message: root document must be an object", path)
	}

	return documentMap, nil
}

func (e *exporter) expandJSON(path string, data string, localData map[string]string) error {
	documentMap, err := e.validateJSON(path, data)
	if err != nil {
		return util.NewGonsulError(err, util.ErrorFailedJsonDecode)
	}

	// Flatten and serialize the decoded JSON document.
	if err := e.expandDocument(path, documentMap, localData); err != nil {
		return util.NewGonsulError(err, util.ErrorFailedJsonEncode)
	}

	return nil
}
