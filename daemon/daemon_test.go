package daemon

import (
	"errors"
	"fmt"
	"testing"
)

type FakeMetadataFetcher struct {
	FAvailable bool
	Meta       map[string]string
}

func (m FakeMetadataFetcher) Available() bool {
	return m.FAvailable
}

func (m FakeMetadataFetcher) GetMetadata(key string) (string, error) {
	v, ok := m.Meta[key]
	if ok {
		return v, nil
	}
	return v, errors.New(fmt.Sprintf("Key %s unknown"))
}

func getD(a bool) Daemon {
	d := Daemon{}
	fake := FakeMetadataFetcher{
		FAvailable: a,
	}
	fake.Meta = make(map[string]string)
	d.MetadataFetcher = fake
	return d
}

func TestSetupUnavailable(t *testing.T) {
	d := getD(false)
	err := d.Setup()
	if err != nil {
		t.Fail()
	}
	if d.MetadataFetcher.Available() {
		t.Fail()
	}
}

func TestSetupAvailable(t *testing.T) {
	d := getD(true)
	err := d.Setup()
	if err != nil {
		t.Fail()
	}
	if !d.MetadataFetcher.Available() {
		t.Fail()
	}
}

func TestGetSubnetIdMacFail(t *testing.T) {
	d := getD(true)
	_, err := d.GetSubnetId()
	if err == nil {
		t.Fail()
	}
}

func TestGetSubnetIdMacFail2(t *testing.T) {
	d := getD(true)
	d.MetadataFetcher.(FakeMetadataFetcher).Meta["mac"] = "06:1d:ea:6f:8c:6e"
	_, err := d.GetSubnetId()
	if err == nil {
		t.Fail()
	}
}

func TestGetSubnetIdMacOk(t *testing.T) {
	d := getD(true)
	d.MetadataFetcher.(FakeMetadataFetcher).Meta["mac"] = "06:1d:ea:6f:8c:6e"
	d.MetadataFetcher.(FakeMetadataFetcher).Meta["network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id"] = "subnet-28b0e940"
	val, err := d.GetSubnetId()
	if err != nil {
		t.Fail()
	}
	if val != "subnet-28b0e940" {
		t.Fail()
	}
}
