package main

import (
	"encoding/json"
	"os"
	"strconv"

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
type extraparsing struct {
	Name  string `json:"Name"`
	Start string `json:"Start"`
	End   string `json:"End"`
}
type configuration struct {
	WatchFolder        string         `json:"WatchFolder"`
	MaxConcurrentFiles int            `json:"MaxConcurrentFiles"`
	DoneFolder         string         `json:"DoneFolder"`
	MoveAfterProcessed bool           `json:"MoveAfterProcessed"`
	IgnoreList         []string       `json:"IgnoreList"`
	SaveNoSauce        bool           `json:"SaveNoSauce"`
	NoSauceFile        string         `json:"NoSauceFile"`
	WaitInterval       int            `json:"WaitInterval"`
	CyberSaucier       string         `json:"CyberSaucier"`
	CSVOptions         csvconfig      `json:"CSVOptions"`
	ElasticSearch      esconfig       `json:"ElasticSearch"`
	ExtraParsing       []extraparsing `json:"ExtraParsing"`
}

func createDefaultConfig() {
	defaultConfig := &configuration{
		WatchFolder:        ".\\Watch",
		DoneFolder:         ".\\Done",
		MaxConcurrentFiles: 3,
		MoveAfterProcessed: true,
		SaveNoSauce:        false,
		NoSauceFile:        "nojuice.csv",
		WaitInterval:       30,
		CyberSaucier:       "",
		IgnoreList:         make([]string, 0),
		CSVOptions: csvconfig{
			FirstRowHeader: false,
			CaptureColumn:  0,
		},
		ElasticSearch: esconfig{
			URL:        "",
			IndexStart: "cybersaucier-",
			DTMask:     "20060102",
			Type:       "data",
			QueueSize:  100,
		},
		ExtraParsing: make([]extraparsing, 0),
	}
	saveConfig("./config.json", defaultConfig)
}

func saveConfig(filepath string, cfg *configuration) {

	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.WithError(err).Fatal("Error opening configuration file")
	}

	defer file.Close()

	jsonEncoder := json.NewEncoder(file)

	err = jsonEncoder.Encode(cfg)
	if err != nil {
		log.WithError(err).Fatal("Error encoding json configuration")
	}

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

	//Load Environment Variable Overrides
	//TODO: make this more generic for all options
	val := os.Getenv("SAUCE_CSV_CAPTURECOLUMN")
	if val != "" {
		v, e := strconv.Atoi(val)
		if e == nil {
			config.CSVOptions.CaptureColumn = v
		}
	}

	log.WithField("Config", config).Debug("Configuration Loaded")
}
