package utils

import (
	"errors"
	"gopkg.in/yaml.v2"
	"reflect"
	"strconv"
)

// GetAsBool parses a string to a bool or returns the bool if bool is passed in
func GetAsBool(value interface{}, defaultValue bool) (result bool, err error) {
	result = defaultValue

	switch value.(type) {
	case string:
		fromString, e := strconv.ParseBool(value.(string))
		if e != nil {
			err = errors.New("Failed to convert value to a bool. Falling back to default")
			result = defaultValue
		} else {
			result = fromString
		}
	case bool:
		result = value.(bool)
	}

	return
}

// GetAsFloat parses a string to a float or returns the float if float is passed in
func GetAsFloat(value interface{}, defaultValue float64) (result float64, err error) {
	result = defaultValue

	switch value.(type) {
	case string:
		fromString, e := strconv.ParseFloat(value.(string), 64)
		if e != nil {
			err = errors.New("Failed to convert value to a bool. Falling back to default")
			result = defaultValue
		} else {
			result = fromString
		}
	case float64:
		result = value.(float64)
	}

	return
}

// GetAsInt parses a string/float to an int or returns the int if int is passed in
func GetAsInt(value interface{}, defaultValue int) (result int, err error) {
	result = defaultValue

	switch value.(type) {
	case string:
		fromString, e := strconv.ParseInt(value.(string), 10, 64)
		if e == nil {
			result = int(fromString)
		} else {
			err = errors.New("Failed to convert value to an int")
		}
	case int:
		result = value.(int)
	case int32:
		result = int(value.(int32))
	case int64:
		result = int(value.(int64))
	case float64:
		result = int(value.(float64))
	}

	return
}

// GetAsString parses a int/float to a string or returns the string if string is passed in
func GetAsString(value interface{}) (result string) {
	result = ""

	switch value.(type) {
	case string:
		result = value.(string)
	case int:
		result = strconv.Itoa(value.(int))
	case float64:
		result = strconv.FormatFloat(value.(float64), 'G', -1, 64)
	}

	return
}

// GetAsMap parses a string to a map[string]string
func GetAsMap(value interface{}) (result map[string]string, err error) {
	result = make(map[string]string)

	switch value.(type) {
	case string:
		e := yaml.Unmarshal([]byte(value.(string)), &result)
		if e != nil {
			err = errors.New("Failed to convert value to a map")
		}
	case map[string]interface{}:
		temp := value.(map[string]interface{})
		for k, v := range temp {
			if str, ok := v.(string); ok {
				result[k] = str
			} else {
				err = errors.New("Expected a string but got" + reflect.TypeOf(value).Name())
			}
		}
	case map[string]string:
		result = value.(map[string]string)
	default:
		err = errors.New("Expected a string but got" + reflect.TypeOf(value).Name())
	}

	return
}

// GetAsSlice : Parses a yaml array string to []string
func GetAsSlice(value interface{}) (result []string, err error) {
	result = []string{}

	switch realValue := value.(type) {
	case string:
		e := yaml.Unmarshal([]byte(realValue), &result)
		if e != nil {
			err = errors.New("Failed to convert string:" + realValue + "to a []string")
		}
	case []string:
		result = realValue
	case []interface{}:
		result = make([]string, len(realValue))
		for i, value := range realValue {
			result[i] = value.(string)
		}
	default:
		err = errors.New("Expected a string array but got" + reflect.TypeOf(realValue).Name() + ". Returning empty slice!")
	}

	return
}
