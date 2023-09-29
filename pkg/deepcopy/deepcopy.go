package deepcopy

func DeepCopyMap(original map[string]interface{}) map[string]interface{} {
  copy := make(map[string]interface{})
  for key, value := range original {
    if subMap, ok := value.(map[string]interface{}); ok {
      copy[key] = DeepCopyMap(subMap)
    } else {
      copy[key] = value
    }
  }
  return copy
}
