package main

import (
	"reflect"
	"sort"
)

func Changed(old, new []ApiInfo) bool {
	return !reflect.DeepEqual(normalizeAPIs(old), normalizeAPIs(new))
}

func normalizeAPIs(apis []ApiInfo) []ApiInfo {
	out := append([]ApiInfo(nil), apis...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		if out[i].ApiID != out[j].ApiID {
			return out[i].ApiID < out[j].ApiID
		}
		if out[i].Stage != out[j].Stage {
			return out[i].Stage < out[j].Stage
		}
		return out[i].Name < out[j].Name
	})
	return out
}
