package main

import (
	"encoding/json"
	"os"

	log "github.com/Sirupsen/logrus"
)

type csvconfig struct {
	FirstRowHeader bool `json:"FirstRowHeader"`
	CaptureColumn  int  `json:"CaptureColumn"`
}
type esconfig struct {
	URL        string `json:"URL"`
	IndexStart string `json:"IndexStart"`
	DTMask     string `json:"DTMask"`
	Type       string `json:"Type"`
	QueueSize  int    `json:"QueueSize"`
}
type configuration struct {
	MoveAfterProcessed bool      `json:"MoveAfterProcessed"`
	SavedUnjuiced      bool      `json:"SaveUnjuiced"`
	WatchFolder        string    `json:"WatchFolder"`
	DoneFolder         string    `json:"DoneFolder"`
	WaitInterval       int       `json:"WaitInterval"`
	CyberSaucier       string    `json:"CyberSaucier"`
	CSVOptions         csvconfig `json:"CSVOptions"`
	ElasticSearch      esconfig  `json:"ElasticSearch"`
}

func loadConfig(filepath string) {
	log.WithField("ConfigFile", filepath).Debug("Loading Configuration from file")

	file, err := os.Open(filepath)
	if err != nil {
		log.WithError(err).Fatal("Error opening configuration file")
	}
	defer file.Close()

	jsonDecoder := json.NewDecoder(file)
	config = &configuration{}
	err = jsonDecoder.Decode(config)
	if err != nil {
		log.WithError(err).Fatal("Could not decode json configuration")
	}
	log.WithField("Config", config).Debug("Configuration Loaded")
}
