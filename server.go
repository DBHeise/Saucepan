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

	oqueue "github.com/otium/queue"

	log "github.com/Sirupsen/logrus"
	"github.com/fsnotify/fsnotify"
)

var (
	configFile string
	loglevel   string
	logfile    string
	config     *configuration
	fileQueue  *oqueue.Queue
)

func init() {
	flag.StringVar(&configFile, "config", "config.json", "Configuration file To use")
	flag.StringVar(&loglevel, "loglevel", "warn", "Level of debugging {debug|info|warn|error|panic}")
}

func sendToCyberS(input string) ([]map[string]interface{}, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: (&http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		}),
	}
	resp, err := client.Post(config.CyberSaucier.URL+config.CyberSaucier.Query, "text/plain", strings.NewReader(input))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ans := make([]map[string]interface{}, 0)
	err = json.Unmarshal(respBytes, &ans)
	if err != nil {
		return nil, err
	}

	//Clean up
	for _, item := range ans {
		if val, ok := item["result"]; ok {
			item["result"] = html.UnescapeString(val.(string))
		}
	}

	return ans, nil
}

func parseExtra(obj *map[string]interface{}, records []string, checkvalue string) {

	for _, extra := range config.ExtraParsing {
		start := strings.Index(checkvalue, extra.Start)
		if start > -1 {
			trueStart := start + len(extra.Start)
			end := strings.Index(checkvalue[trueStart:], extra.End)
			if end <= -1 {
				end = len(checkvalue)
			} else {
				end += trueStart
			}

			(*obj)[extra.Name] = checkvalue[trueStart:end]
		}
	}

}

func shouldIgnore(fullpath string) bool {
	ans := false
	for _, tst := range config.IgnoreList {
		if strings.Index(fullpath, tst) > -1 {
			ans = true
			break
		}
	}
	return ans
}

func fileWalkHandler(fullpath string, info os.FileInfo, err error) error {
	if err != nil {
		log.Warn(err)
	} else if !info.IsDir() {
		queueFile(fullpath)
	}
	return nil
}

func fileHandler(obj interface{}) {
	fullpath := obj.(string)
	if fullpath != "" {
		info, err := os.Stat(fullpath)

		if err != nil {
			log.Warn(err)
			return
		}

		if !info.IsDir() {
			if shouldIgnore(fullpath) {
				log.WithFields(log.Fields{"File": fullpath}).Info("Ignoring file")
			} else if info.Size() == 0 {
				log.WithFields(log.Fields{"File": fullpath}).Info("Empty file")
				os.Remove(fullpath)
			} else {
				log.WithFields(log.Fields{"File": fullpath}).Info("Processing file")

				filename := info.Name()
				var dtStamp string
				var tag string
				i := strings.Index(filename, "_")
				if i > -1 {
					parts := strings.Split(filename, "_")
					dtStamp = strings.Split(parts[1], ".")[0]
					tag = parts[0]
				}

				f, err := os.Open(fullpath)
				if err != nil {
					log.WithError(err).Warn("Could not open file")
					return
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

					if err == io.EOF {
						break
					}

					if err != nil {
						hadAnyErrors = true
						log.WithError(err).Warn("Could not read record")
						break
					}

					obj := make(map[string]interface{})
					obj["FileName"] = filename
					obj["Line"] = line
					obj["Tag"] = tag
					if dtStamp != "" {
						obj["DateTime"] = dtStamp
					}
					numRecords := len(record)
					var checkvalue string

					if len(headers) == numRecords {
						for i := 0; i < numRecords; i++ {
							switch headers[i] {
							case "dest_ip":
								fallthrough
							case "dest_port":
								fallthrough
							case "src_ip":
								if strings.Contains(record[i], " ") {
									obj[headers[i]] = strings.Split(record[i], " ")
								} else {
									obj[headers[i]] = record[i]
								}
								break
							default:
								obj[headers[i]] = record[i]
							}
						}
					}

					if config.CSVOptions.CaptureColumn <= numRecords {
						checkvalue = record[config.CSVOptions.CaptureColumn]
					} else {
						checkvalue = strings.Join(record, ",")
					}

					//Extra parsing
					parseExtra(&obj, record, checkvalue)

					//Send to CyberSaucier
					if config.CyberSaucier.Enabled {
						cybers, err := sendToCyberS(checkvalue)
						if err != nil {
							log.WithError(err).Warn("Error in CyberSaucier")
							hadAnyErrors = true
						}

						//Append CyberSaucier results to obj
						cResults := make([]map[string]interface{}, 0)
						for _, result := range cybers {
							if val, ok := result["result"]; ok && len(val.(string)) > 0 {
								cResults = append(cResults, result)
							}
						}

						//Only push if there are results
						if len(cResults) > 0 {
							cs := make([]interface{}, 0)
							hitlist := make([]string, 0)
							recipeNameList := make([]string, 0)
							for _, item := range cResults {
								rslt := item["result"].(string)
								if fieldname, ok := item["fieldname"]; ok {
									obj[fieldname.(string)] = strings.Split(rslt, "\n")
								} else {
									cs = append(cs, item)
									hitlist = append(hitlist, strings.Split(rslt, "\n")...)
									recipeNameList = append(recipeNameList, item["recipeName"].(string))
								}
							}

							obj["Hits"] = hitlist
							obj["RecipeNames"] = recipeNameList
							obj["CyberSaucier"] = cs

							//Send to ES
							err = sendDataToES(obj)
							if err != nil {
								log.WithError(err).Warn("Error in CyberSaucier")
								hadAnyErrors = true
							}
						} else {
							if config.SaveNoSauce {
								nojuice = append(nojuice, record)
							}
						}
					} else {
						//CyberSaucier is disabled - push it all
						//Send to ES
						err = sendDataToES(obj)
						if err != nil {
							log.WithError(err).Warn("Error in CyberSaucier")
							hadAnyErrors = true
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
				} else {
					log.WithFields(log.Fields{
						"src": fullpath,
					}).Debug("File had errors")
				}

				if config.SaveNoSauce && len(nojuice) > 0 {
					outFile := path.Join(config.DoneFolder, config.NoSauceFile)
					oFile, err := os.OpenFile(outFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
					if err != nil {
						log.WithError(err).Warn("Error opening NoSauceFile")
					} else {
						defer oFile.Close()
						writer := csv.NewWriter(oFile)
						writer.WriteAll(nojuice)

						if err := writer.Error(); err != nil {
							log.WithError(err).Warn("Failure writing to NoSauceFile")
						}
					}
				}

				log.WithFields(log.Fields{"File": fullpath}).Info("File Processing Complete")

			}
		}
	}
}

func waitFile(fullpath string) {
	log.WithField("Fullpath", fullpath).Debug("New File Created")

	duration := time.Second * time.Duration(config.WaitInterval)
	log.WithField("Seconds", config.WaitInterval).Debug("Sleeping")
	time.Sleep(duration)

	queueFile(fullpath)
}

func queueFile(fullpath string) {
	log.WithField("Fullpath", fullpath).Debug("Queueing File")
	fileQueue.Push(fullpath)
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

	fileQueue = oqueue.NewQueue(fileHandler, config.MaxConcurrentFiles)

	//Initialization connection to ElasticSearch
	initES()

	go func() {
		//Handle all the existing files
		filepath.Walk(config.WatchFolder, fileWalkHandler)
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
					go waitFile(event.Name)
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
		log.WithError(err).Fatal("Unable to create folder watcher")
	}
	log.WithField("Folder", config.WatchFolder).Info("Watching Folder")

	<-done
}
