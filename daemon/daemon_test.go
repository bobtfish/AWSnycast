package daemon

import (
	"testing"
)

type FakeMetadataFetcher struct {
    FAvailable bool
    Meta map[string]string
}

func (m FakeMetadataFetcher) Available() bool {
    return m.FAvailable
}

func (m FakeMetadataFetcher) GetMetadata(key string) (string, error) {
    return "", nil
}

func TestSetupUnavailable(t *testing.T) {
    d := Daemon{}
    fake := FakeMetadataFetcher{}
    d.MetadataFetcher = fake
    err := d.Setup()
    if err != nil {
        t.Fail()
    }
    if d.MetadataFetcher.Available() {
        t.Fail()
    }
}

func TestSetupAvailable(t *testing.T) {
    d := Daemon{}
    fake := FakeMetadataFetcher{
        FAvailable: true,
    }
    d.MetadataFetcher = fake
    err := d.Setup()
    if err != nil {
        t.Fail()
    }
    if !d.MetadataFetcher.Available() {
        t.Fail()
    }
}

