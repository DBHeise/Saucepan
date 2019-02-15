package main

import (
	"encoding/json"
	"os"
	"reflect"
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
type cybersaucierConfig struct {
	URL   string `json:"URL"`
	Query string `json:"Query"`
}
type configuration struct {
	WatchFolder        string             `json:"WatchFolder"`
	MaxConcurrentFiles int                `json:"MaxConcurrentFiles"`
	DoneFolder         string             `json:"DoneFolder"`
	MoveAfterProcessed bool               `json:"MoveAfterProcessed"`
	IgnoreList         []string           `json:"IgnoreList"`
	SaveNoSauce        bool               `json:"SaveNoSauce"`
	NoSauceFile        string             `json:"NoSauceFile"`
	WaitInterval       int                `json:"WaitInterval"`
	CyberSaucier       cybersaucierConfig `json:"CyberSaucier"`
	CSVOptions         csvconfig          `json:"CSVOptions"`
	ElasticSearch      esconfig           `json:"ElasticSearch"`
	ExtraParsing       []extraparsing     `json:"ExtraParsing"`
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
		CyberSaucier: cybersaucierConfig{
			URL:   "",
			Query: "",
		},
		IgnoreList: make([]string, 0),
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
	getFromEnvVariables("SAUCE_", config)

	log.WithField("Config", config).Debug("Configuration Loaded")
}

///Taken & Modified from:https://github.com/tkanos/gonfig/blob/master/gonfig.go
func getFromEnvVariables(parent string, obj interface{}) {
	typ := reflect.TypeOf(obj)
	// if a pointer to a struct is passed, get the type of the dereferenced object
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)

		// check if we've got a field name override for the environment
		tagContent := p.Tag.Get("json")
		var envKey string
		if len(tagContent) > 0 {
			envKey = parent + tagContent
		} else {
			envKey = parent + p.Name
		}

		value, ok := os.LookupEnv(envKey)

		if !p.Anonymous {
			// struct
			s := reflect.ValueOf(obj).Elem()
			if s.Kind() == reflect.Struct {
				// exported field
				f := s.FieldByName(p.Name)

				if f.Kind() == reflect.Struct {
					getFromEnvVariables(envKey+"_", f.Addr().Interface())
				} else if f.IsValid() && f.CanSet() && ok {
					// A Value can be changed only if it is
					// addressable and was not obtained by
					// the use of unexported struct fields.

					log.WithFields(log.Fields{
						"Key":   envKey,
						"Name":  p.Name,
						"Value": value,
					}).Debug("Environment Override")

					// change value
					kind := f.Kind()
					if kind == reflect.Int || kind == reflect.Int64 {
						setStringToInt(f, value, 64)
					} else if kind == reflect.Int32 {
						setStringToInt(f, value, 32)
					} else if kind == reflect.Int16 {
						setStringToInt(f, value, 16)
					} else if kind == reflect.Uint || kind == reflect.Uint64 {
						setStringToUInt(f, value, 64)
					} else if kind == reflect.Uint32 {
						setStringToUInt(f, value, 32)
					} else if kind == reflect.Uint16 {
						setStringToUInt(f, value, 16)
					} else if kind == reflect.Bool {
						setStringToBool(f, value)
					} else if kind == reflect.Float64 {
						setStringToFloat(f, value, 64)
					} else if kind == reflect.Float32 {
						setStringToFloat(f, value, 32)
					} else if kind == reflect.String {
						f.SetString(value)
					} else {
						log.WithField("Kind", kind).Info("Other kind")
					}
				}
			}
		}
	}
}
func setStringToInt(f reflect.Value, value string, bitSize int) {
	convertedValue, err := strconv.ParseInt(value, 10, bitSize)

	if err == nil {
		if !f.OverflowInt(convertedValue) {
			f.SetInt(convertedValue)
		}
	}
}

func setStringToUInt(f reflect.Value, value string, bitSize int) {
	convertedValue, err := strconv.ParseUint(value, 10, bitSize)

	if err == nil {
		if !f.OverflowUint(convertedValue) {
			f.SetUint(convertedValue)
		}
	}
}

func setStringToBool(f reflect.Value, value string) {
	convertedValue, err := strconv.ParseBool(value)

	if err == nil {
		f.SetBool(convertedValue)
	}
}

func setStringToFloat(f reflect.Value, value string, bitSize int) {
	convertedValue, err := strconv.ParseFloat(value, bitSize)

	if err == nil {
		if !f.OverflowFloat(convertedValue) {
			f.SetFloat(convertedValue)
		}
	}
}
