package exporter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

func (e *exporter) expandDocument(path string, document map[string]interface{}, localData map[string]string) error {
	return e.flattenMap(path, document, localData)
}

func (e *exporter) flattenMap(path string, document map[string]interface{}, localData map[string]string) error {
	keys := make([]string, 0, len(document))
	for key := range document {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := document[key]
		// Append key to path
		newPath := path + "/" + key

		switch typedValue := value.(type) {
		case map[string]interface{}:
			// we have an object, recurse casting the value
			if err := e.flattenMap(newPath, typedValue, localData); err != nil {
				return err
			}

		default:
			serialized, err := serializeValue(typedValue)
			if err != nil {
				return err
			}

			piece := e.createPiece(newPath, serialized)
			localData[piece.KVPath] = piece.Value
		}
	}

	return nil
}

func serializeValue(value interface{}) (string, error) {
	switch typedValue := value.(type) {
	case string:
		return typedValue, nil
	case bool:
		return strconv.FormatBool(typedValue), nil
	case nil:
		return "null", nil
	case float64:
		return fmt.Sprint(typedValue), nil
	case float32:
		return strconv.FormatFloat(float64(typedValue), 'g', -1, 32), nil
	case int:
		return strconv.Itoa(typedValue), nil
	case int8:
		return strconv.FormatInt(int64(typedValue), 10), nil
	case int16:
		return strconv.FormatInt(int64(typedValue), 10), nil
	case int32:
		return strconv.FormatInt(int64(typedValue), 10), nil
	case int64:
		return strconv.FormatInt(typedValue, 10), nil
	case uint:
		return strconv.FormatUint(uint64(typedValue), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(typedValue), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(typedValue), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(typedValue), 10), nil
	case uint64:
		return strconv.FormatUint(typedValue, 10), nil
	case []interface{}:
		return serializeCollection(typedValue)
	default:
		return "", fmt.Errorf("unsupported value type %T", value)
	}
}

func serializeCollection(value interface{}) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("serialize collection as JSON: %w", err)
	}

	return string(data), nil
}
