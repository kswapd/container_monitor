package main

import (
	"flag"
	"log"
	"monitor/container_monitor/storage/kafka"
	"time"
)

func main() {
	flag.Parse()

	//send to kafka
	kafkaSender, errk := kafka.New("container")
	if errk != nil {
		log.Println("error", errk)
	}
	senderStartError := kafkaSender.Start()
	if senderStartError != nil {
		log.Println("error", senderStartError)
	}

	//让主进程停住，不然主进程退了，goroutine也就退了
	for {
		time.Sleep(100 * time.Second)
	}
}
