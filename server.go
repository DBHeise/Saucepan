package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"html"
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
	"github.com/fsnotify/fsnotify"
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

	//Clean up
	for _, item := range ans {
		item["result"] = html.UnescapeString(item["result"])
	}

	return ans, nil
}

func parseExtra(obj *map[string]interface{}, records *[]string, checkvalue *string) {

}

func fileHandler(fullpath string, info os.FileInfo, err error) error {
	if err != nil {
		log.Warn(err)
	} else if !info.IsDir() {
		i := strings.Index(fullpath, "completed")
		if i > -1 {
			log.WithFields(log.Fields{"File": fullpath}).Info("Ignoring file")
		} else {
			log.WithFields(log.Fields{"File": fullpath}).Info("Processing file")

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
			headers := make([]string, 0)
			nojuice := make([][]string, 0)

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
				} else {
					checkvalue = strings.Join(record, ",")
				}

				//Extra parsing
				parseExtra(&obj, &record, &checkvalue)

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

				//Only push if there are results
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
						nojuice = append(nojuice, record)
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

			if config.SavedUnjuiced && len(nojuice) > 0 {
				outFile := path.Join(config.DoneFolder, "nojuice.csv")
				oFile, err := os.OpenFile(outFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
				if err != nil {
					log.WithError(err).Warn("Error opening nojuice.csv")
				} else {
					defer oFile.Close()
					writer := csv.NewWriter(oFile)
					writer.WriteAll(nojuice)

					if err := writer.Error(); err != nil {
						log.WithError(err).Warn("Failure writing to nojuice.csv")
					}
				}
			}

			log.WithFields(log.Fields{"File": fullpath}).Info("File Processing Complete")
		}
	}
	return nil
}

func queueFile(fullpath string) {
	log.WithField("Fullpath", fullpath).Debug("New File Created")

	duration := time.Second * time.Duration(config.WaitInterval)
	log.WithField("Seconds", config.WaitInterval).Debug("Sleeping")
	time.Sleep(duration)

	info, err := os.Stat(fullpath)

	fileHandler(fullpath, info, err)
}

func main() {
	//Setup everything
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

	//Load configuration
	loadConfig(configFile)

	//Initialization connection to ElasticSearch
	initES()

	go func() {
		//Handle all the existing files
		filepath.Walk(config.WatchFolder, fileHandler)
		flushQueue()
	}()

	//Setup folder watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Fatal("Unable to create Folder Watcher")
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					go queueFile(event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.WithError(err).Warn("Error during folder watching")
			}
		}
	}()

	err = watcher.Add(config.WatchFolder)
	if err != nil {
		log.Fatal(err)
	}
	log.WithField("Folder", config.WatchFolder).Info("Watching Folder")

	<-done
}
