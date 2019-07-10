package main

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReadWriteConfig(t *testing.T) {
	configFile, err := ioutil.TempFile("", "testing_")
	if err != nil {
		t.Error(err)
	}

	fullFile := configFile.Name()
	expected := &configuration{
		WatchFolder:        "/data/test/folder/input",
		DoneFolder:         "T:\\data\\test\\folder\\output",
		MaxConcurrentFiles: 6,
		MoveAfterProcessed: true,
		SaveNoSauce:        false,
		NoSauceFile:        "asdf.csv",
		ParseErrorFile:     "lkjf.csv",
		WaitInterval:       17,
		CyberSaucier: cybersaucierConfig{
			URL: "http://127.0.0.1:7000",
		},
		IgnoreList: make([]string, 0),
		CSVOptions: csvconfig{
			FirstRowHeader: false,
			CaptureColumn:  0,
		},
		ElasticSearch: esconfig{
			URL:        "",
			IndexStart: "aetfha",
			DTMask:     "20060102",
			Type:       "jhrt",
			UserName:   "bob",
			Password:   "secretpasswordyo",
			QueueSize:  1,
		},
		ExtraParsing: make([]extraparsing, 0),
	}

	saveConfig(fullFile, expected)

	loadConfig(fullFile)

	actual := config

	assert.EqualValues(t, expected, actual)

	os.Remove(fullFile)
}

func TestEnvironmentOverride(t *testing.T) {
	configFile, err := ioutil.TempFile("", "testing_")
	if err != nil {
		t.Error(err)
	}

	fullFile := configFile.Name()
	expected := &configuration{
		WatchFolder:        "/data/test/folder/input",
		DoneFolder:         "T:\\data\\test\\folder\\output",
		MoveAfterProcessed: true,
		SaveNoSauce:        false,
		NoSauceFile:        "asdf.csv",
		ParseErrorFile:     "lkjf.csv",
		WaitInterval:       30,
		CyberSaucier: cybersaucierConfig{
			URL: "http://127.0.0.1:7000",
		},
		IgnoreList: make([]string, 0),
		CSVOptions: csvconfig{
			FirstRowHeader: false,
			CaptureColumn:  1,
		},
		ElasticSearch: esconfig{
			URL:        "",
			IndexStart: "aetfha",
			DTMask:     "20060102",
			Type:       "jhrt",
			QueueSize:  1,
		},
		ExtraParsing: make([]extraparsing, 0),
	}

	saveConfig(fullFile, expected)

	os.Setenv("SAUCE_WaitInterval", "13")
	os.Setenv("SAUCE_CSVOptions_CaptureColumn", "10")
	os.Setenv("SAUCE_CyberSaucier_Query", "?match=FooBar")
	os.Setenv("SAUCE_ElasticSearch_Password", "")

	loadConfig(fullFile)

	actual := config

	assert.EqualValues(t, 13, actual.WaitInterval)
	assert.EqualValues(t, 10, actual.CSVOptions.CaptureColumn)
	assert.EqualValues(t, "?match=FooBar", actual.CyberSaucier.Query)
	assert.EqualValues(t, "", actual.ElasticSearch.Password)
	os.Remove(fullFile)
}

func TestMacros(t *testing.T) {
	config := createDefaultConfig()
	config.NoSauceFile = "test_$date$.csv"
	config.ParseErrorFile = "parseError_$name$_$date$.csv"

	dt := time.Now().Format("2006-01-02_150405")
	assert.EqualValues(t, dt, config.doMacro("$date$_$time$"))

	d := time.Now().Format("2006-01-02")
	assert.EqualValues(t, "test_"+d+".csv", config.GetMacrod("NoSauceFile"))
	assert.EqualValues(t, "parseError_"+config.Name+"_"+d+".csv", config.GetMacrod("ParseErrorFile"))

}
