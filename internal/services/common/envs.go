package common

import (
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

func BindConfigEnvVars(v *viper.Viper, cfgType reflect.Type, prefix string) {
	bindEnvForType(v, cfgType, prefix, nil)
}

func bindEnvForType(v *viper.Viper, t reflect.Type, prefix string, path []string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}
		curPath := append(path, tag)
		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		switch ft.Kind() {
		case reflect.Struct:
			bindEnvForType(v, ft, prefix, curPath)
		default:
			envKey := prefix + strings.ToUpper(strings.Join(curPath, "_"))
			_ = v.BindEnv(strings.Join(curPath, "."), envKey)
		}
	}
}
