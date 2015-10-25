package daemon

import (
	"testing"
)

func TestgetSubnetIdMacFail(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "mac")
	_, err := getSubnetId(mdf)
	if err == nil {
		t.Fail()
	}
}

func TestgetSubnetIdMacFail2(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id")
	_, err := getSubnetId(mdf)
	if err == nil {
		t.Fail()
	}
}

func TestgetSubnetIdMacOk(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	val, err := getSubnetId(mdf)
	if err != nil {
		t.Fail()
	}
	if val != "subnet-28b0e940" {
		t.Fail()
	}
}
