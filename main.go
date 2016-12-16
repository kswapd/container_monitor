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

	"monitor/container_monitor/container"
)

var (
	monitorType   = flag.String("monitor_type", "container", "")
	storageDriver = flag.String("storage_driver", "kafka", fmt.Sprintf("Storage `driver` to use. Data is always cached shortly in memory, this controls where data is pushed besides the local cache. Empty means none. Options are: <empty>, %s", strings.Join(storage.ListDrivers(), ", ")))
)

func main() {
	flag.Parse()

	for {
		//get evironment id first.
		envid, err := container.GetEvironmentId()
		if err != nil {
			log.Println("GetEvironmentId error", err)
			continue
		}
		if len(envid) != 0 {
			break
		}
		time.Sleep(time.Second * 1)
	}

	var sender storage.StorageDriver
	var errk error
	if *storageDriver == "kafka" {
		sender, errk = kafka.New(*monitorType)
	} else if *storageDriver == "influxdb" {
		sender, errk = influxdb.New(*monitorType)
	} else {
		sender, errk = kafka.New(*monitorType)
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
