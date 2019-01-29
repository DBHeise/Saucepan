package main

import (
	"io/ioutil"
	"os"
	"testing"

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
		MoveAfterProcessed: true,
		SaveNoSauce:        false,
		NoSauceFile:        "asdf.csv",
		WaitInterval:       30,
		CyberSaucier:       "http://1.2.3.4:9999",
		IgnoreList:         make([]string, 0),
		CSVOptions: csvconfig{
			FirstRowHeader: false,
			CaptureColumn:  0,
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

	loadConfig(fullFile)

	actual := config

	assert.EqualValues(t, expected, actual)

	os.Remove(fullFile)
}
