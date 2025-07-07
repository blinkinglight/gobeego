package utils

import "encoding/json"

type M = map[string]any

func ToStructpbMap(v any) M {
	b, _ := json.Marshal(v)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m
}

func FromStructpbMap(m M, v any) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
