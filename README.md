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
All configuration can be set either with a json file or environment variables prepended with "SAUCE_"
* Name - string - this is a string used to differentiate this saucepan from other saucepans that may or may not be running
* MoveAfterProcessed - bool - should the files be moved from the input folder to the output folder after it is successfully processed
* WatchFolder - string(path) - The path to monitor for files (includes all subfolders)
* DoneFolder - string(path) - The path to place files when they are completed
* IgnoreList - array of strings - strings that (if found in the FULLPATH of the file) will cause the program to ignore (i.e. NOT process) the file
* CyberSaucier
    - Enabled - bool - should we even call CyberSaucier
    - URL - string(url) - URL to [CyberSaucier](https://github.com/DBHeise/CyberSaucier)
    - Query - string - additional string to append to CyberSaucier URL request
* WaitInterval - int - seconds to wait after a file is created before trying to process it
* MaxConcurrentFiles - int - the maximum number of files to process simultaniously
* SaveNoSauce - bool - should we save a records that do NOT have any valid hits from CyberChef
* NoSauceFile - string(filename) - file to use to save records that do NOT have any valid hits from CyberChef (will be in the DoneFolder)
    - supports optional ```$date$``` macro for including the current date time in the nosaucefile
* IgnoreList - array of string - if any of these strings are found in the path of the file, it will not be processed
* CSVOptions - object - Options for CSV parsing
    - FirstRowHeader - bool - is the first row in the CSV the header names
    - CaptureColumn - int - the zero based index of the column that you want to run through CyberChef
* ElasticSearch - object - Options for connecting to ElasticSearch
    - URL - string(url) - base URL to ElasticSearch
    - IndexStart - string - ElasticSearch Index start; the full index is "IndexStart + unmask(DTMask)"
    - DTMask - string - GOLang DateTime mask (see [go documenation](https://golang.org/pkg/time/#Parse) for more information)
    - Type - string - ElasticSearch type
    - QueueSize - int - number of records to use in the ElasticSearch Bulk insert
    - Sleep - int - the number of seconds to wait after each ElasticSearch Insert
* ExtraParsing - array of objects - Extra parsing to perform from the CaptureColumn
    - Name - string - Name to use in the ES record
    - Start - string - String to match on that occurs before the capture text
    - End - string  - String to match on that occurs after the capture text
* InputAlert 
    - Threshold - int - the time (in seconds) passed before an alert email is sent in input file ingestion (e.g. if set to 60 then if more than 60 seconds passes between recieving input files, an alert will be sent)
    - Email - string - the email address that will recieve the alert email 
* OutputAlert
    - Threshold - int - the time (in seconds) passed between queueing data to ElasticSearch before an alert email is sent (e.g. if set to 60 then if more than 60 seconds passes between queueing data to send to ES, an alert will be sent)
    - Email - string - the email address that will recieve the alert email 
* MailConfig
    - From - string - the email address the alerts will be sent from
    - Server - string - the SMTP server handling the emails
    - Port - int - the port the SMTP server is listening on
    - User - string - the user used to authenticate to the SMTP Server
    - Password - string - the password used to authenticate to the SMTP Server

### Example config.json
```
{    
    "MoveAfterProcessed": true,
    "WatchFolder": "E:/temp/CSV/input",
    "DoneFolder": "E:/temp/CSV/output",
    "CyberSaucier": {
        "Enabled": true,
        "URL": "http://127.0.0.1:7000",
        "Query":""
    },
    "WaitInterval": 10,    
    "MaxConcurrentFiles": 1,
    "SaveNoSauce": true,
    "NoSauceFile": "nojuice.csv",
    "IgnoreList": ["completed", "ignore", "nojuice"],
    "CSVOptions": {
        "FirstRowHeader": true,
        "CaptureColumn": 6
    },
    "ElasticSearch": {
        "URL": "http://127.0.0.1:9200",        
        "IndexStart" : "cybersaucier-",
        "DTMask": "2006-01-02",
        "Type": "juice",        
        "QueueSize": 100
    },
    "ExtraParsing": [
        { "Name": "X-Forwarded-For", "Start": "\r\nX-Forwarded For: ", "End": "\r\n" },
        { "Name": "XFF", "Start": "\r\nXFF=direct: ", "End": "\r\n" },
        { "Name": "True-Client-IP", "Start": "\r\nTrue-Client-IP: ", "End": "\r\n" }
    ]    
}
```

### Example Environment Variables (almost same config as the json)
ExtraParsing is not supported via environment variables
```
set SAUCE_MoveAfterProcessed=true
set SAUCE_WatchFolder=E:\temp\CSV\input
set SAUCE_DoneFolder=E:\temp\CSV\output
set SAUCE_CyberSaucier_Enabled=true
set SAUCE_CyberSaucier_URL=http://127.0.0.1:7000
set SAUCE_CyberSaucier_Query=
set SAUCE_WaitInterval=10
set SAUCE_SaveNoSauce=true
set SAUCE_NoSauceFile=nojuice.csv
set SAUCE_CSVOptions_FirstRowHeader=true
set SAUCE_CSVOptions_CaptureColumn=6
set SAUCE_ElasticSearch_URL=http://127.0.0.1:9200
set SAUCE_ElasticSearch_IndexStart=cybersaucier-
set SAUCE_ElasticSearch_DTMask=2006-01-02
set SAUCE_ElasticSearch_Type=juice
set SAUCE_ElasticSearch_QueueSize=100
set SAUCE_IgnoreList=completed|ignore|nojuice
```