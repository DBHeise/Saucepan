package main

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	lastInputActionTime  time.Time
	lastOutputActionTime time.Time
)

func init() {
	lastInputActionTime = time.Now()
	lastOutputActionTime = time.Now()
}

func timeWatcher(lastActionTime *time.Time, alert alertConfig, subject string) {
	ticker := time.NewTicker(1 * time.Second)
	lastAlertTime := time.Now()

	for range ticker.C {
		durationFile := time.Since(*lastActionTime)
		durationAlert := time.Since(lastAlertTime)

		if int(durationFile.Seconds()) > alert.Threshold {
			if int(durationAlert.Seconds()) > alert.Threshold {

				log.WithFields(log.Fields{
					"LastActionTime": lastActionTime,
					"LastAlertTime":  lastAlertTime,
				}).Warn("Alert!!")

				var sb strings.Builder
				sb.WriteString("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title></title></head><body><h1>")
				fmt.Fprintf(&sb, "%s", subject)
				sb.WriteString("</h1><table><tbody><tr><th>Last Action Time</th><td>")
				fmt.Fprintf(&sb, "%s", lastActionTime)
				sb.WriteString("</td></tr><tr><th>Last Alert Time</th><td>")
				fmt.Fprintf(&sb, "%s", lastAlertTime)
				sb.WriteString("</td></tr><tr><th>Threshold (in seconds)</th><td>")
				fmt.Fprintf(&sb, "%d", alert.Threshold)
				sb.WriteString("</td></tr></tbody></table></body></html>")

				sendMessage(alert.Email, subject, sb.String())
				lastAlertTime = time.Now()
			}
		}
	}
}

func timerInputWatcher() {
	if config.InputAlert.Threshold > 0 {
		timeWatcher(&lastInputActionTime, config.InputAlert, "["+config.Name+"] NO INPUT FILES!!")
	}
}

func timerOutputWatcher() {
	if config.OutputAlert.Threshold > 0 {
		timeWatcher(&lastOutputActionTime, config.OutputAlert, "["+config.Name+"] NO DATA PUSHED!!")
	}
}
