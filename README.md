# SaucePan
[![Build Status](https://travis-ci.org/DBHeise/Saucepan.svg?branch=master)](https://travis-ci.org/DBHeise/Saucepan)
[![Go Report Card](https://goreportcard.com/badge/github.com/DBHeise/Saucepan)](https://goreportcard.com/report/github.com/DBHeise/Saucepan)

SaucePan takes your CSV logs runs them through CyberChef (via [CyberSaucier](https://github.com/DBHeise/CyberSaucier)) and then pushes it all to your ElasticSearch backend


## Dockerfile
- available as a docker image, and on dockerhub: [crazydave42/saucepan](https://hub.docker.com/r/crazydave42/saucepan)


## Command-Line Options
```
- config {file}     JSON Configuration file to use
- loglevel {level}  Level of logging: debug|info|warn|error|panic
```

## Configuration
* MoveAfterProcessed - bool - should the files be moved from the input folder to the output folder after it is successfully processed
* WatchFolder - string(path) - The path to monitor for files (includes all subfolders)
* DoneFolder - string(path) - The path to place files when they are completed
* IgnoreList - array of strings - strings that (if found in the FULLPATH of the file) will cause the program to ignore (i.e. NOT process) the file
* CyberSaucier - string(url) - URL to [CyberSaucier](https://github.com/DBHeise/CyberSaucier)
* WaitInterval - int - seconds to wait after a file is created before trying to process it
* SaveNoSauce - bool - should we save a records that do NOT have any valid hits from CyberChef
* NoSauceFile - string(filename) - file to use to save records that do NOT have any valid hits from CyberChef (will be in the DoneFolder)
* CSVOptions - object - Options for CSV parsing
    - FirstRowHeader - bool - is the first row in the CSV the header names
    - CaptureColumn - int - the zero based index of the column that you want to run through CyberChef
* ElasticSearch - object - Options for connecting to ElasticSearch
    - URL - string(url) - base URL to ElasticSearch
    - IndexStart - string - ElasticSearch Index start; the full index is "IndexStart + unmask(DTMask)"
    - DTMask - string - GOLang DateTime mask (see [go documenation](https://golang.org/pkg/time/#Parse) for more information)
    - Type - string - ElasticSearch type
    - QueueSize - int - number of records to use in the ElasticSearch Bulk insert
* ExtraParsing - array of objects - Extra parsing to perform from the CaptureColumn
    - Name - string - Name to use in the ES record
    - Start - string - String to match on that occurs before the capture text
    - End - string  - String to match on that occurs after the capture text
    