package config

import (
	"fmt"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	path := "../simple.toml"
	cnf, err := Load(path)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(cnf.KafkaCnf)
}
