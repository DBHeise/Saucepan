package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseExtra(t *testing.T) {
	extra := make([]extraparsing, 3)
	extra[0] = extraparsing{Name: "test1", Start: "a", End: "b"}
	extra[1] = extraparsing{Name: "test2", Start: "c", End: "d"}
	extra[2] = extraparsing{Name: "test3", Start: "e", End: "\r\n"}
	config = &configuration{ExtraParsing: extra}

	testObj := make(map[string]interface{})
	records := make([]string, 0)
	testValue := "a123b456c789d\r\ntesttesttest"

	parseExtra(&testObj, records, testValue)

	assert.Equal(t, "123", testObj["test1"])
	assert.Equal(t, "789", testObj["test2"])
	assert.Equal(t, "sttesttest", testObj["test3"])
}

func TestShouldIgnore(t *testing.T) {
	igList := make([]string, 4)
	igList[0] = "test"
	igList[1] = " "
	igList[2] = "♬"
	igList[3] = "Q"
	config = &configuration{IgnoreList: igList}

	assert.True(t, shouldIgnore("hi i am\ta test str♬ng"))
	assert.True(t, shouldIgnore("hi_i_am_a_test_string"))
	assert.True(t, shouldIgnore("hi_i_am_a_♬_string"))
	assert.False(t, shouldIgnore("another-string"))
	assert.True(t, shouldIgnore("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"))
}

func TestShouldIgnore_noStrings(t *testing.T) {
	igList := make([]string, 0)
	config = &configuration{IgnoreList: igList}

	assert.False(t, shouldIgnore("hi i am\ta test str♬ng"))
	assert.False(t, shouldIgnore("hi_i_am_a_test_string"))
	assert.False(t, shouldIgnore("hi_i_am_a_♬_string"))
	assert.False(t, shouldIgnore("another-string"))
	assert.False(t, shouldIgnore("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"))
}
