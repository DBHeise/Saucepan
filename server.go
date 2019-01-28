package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

var (
	configFile string
	loglevel   string
	logfile    string

	config *configuration
)

func init() {
	flag.StringVar(&configFile, "config", "config.json", "Configuration file To use")
	flag.StringVar(&loglevel, "loglevel", "warn", "Level of debugging {debug|info|warn|error|panic}")
}

func sendToCyberS(input string) ([]map[string]string, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: (&http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		}),
	}
	resp, err := client.Post(config.CyberSaucier, "text/plain", strings.NewReader(input))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ans := make([]map[string]string, 0)
	err = json.Unmarshal(respBytes, &ans)
	if err != nil {
		return nil, err
	}

	return ans, nil
}

func fileHandler(fullpath string, info os.FileInfo, err error) error {
	if err != nil {
		log.Warn(err)
	} else if !info.IsDir() {
		log.WithFields(log.Fields{"File": fullpath}).Debug("Handling file")

		filename := info.Name()
		var dtStamp string
		i := strings.Index(filename, "_")
		if i > -1 {
			parts := strings.Split(filename, "_")
			dtStamp = strings.Split(parts[1], ".")[0]
		}

		f, err := os.Open(fullpath)
		if err != nil {
			log.WithError(err).Warn("Could not open file")
			return nil
		}

		reader := csv.NewReader(f)
		hadAnyErrors := false
		var headers []string
		headers = make([]string, 0)

		line := 0
		if config.CSVOptions.FirstRowHeader {
			headers, err = reader.Read()
			line++
			if err != nil {
				log.WithError(err).Warn("Error reading first record")
			}
		}

		for {
			record, err := reader.Read()
			line++
			obj := make(map[string]interface{})
			obj["FileName"] = filename
			obj["Line"] = line
			if dtStamp != "" {
				obj["DateTime"] = dtStamp
			}
			numRecords := len(record)
			var checkvalue string

			if err == io.EOF {
				break
			}

			if err != nil {
				hadAnyErrors = true
				log.WithError(err).Warn("Could not read record")
				break
			}

			if len(headers) == numRecords {
				for i := 0; i < numRecords; i++ {
					obj[headers[i]] = record[i]
				}
			}

			if config.CSVOptions.CaptureColumn <= numRecords {
				checkvalue = record[config.CSVOptions.CaptureColumn]
			}

			//Send to CyberSaucier
			cybers, err := sendToCyberS(checkvalue)
			if err != nil {
				log.WithError(err).Warn("Error in CyberSaucier")
				hadAnyErrors = true
			}

			//Append CyberSaucier results to obj
			cResults := make([]map[string]string, 0)
			for _, result := range cybers {
				if len(result["result"]) > 0 {
					cResults = append(cResults, result)
				}
			}
			if len(cResults) > 0 {
				obj["CyberSaucier"] = cResults

				//Send to ES
				err = sendDataToES(obj)
				if err != nil {
					log.WithError(err).Warn("Error in CyberSaucier")
					hadAnyErrors = true
				}
			} else {
				if config.SavedUnjuiced {

				}
			}
		}

		f.Close()

		if !hadAnyErrors && config.MoveAfterProcessed {
			newDst := path.Join(config.DoneFolder, filename)
			log.WithFields(log.Fields{
				"src": fullpath,
				"dst": newDst,
			}).Debug("Moving File")
			os.Rename(fullpath, newDst)
		}
	}
	return nil
}

func main() {
	flag.Parse()
	log.SetOutput(os.Stdout)

	logL, err := log.ParseLevel(loglevel)
	if err != nil {
		log.Warn("Unable to parse loglevel, setting to default: info")
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(logL)
	}

	log.Debug("Starting up")

	loadConfig(configFile)
	initES()

	for {

		filepath.Walk(config.WatchFolder, fileHandler)
		flushQueue()

		duration := time.Second * time.Duration(config.WaitInterval)
		log.WithField("Seconds", config.WaitInterval).Debug("Sleeping")
		time.Sleep(duration)
	}

}
