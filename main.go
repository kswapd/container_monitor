package main

import (
	"flag"
	"fmt"
	"log"
	"monitor/container_monitor/storage/kafka"
)

func main() {
	flag.Parse()
	//send to kafka
	kafkaSender, errk := kafka.New("monitor_container")
	if errk != nil {
		log.Println("error", errk)
	}
	senderStartError := kafkaSender.Start()
	if senderStartError != nil {
		log.Println("error", senderStartError)
	}

	//让主进程停住，不然主进程退了，goroutine也就退了
	var input string
	fmt.Scanln(&input)
	fmt.Println("done")
}
