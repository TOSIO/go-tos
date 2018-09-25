package configfile

import (
	"testing"
)

func TestReadConf(t *testing.T) {
	p, err := ReadConf("./config.toml")
	if err != nil {
		t.Logf("%v", err)
	}

	t.Logf("person %v", p.Urlstring)
}
