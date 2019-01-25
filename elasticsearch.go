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
)

func initES() {
	var err error
	esContext = context.Background()
	queue = make([]map[string]interface{}, 0)
	esClient, err = elastic.NewClient(elastic.SetURL(config.ElasticSearch.URL))
	if err != nil {
		log.WithError(err).Fatal("Unable to create an ElasticSearch Client")
	}
}

func flushQueue() {
	index := config.ElasticSearch.IndexStart + time.Now().Format(config.ElasticSearch.DTMask)

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

func sendDataToES(object map[string]interface{}) error {
	queue = append(queue, object)

	if len(queue) >= config.ElasticSearch.QueueSize {
		flushQueue()
	}
	return nil
}
