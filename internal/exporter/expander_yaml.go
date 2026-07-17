package exporter

import (
	"fmt"

	"github.com/grizzlybite/gonsul/internal/util"
	"gopkg.in/yaml.v3"
)

func (e *exporter) validateYAML(path string, data string) (map[string]interface{}, error) {
	var document interface{}

	err := yaml.Unmarshal([]byte(data), &document)

	// Decoded YAML ok?
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML file: %s with Message: %s", path, err.Error())
	}

	documentMap, ok := document.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error parsing YAML file: %s with Message: root document must be an object", path)
	}

	return documentMap, nil
}

func (e *exporter) expandYAML(path string, data string, localData map[string]string) error {
	documentMap, err := e.validateYAML(path, data)
	if err != nil {
		return util.NewGonsulError(err, util.ErrorFailedJsonDecode)
	}

	// Flatten and serialize the decoded YAML document.
	if err := e.expandDocument(path, documentMap, localData); err != nil {
		return util.NewGonsulError(err, util.ErrorFailedJsonEncode)
	}

	return nil
}
