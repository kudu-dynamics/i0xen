package config

// HCL allows for implicit array structures.
//
// ```
// foo {
//   a = 1
// }
//
// foo {
//   b = 2
// }
// ```
//
// By default, this would result in a structure like:
//
// ```
// foo = [
//   {"a": 1},
//   {"b": 2},
// ]
// ```
//
// This file overrides this behaviour and squashes lists of maps instead.
//
// https://stackoverflow.com/questions/48240461/unmarshal-hcl-to-struct-using-viper

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func GetHclOverride() viper.DecoderConfigOption {
	return viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			sliceOfMapsToMapHookFunc(),
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	)
}

// sliceOfMapsToMapHookFunc merges a slice of maps to a map
func sliceOfMapsToMapHookFunc() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if from.Kind() == reflect.Slice && from.Elem().Kind() == reflect.Map && (to.Kind() == reflect.Struct || to.Kind() == reflect.Map) {
			source, ok := data.([]map[string]interface{})
			if !ok {
				return data, nil
			}
			if len(source) == 0 {
				return data, nil
			}
			if len(source) == 1 {
				return source[0], nil
			}
			// flatten the slice into one map
			convert := make(map[string]interface{})
			for _, mapItem := range source {
				for key, value := range mapItem {
					convert[key] = value
				}
			}
			return convert, nil
		}
		return data, nil
	}
}
