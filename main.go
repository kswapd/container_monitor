package main

import (
	"flag"
	"fmt"
	"log"
	"monitor/container_monitor/storage"
	"monitor/container_monitor/storage/influxdb"
	"monitor/container_monitor/storage/kafka"
	"strings"
	"time"
)

var (
	storageDriver = flag.String("storage_driver", "kafka", fmt.Sprintf("Storage `driver` to use. Data is always cached shortly in memory, this controls where data is pushed besides the local cache. Empty means none. Options are: <empty>, %s", strings.Join(storage.ListDrivers(), ", ")))
)

func main() {
	flag.Parse()

	var sender storage.StorageDriver
	var errk error

	if *storageDriver == "kafka" {
		sender, errk = kafka.New("container")
	} else if *storageDriver == "influxdb" {
		sender, errk = influxdb.New("container")
	} else {
		sender, errk = kafka.New("container")
	}

	if errk != nil {
		log.Println("StorageDriver.New error", errk)
		return
	}

	senderStartError := sender.Start()
	if senderStartError != nil {
		log.Println("error", senderStartError)
	}

	//让主进程停住，不然主进程退了，goroutine也就退了
	for {
		time.Sleep(100 * time.Second)
	}
}
