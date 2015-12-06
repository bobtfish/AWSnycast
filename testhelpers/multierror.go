package testhelpers

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"testing"
)

func CheckOneMultiError(t *testing.T, err error, validate string) {
	if assert.NotNil(t, err) {
		merr, ok := err.(*multierror.Error)
		if assert.Equal(t, ok, true, "Not multierror") {
			if len(merr.Errors) != 1 {
				t.Log(fmt.Sprintf("%v not 1 errors", len(merr.Errors)))
				for i, err := range merr.Errors {
					t.Log(fmt.Sprintf("Error %v is %s", i, err.Error()))
				}
				t.Fail()
				return
			}
			assert.Equal(t, merr.Errors[0].Error(), validate)
		}
	}
}
