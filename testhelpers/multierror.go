package testhelpers

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"testing"
)

func CheckOneMultiError(t *testing.T, err error, validate string) {
	if err == nil {
		t.Fail()
	}
	if merr, ok := err.(*multierror.Error); ok {
		if len(merr.Errors) != 1 {
			t.Log(fmt.Sprintf("%v not 1 errors", len(merr.Errors)))
			for i, err := range merr.Errors {
				t.Log(fmt.Sprintf("Error %v is %s", i, err.Error()))
			}
			t.Fail()
			return
		}
		if merr.Errors[0].Error() != validate {
			t.Log("'" + merr.Errors[0].Error() + "' not '" + validate + "'")
			t.Fail()
		}
	} else {
		t.Log("Not multierror")
		t.Log(err)
		t.Fail()
	}
}
