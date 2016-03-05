package utils

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestGetBool(t *testing.T) {
	val, err := GetAsBool("false", true)
	assert.Equal(t, val, false)
	assert.Nil(t, err)

	val, err = GetAsBool("notabool", false)
	assert.Equal(t, val, false)
	assert.NotNil(t, err)

	val, err = GetAsBool(true, false)
	assert.Equal(t, val, true)
	assert.Nil(t, err)

	val, err = GetAsBool("True", false)
	assert.Equal(t, val, true)
	assert.Nil(t, err)
}

func TestGetInt(t *testing.T) {
	val, err := GetAsInt("10", 123)
	assert.Equal(t, val, 10)
	assert.Nil(t, err)

	val, err = GetAsInt("notanint", 123)
	assert.Equal(t, val, 123)
	assert.NotNil(t, err)

	val, err = GetAsInt(12.123, 123)
	assert.Equal(t, val, 12)
	assert.Nil(t, err)

	val, err = GetAsInt(12, 123)
	assert.Equal(t, val, 12)
	assert.Nil(t, err)

	var intThirtyTwo int32 = 12
	val, err = GetAsInt(intThirtyTwo, 123)
	assert.Equal(t, val, 12)
	assert.Nil(t, err)

	var intSixtyFour int64 = 12
	val, err = GetAsInt(intSixtyFour, 123)
	assert.Equal(t, val, 12)
	assert.Nil(t, err)
}

func TestGetFloat(t *testing.T) {
	val, err := GetAsFloat("10", 123)
	assert.Equal(t, val, 10.0)
	assert.Nil(t, err)

	val, err = GetAsFloat("10.21", 123)
	assert.Equal(t, val, 10.21)
	assert.Nil(t, err)

	val, err = GetAsFloat("notafloat", 123)
	assert.Equal(t, val, 123.0)
	assert.NotNil(t, err)

	val, err = GetAsFloat(12.123, 123)
	assert.Equal(t, val, 12.123)
	assert.Nil(t, err)
}

func TestGetString(t *testing.T) {
	val := GetAsString("10")
	assert.Equal(t, val, "10")

	val = GetAsString(10)
	assert.Equal(t, val, "10")

	val = GetAsString(10.123)
	assert.Equal(t, val, "10.123")
}

func TestGetAsMap(t *testing.T) {
	// Test if string can be converted to map[string]string
	stringToParse := "{\"foo\" : \"bar\", \"alice\":\"bob\"}"
	expectedValue := map[string]string{
		"foo":   "bar",
		"alice": "bob",
	}
	actualValue, err := GetAsMap(stringToParse)
	assert.Nil(t, err)
	assert.Equal(t, actualValue, expectedValue)

	// Test if map[string]interface{} can be converted to map[string]string
	interfaceMapToParse := make(map[string]interface{})
	interfaceMapToParse["foo"] = "bar"
	interfaceMapToParse["alice"] = "bob"

	actualValue, err = GetAsMap(interfaceMapToParse)
	assert.Nil(t, err)
	assert.Equal(t, actualValue, expectedValue)

	_, err = GetAsMap(123)
	assert.NotNil(t, err)

	stringMap := make(map[string]string)
	stringMap["foo"] = "bar"
	stringMap["alice"] = "bob"
	actualValue, err = GetAsMap(stringMap)
	assert.Nil(t, err)
	assert.Equal(t, actualValue, expectedValue)

	_, err = GetAsMap("{\"foo\" : \"bar\", \"alice\":\"bob\"")
	assert.NotNil(t, err)
}

func TestGetAsSlice(t *testing.T) {
	// Test if string array can be converted to []string
	stringToParse := "[\"baz\", \"bat\"]"
	expectedValue := []string{"baz", "bat"}
	actualValue, err := GetAsSlice(stringToParse)
	assert.Equal(t, actualValue, expectedValue)

	sliceToParse := []string{"baz", "bat"}
	actualValue, err = GetAsSlice(sliceToParse)
	assert.Equal(t, actualValue, expectedValue)

	actualValue, err = GetAsSlice(123)
	assert.NotNil(t, err)
}

func TestGetAsSliceFromYAML(t *testing.T) {
	var data map[string]interface{}
	yamlString := []byte(`{"listOfStrings": ["a", "b", "c", 5]}`)

	err := yaml.Unmarshal(yamlString, &data)
	assert.Nil(t, err)

	if err == nil {
		temp := data

		res, err := GetAsSlice(temp["listOfStrings"])
		assert.Equal(t, []string{"a", "b", "c", "5"}, res)

		res, err = GetAsSlice(123)
		assert.NotNil(t, err)
	}
}

func TestGetAsSliceBadYaml(t *testing.T) {
	// Test if string array can be converted to []string
	stringToParse := "[\"baz, \"bat\"]"
	_, err := GetAsSlice(stringToParse)
	assert.NotNil(t, err)
}
