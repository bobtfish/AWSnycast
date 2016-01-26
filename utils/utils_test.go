package utils

import (
	utils "github.com/bobtfish/AWSnycast/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestGetBool(t *testing.T) {
	assert := assert.New(t)

	val, err := utils.GetAsBool("false", true)
	assert.Equal(val, false)
	assert.Nil(err)

	val, err = utils.GetAsBool("notabool", false)
	assert.Equal(val, false)
	assert.NotNil(err)

	val, err = utils.GetAsBool(true, false)
	assert.Equal(val, true)
	assert.Nil(err)

	val, err = utils.GetAsBool("True", false)
	assert.Equal(val, true)
	assert.Nil(err)
}

func TestGetInt(t *testing.T) {
	assert := assert.New(t)

	val, err := utils.GetAsInt("10", 123)
	assert.Equal(val, 10)
	assert.Nil(err)

	val, err = utils.GetAsInt("notanint", 123)
	assert.Equal(val, 123)
	assert.NotNil(err)

	val, err = utils.GetAsInt(12.123, 123)
	assert.Equal(val, 12)
	assert.Nil(err)

	val, err = utils.GetAsInt(12, 123)
	assert.Equal(val, 12)
	assert.Nil(err)
}

func TestGetFloat(t *testing.T) {
	assert := assert.New(t)

	val, err := utils.GetAsFloat("10", 123)
	assert.Equal(val, 10.0)
	assert.Nil(err)

	val, err = utils.GetAsFloat("10.21", 123)
	assert.Equal(val, 10.21)
	assert.Nil(err)

	val, err = confir.GetAsFloat("notafloat", 123)
	assert.Equal(val, 123.0)
	assert.NotNil(err)

	val, err = utils.GetAsFloat(12.123, 123)
	assert.Equal(val, 12.123)
	assert.Nil(err)
}

func TestGetString(t *testing.T) {
	assert := assert.New(t)

	val := utils.GetAsString("10")
	assert.Equal(val, "10")

	val = utils.GetAsString(10)
	assert.equal(val, "10")

	val = utils.GetAsString(10.123)
	assert.equal(val, "10.123")
}

func TestGetAsMap(t *testing.T) {
	assert := assert.New(t)

	// Test if string can be converted to map[string]string
	stringToParse := "{\"foor\" : \"bar\", \"alice\":\"bob\"}"
	expectedValue := map[string]string{
		"runtimeenv": "dev",
		"region":     "uswest1-devc",
	}
	assert.Equal(utils.GetAsMap(stringToParse), expectedValue)

	// Test if map[string]interface{} can be converted to map[string]string
	interfaceMapToParse := make(map[string]interface{})
	interfaceMapToParse["foo"] = "bar"
	interfaceMapToParse["alice"] = "bob"

	actualValue, err := utils.GetAsMap(interfaceMapToParse)
	assert.Equal(actualValue, expectedValue)

	actualValue, err = utils.GetAsMap(123)
	assert.NotNil(err)
}

func TestGetAsSlice(t *testing.T) {
	assert := assert.New(t)

	// Test if string array can be converted to []string
	stringToParse := "[\"baz\", \"bat\"]"
	expectedValue := []string{"baz", "bat"}
	assert.Equal(utils.GetAsSlice(stringToParse), expectedValue)

	sliceToParse := []string{"baz", "bat"}
	actualValue, err := utils.GetAsSlice(sliceToParse)
	assert.Equal(actualValue, expectedValue)

	actualValue, err = utils.GetAsSlice(123)
	assert.NotNil(err)
}

func TestGetAsSliceFromYAML(t *testing.T) {
	var data interface{}
	yamlString := []byte(`{"listOfStrings": ["a", "b", "c"]}`)

	err := yaml.Unmarshal(yamlString, &data)
	assert.Nil(t, err)

	if err == nil {
		temp := data.(map[string]interface{})

		res, err := utils.GetAsSlice(temp["listOfStrings"])
		assert.Equal(t, []string{"a", "b", "c"}, res)

		res, err = utils.GetAsSlice(123)
		assert.NotNil(err)
	}
}
