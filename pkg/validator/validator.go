package validator

import (
	"reflect"

	"gopkg.in/yaml.v3"
)

// FIX THIS SHIT LOGIC LATER. Diff result based on removing lines from current state, compare with desired ant return transformed map.
// Which causes issue when for example kubectl edit resource was called & that causes changes isn't detected.
// My main goal at this point is to implement and save basic functionallity with less effort.

// ValidateYaml takes two YAML strings and returns true if they match according to the criteria defined in compareSpecs.
// It now also returns the processed maps.
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
	return reflect.DeepEqual(ModifiedCurrentMap, DesiredMap), DesiredMap, ModifiedCurrentMap // return maps
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
		case nil:
			delete(data, k)
		}
	}
}

// I was talking about this piece of crap:
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
