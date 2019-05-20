package main

import (
	"context"
	"time"

	log "github.com/Sirupsen/logrus"
	elastic "github.com/olivere/elastic"
)

var (
	queue []map[string]interface{}

	esClient  *elastic.Client
	esContext context.Context
	lastFlush time.Time
)

func initES() {
	var err error
	esContext = context.Background()
	queue = make([]map[string]interface{}, 0)
	if config.ElasticSearch.UserName != "" {
		esClient, err = elastic.NewClient(elastic.SetURL(config.ElasticSearch.URL), elastic.SetBasicAuth(config.ElasticSearch.UserName, config.ElasticSearch.Password))
	} else {
		esClient, err = elastic.NewClient(elastic.SetURL(config.ElasticSearch.URL))
	}
	if err != nil {
		log.WithError(err).Fatal("Unable to create an ElasticSearch Client")
	}
}

func flushQueue() {
	index := config.ElasticSearch.IndexStart + time.Now().Format(config.ElasticSearch.DTMask)

	if len(queue) > 0 {
		req := esClient.Bulk()
		for _, obj := range queue {
			breq := elastic.NewBulkIndexRequest().Index(index).Type(config.ElasticSearch.Type).Doc(obj)
			req.Add(breq)
		}

		resp, err := req.Do(esContext)
		if err != nil {
			log.WithError(err).Warn("Unable to push data to ElasticSearch")
		}

		log.WithField("Result", resp).Debug("ElasticSearch Response")
		queue = make([]map[string]interface{}, 0)
	}
}

func sendDataToES(object map[string]interface{}) error {
	if config.ElasticSearch.Enabled {
		queue = append(queue, object)

		if len(queue) >= config.ElasticSearch.QueueSize || int(time.Now().Sub(lastFlush).Seconds()) >= config.WaitInterval {
			flushQueue()
			lastFlush = time.Now()

			log.WithField("Seconds", config.ElasticSearch.Sleep).Debug("Sleeping")
			time.Sleep(time.Second * time.Duration(config.ElasticSearch.Sleep))
		}
	}
	return nil
}
