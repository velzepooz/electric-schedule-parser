package main

import "gopkg.in/robfig/cron.v2"

func startCroneJob(pattern string, cb func()) {
	cronJob := cron.New()

	_, err := cronJob.AddFunc(pattern, cb)
	if err != nil {
		return
	}

	cronJob.Start()
}
