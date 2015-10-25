package instancemetadata

import (
	"errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	var mdf MetadataFetcher
	mdf = New(true)
	if mdf == nil {
		t.Fail()
	}
}

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
	return v, errors.New(fmt.Sprintf("Key %s unknown", key))
}

func getFakeMetadataFetcher(a bool) MetadataFetcher {
	fakeM := FakeMetadataFetcher{
		FAvailable: a,
	}
	fakeM.Meta = make(map[string]string)
	fakeM.Meta["placement/availability-zone"] = "us-west-1a"
	fakeM.Meta["instance-id"] = "i-1234"
	fakeM.Meta["mac"] = "06:1d:ea:6f:8c:6e"
	fakeM.Meta["network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id"] = "subnet-28b0e940"
	return fakeM
}

func TestGetSubnetIdMacFail(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "mac")
	_, err := getSubnetId(mdf)
	if err == nil {
		t.Fail()
	}
}

func TestGetSubnetIdMacFail2(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id")
	_, err := getSubnetId(mdf)
	if err == nil {
		t.Fail()
	}
}

func TestGetSubnetIdMacOk(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	val, err := getSubnetId(mdf)
	if err != nil {
		t.Fail()
	}
	if val != "subnet-28b0e940" {
		t.Fail()
	}
}

func TestFetchMetadataFail(t *testing.T) {
	_, err := FetchMetadata(getFakeMetadataFetcher(false))
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "No metadata service" {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestFetchMetadataAzFail(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "placement/availability-zone")
	_, err := FetchMetadata(mdf)
	if err == nil {
		t.Fail()
	}
}

func TestFetchMetadataInstanceFail(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "instance-id")
	_, err := FetchMetadata(mdf)
	if err == nil {
		t.Fail()
	}
}

func TestFetchMetadataMacFail(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "mac")
	_, err := FetchMetadata(mdf)
	if err == nil {
		t.Fail()
	}
}

func TestFetchMetadata(t *testing.T) {
	m, err := FetchMetadata(getFakeMetadataFetcher(true))
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if m.Instance != "i-1234" {
		t.Log(m.Instance)
		t.Fail()
	}
	if m.Subnet != "subnet-28b0e940" {
		t.Fail()
	}
	if m.AvailabilityZone != "us-west-1a" {
		t.Fail()
	}
	if m.Region != "us-west-1" {
		t.Fail()
	}
}
