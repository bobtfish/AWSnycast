package instancemetadata

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNew(t *testing.T) {
	var mdf MetadataFetcher
	mdf = New(true)
	assert.NotNil(t, mdf)
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
	fakeM.Meta["local-ipv4"] = "127.0.0.1"
	fakeM.Meta["network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id"] = "subnet-28b0e940"
	return fakeM
}

func TestGetSubnetIdMacFail(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "mac")
	_, err := getSubnetId(mdf)
	assert.NotNil(t, err)
}

func TestGetSubnetIdMacFail2(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id")
	_, err := getSubnetId(mdf)
	assert.NotNil(t, err)
}

func TestGetSubnetIdMacOk(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	val, err := getSubnetId(mdf)
	if assert.Nil(t, err) {
		assert.Equal(t, val, "subnet-28b0e940")
	}
}

func TestFetchMetadataFail(t *testing.T) {
	_, err := FetchMetadata(getFakeMetadataFetcher(false))
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "No metadata service")
	}
}

func TestFetchMetadataAzFail(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "placement/availability-zone")
	_, err := FetchMetadata(mdf)
	assert.NotNil(t, err)
}

func TestFetchMetadataInstanceFail(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "instance-id")
	_, err := FetchMetadata(mdf)
	assert.NotNil(t, err)
}

func TestFetchMetadataMacFail(t *testing.T) {
	mdf := getFakeMetadataFetcher(true)
	delete(mdf.(FakeMetadataFetcher).Meta, "mac")
	_, err := FetchMetadata(mdf)
	assert.NotNil(t, err)
}

func TestFetchMetadata(t *testing.T) {
	m, err := FetchMetadata(getFakeMetadataFetcher(true))
	assert.Nil(t, err)
	assert.Equal(t, m.Instance, "i-1234")
	assert.Equal(t, m.Subnet, "subnet-28b0e940")
	assert.Equal(t, m.AvailabilityZone, "us-west-1a")
	assert.Equal(t, m.Region, "us-west-1")
}
