package config

import (
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// MustLoad 是一个泛型配置加载函数，发生错误时会panic
func MustLoad[T any](path string) (*koanf.Koanf, *T) {
	k := koanf.New(".")

	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		panic(err)
	}

	var config T
	if err := k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		panic(err)
	}
	return k, &config
}
