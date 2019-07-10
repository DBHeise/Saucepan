package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
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

func TestBadCSVFiles(t *testing.T) {
	//Generate test config
	config = createDefaultConfig()
	fldr1, err := ioutil.TempDir(os.TempDir(), "saucepan_input_")
	if err != nil {
		t.Errorf("Could not create temporary folder: %s", err)
	}
	fldr2, err := ioutil.TempDir(os.TempDir(), "saucepan_output_")
	if err != nil {
		t.Errorf("Could not create temporary folder: %s", err)
	}

	config.WatchFolder = fldr1
	config.DoneFolder = fldr2
	config.CSVOptions.FirstRowHeader = true
	config.CSVOptions.CaptureColumn = 99999
	config.CyberSaucier.Enabled = false
	config.ElasticSearch.UseSimpleClient = true
	config.ElasticSearch.Enabled = false

	//Copy test files
	baseInFolder, err := filepath.Abs(".")
	if err != nil {
		t.Errorf("Unable to resolve current path: %s", err)
	}
	testFolder := filepath.Join(baseInFolder, "testfiles")
	err = copy.Copy(testFolder, config.WatchFolder)
	if err != nil {
		t.Errorf("Unable to copy test files to input folder: %s", err)
	}

	//Get test files
	testfiles, err := ioutil.ReadDir(config.WatchFolder)
	if err != nil {
		t.Errorf("Unable to read Watch Folder")
	}

	//Process each file
	for _, tfile := range testfiles {
		filename := tfile.Name()
		fullFilePath := filepath.Join(config.WatchFolder, filename)

		fileHandler(fullFilePath)

		expectedOutputFile := filepath.Join(config.DoneFolder, filename)
		if _, err := os.Stat(expectedOutputFile); os.IsNotExist(err) {
			assert.Fail(t, "Processed file not in done folder")
		}

	}

	//Cleanup
	os.RemoveAll(config.WatchFolder)
	os.RemoveAll(config.DoneFolder)
}
