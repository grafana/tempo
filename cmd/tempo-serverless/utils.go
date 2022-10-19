package serverless

import (
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

func bindEnvs(v *viper.Viper, in interface{}) error {
	return bindEnvsFromValue(v, reflect.ValueOf(in))
}

func bindEnvsFromValue(v *viper.Viper, in reflect.Value, path ...string) error {
	if !in.IsValid() || in.Kind() == reflect.Ptr && in.IsNil() {
		return nil
	}
	switch in.Kind() {
	case reflect.Ptr:
		return bindEnvsFromValue(v, in.Elem(), path...)
	case reflect.Struct:
		st := in.Type()
		n := st.NumField()
		for i := 0; i != n; i++ {
			field := st.Field(i)
			tag := field.Tag.Get("yaml")
			if tag != "" {
				fields := strings.Split(tag, ",")
				name := fields[0]
				path := append(path, name)
				if err := bindEnvsFromValue(v, in.Field(i), path...); err != nil {
					return err
				}
			}
		}
	default:
		err := v.BindEnv(strings.Join(path, "."))
		if err != nil {
			return err
		}
	}
	return nil
}
