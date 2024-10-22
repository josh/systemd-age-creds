package main

import (
	"testing"
)

func TestParsePeerName(t *testing.T) {
	// TODO: Get real example of peer name
	unitName, credID, err := parsePeerName("@foo/unit/bar/baz")
	if err != nil {
		t.Error(err)
	}
	if unitName != "bar" {
		t.Errorf("unitName = %s; want bar", unitName)
	}
	if credID != "baz" {
		t.Errorf("credID = %s; want baz", credID)
	}
}
