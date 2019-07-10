package main

import (
	"crypto/tls"
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

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

var (
	configFile string
	loglevel   string
	config     *configuration
	fileQueue  *oqueue.Queue
)

type SauceParseError struct {
	File       string
	Line       int
	Column     int
	ErrMessage string
	Raw        string
}

func init() {
	flag.StringVar(&configFile, "config", "config.json", "Configuration file To use")
	flag.StringVar(&loglevel, "loglevel", "warn", "Level of debugging {debug|info|warn|error|panic}")
}

func sendToCyberS(input string) ([]map[string]interface{}, error) {
	//proxyURL, _ := url.Parse("http://localhost:8888")
	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: (&http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
			//Proxy:               http.ProxyURL(proxyURL),
		}),
	}
	req, reqErr := http.NewRequest("POST", config.CyberSaucier.URL+config.CyberSaucier.Query, strings.NewReader(input))
	if reqErr != nil {
		return nil, reqErr
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Connection", "close")
	req.Close = true
	resp, err := client.Do(req)
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
		if strings.Contains(fullpath, tst) {
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

func parseLine(filename string, line int, tag string, dtStamp string, headers []string, record []string) (map[string]interface{}, string) {

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

	return obj, checkvalue
}

func fileHandler(infileObj interface{}) {
	fullpath := infileObj.(string)
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
				reader.ReuseRecord = false
				reader.LazyQuotes = true
				reader.TrimLeadingSpace = true

				headers := make([]string, 0)
				nojuice := make([][]string, 0)
				parseerrors := make([]SauceParseError, 0)

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
						if pe, ok := err.(*csv.ParseError); ok {
							if pe.Err != csv.ErrFieldCount {
								spe := SauceParseError{
									File:       fullpath,
									Line:       line,
									Column:     pe.Column,
									ErrMessage: pe.Err.Error(),
									Raw:        "",
								}
								parseerrors = append(parseerrors, spe)
							}
						} else {
							log.WithError(err).Warn("Could not read record")
							continue
						}
					}

					obj, checkvalue := parseLine(filename, line, tag, dtStamp, headers, record)

					//Send to CyberSaucier
					if config.CyberSaucier.Enabled {
						cybers, err := sendToCyberS(checkvalue)
						if err != nil {
							log.WithError(err).Warn("Error in CyberSaucier")
							//hadAnyErrors = true
						}

						//Append CyberSaucier results to obj
						cResults := make([]map[string]interface{}, 0)
						for _, result := range cybers {
							if val, ok := result["result"]; ok && len(val.(string)) > 0 { //looking for non-empty "result" fields
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

							log.WithFields(log.Fields{"Record": record, "CyberSaucier": cybers, "Obj": obj}).Trace("Juice")
							//Send to ES
							err = sendDataToES(obj)
							if err != nil {
								log.WithError(err).Warn("Error in CyberSaucier")
								//hadAnyErrors = true
							}
						} else {
							log.WithFields(log.Fields{"Record": record, "CyberSaucier": cybers, "Obj": obj}).Trace("No Juice")
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
							//hadAnyErrors = true
						}
					}
				}

				f.Close()

				//Move the file
				if config.MoveAfterProcessed {
					newDst := path.Join(config.DoneFolder, filename)
					log.WithFields(log.Fields{
						"src": fullpath,
						"dst": newDst,
					}).Debug("Moving File")
					err := os.Rename(fullpath, newDst)
					if err != nil {
						log.WithFields(log.Fields{
							"src": fullpath,
							"dst": newDst,
							"err": err,
						}).Warning("Error moving File")
					}
				}

				//Save Parse errors
				if len(parseerrors) > 0 {
					parseerrorfile := config.GetMacrod("ParseErrorFile")
					outFile := path.Join(config.DoneFolder, parseerrorfile)
					oFile, err := os.OpenFile(outFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
					if err != nil {
						log.WithError(err).Warn("Error opening ParseErrorFile")
					} else {
						defer oFile.Close()
						for _, e := range parseerrors {
							errorJson, err := json.Marshal(e)
							if err != nil {
								log.WithError(err).Warn("Error marshalling parseerror to json")
							} else {
								_, err = io.WriteString(oFile, string(errorJson)+"\n")
								if err != nil {
									log.WithError(err).Warn("Error writing to ParseErrorFile")
								}
							}
						}
					}
				}

				//Save "NoSauce" results
				if config.SaveNoSauce && len(nojuice) > 0 {
					nosaucefile := config.GetMacrod("NoSauceFile")

					outFile := path.Join(config.DoneFolder, nosaucefile)
					oFile, err := os.OpenFile(outFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
					if err != nil {
						log.WithError(err).Warn("Error opening NoSauceFile")
					} else {
						defer oFile.Close()
						writer := csv.NewWriter(oFile)
						err := writer.WriteAll(nojuice)
						if err != nil {
							log.WithError(err).Warn("Failure writing to NoSauceFile")
						}

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
	lastInputActionTime = time.Now()
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

	if config.IgnoreCertErrors {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	//Initialization connection to ElasticSearch
	initES()

	fileQueue = oqueue.NewQueue(fileHandler, config.MaxConcurrentFiles)

	go func() {
		//Handle all the existing files
		err := filepath.Walk(config.WatchFolder, fileWalkHandler)
		if err != nil {
			log.WithError(err).Warning("Error walking filepath")
		}
		flushQueue()
	}()

	go timerInputWatcher()
	go timerOutputWatcher()

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
