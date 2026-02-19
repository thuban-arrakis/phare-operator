package validator

import (
	"reflect"

	"gopkg.in/yaml.v3"
)

// This logic compares the desired YAML with a filtered version of the current YAML.
// It removes fields from current data that do not exist in desired data.
// This is simple, but it can hide some drift from manual edits.

// ValidateYaml compares two YAML strings and returns:
// 1) whether they match after filtering
// 2) the processed desired map
// 3) the processed current map
func ValidateYaml(desiredYaml, currentYaml string) (bool, map[string]interface{}, map[string]interface{}) {
	var DesiredMap, CurrentMap map[string]interface{}
	err := yaml.Unmarshal([]byte(desiredYaml), &DesiredMap)
	if err != nil {
		return false, map[string]interface{}{}, map[string]interface{}{}
	}

	err = yaml.Unmarshal([]byte(currentYaml), &CurrentMap)
	if err != nil {
		return false, DesiredMap, map[string]interface{}{}
	}

	filterEmptyFields(DesiredMap)
	filterEmptyFields(CurrentMap)

	ModifiedCurrentMap := cleanMapBasedOnAnother(CurrentMap, DesiredMap)
	return reflect.DeepEqual(ModifiedCurrentMap, DesiredMap), DesiredMap, ModifiedCurrentMap
}

// filterEmptyFields recursively filters out keys with empty values like "", null, {}, 0.
func filterEmptyFields(data map[string]interface{}) {
	for k, v := range data {
		switch value := v.(type) {
		case map[string]interface{}:
			filterEmptyFields(value)
			if len(value) == 0 {
				delete(data, k)
			}
		case string:
			if value == "" {
				delete(data, k)
			}
		case int:
			if value == 0 {
				delete(data, k)
			}
		case float64:
			// encoding/yaml unmarshals all numbers as float64.
			if value == 0 {
				delete(data, k)
			}
		case nil:
			delete(data, k)
		}
	}
}

// cleanMapBasedOnAnother removes keys from baseMap that aren't present in referenceMap.
func cleanMapBasedOnAnother(baseMap, referenceMap map[string]interface{}) map[string]interface{} {
	for k, v := range baseMap {
		refVal, exists := referenceMap[k]

		// If the key doesn't exist in the reference map, delete it from the base map
		if !exists {
			delete(baseMap, k)
			continue
		}

		// If both maps have this key and its value is another map, then process recursively
		if baseSubMap, ok := v.(map[string]interface{}); ok {
			if refSubMap, ok := refVal.(map[string]interface{}); ok {
				cleanMapBasedOnAnother(baseSubMap, refSubMap)

				// If after cleaning, the sub-map in base is empty but not in reference, delete it
				if len(baseSubMap) == 0 && len(refSubMap) > 0 {
					delete(baseMap, k)
				}
			} else {
				delete(baseMap, k)
			}
		} else if baseList, ok := v.([]interface{}); ok {
			if refList, ok := refVal.([]interface{}); ok {
				for i := range baseList {
					if i < len(refList) {
						if baseListItemMap, ok := baseList[i].(map[string]interface{}); ok {
							if refListItemMap, ok := refList[i].(map[string]interface{}); ok {
								cleanMapBasedOnAnother(baseListItemMap, refListItemMap)
							}
						}
					}
				}
			}
		}
	}
	return baseMap
}

func PrintMap(data map[string]interface{}) string {
	bytes, err := yaml.Marshal(data)
	if err != nil {
		return ""
	}
	return string(bytes)
}
