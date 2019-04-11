package main

import (
	"os"
	"time"
)

func checkConfig(filePath string) {

	ohash, err := hashFileMd5(filePath)
	if err != nil {
		panic(err)
	}
	log.Infof("Check config started on %s  (hash: %s)", filePath, ohash)
	for {
		hash, err := hashFileMd5(filePath)
		if err != nil {
			panic(err)
		}
		if ohash != hash {
			log.Info("New Config found stop sidecar")
			os.Exit(0)
		}
		time.Sleep(5 * time.Minute)
		//time.Sleep(5 * time.Second)
	}
}
