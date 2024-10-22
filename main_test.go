package main

import (
	"testing"
)

func TestParsePeerNameOk(t *testing.T) {
	unit, cred, err := parsePeerName("@f4b4692a71d9438e/unit/age-creds-test.service/foo")
	if err != nil {
		t.Error(err)
	}
	if unit != "age-creds-test.service" {
		t.Errorf("unit = %s; want age-creds-test.service", unit)
	}
	if cred != "foo" {
		t.Errorf("cred = %s; want foo", cred)
	}
}

func TestParsePeerNameBlank(t *testing.T) {
	_, _, err := parsePeerName("")
	if err == nil {
		t.Error("expected parse error")
	}
}
