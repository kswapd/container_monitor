// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"monitor/container_monitor/container"
	"os"
	"strings"
	"time"

	kafka "github.com/Shopify/sarama"
)

var (
	brokers      = flag.String("kafka_broker_list", "223.202.32.59:8065", "kafka broker(s) csv")
	topic        = flag.String("kafka_topic", "capability-container", "kafka topic")
	certFile     = flag.String("kafka_ssl_cert", "", "optional certificate file for TLS client authentication")
	keyFile      = flag.String("kafka_ssl_key", "", "optional key file for TLS client authentication")
	caFile       = flag.String("kafka_ssl_ca", "", "optional certificate authority file for TLS client authentication")
	verifySSL    = flag.Bool("kafka_ssl_verify", true, "verify ssl certificate chain")
	sendInterval = flag.Duration("kafka_send_interval", 1*time.Second, "send interval")
)

type KafkaStorage struct {
	stop         chan bool
	sendInterval time.Duration
	producer     kafka.AsyncProducer
	topic        string
	machineName  string
	monitorType  string
}

func (driver *KafkaStorage) Stop() error {
	driver.stop <- true
	return nil
}
func (driver *KafkaStorage) Start() error {
	log.Println("Enter kafka Start.")
	go driver.startSend()
	return nil
}
func (driver *KafkaStorage) startSend() {
	last := time.Now()
	for {
		select {
		case <-driver.stop:
			return
		default:
			now := time.Now()
			err := driver.send(last, now)
			if err != nil {
				log.Println("~~ Kafka send error.", err)
			}

			last = now
			time.Sleep(driver.sendInterval)
		}
	}
}

type DetailSpec struct {
	MonitorType string                          `json:"type"`
	Data        []container.DetailContainerInfo `json:"data"`
}

func (driver *KafkaStorage) infoToDetailSpec(info []container.DetailContainerInfo) DetailSpec {
	return DetailSpec{
		MonitorType: driver.monitorType,
		Data:        info,
	}
}
func (driver *KafkaStorage) send(start, end time.Time) error {
	log.Println("~~~ Enter Kafka Send.", start, end)

	stats, errCache := container.RecentStats(start, end, -1)

	if errCache != nil {
		log.Println("~~~ Kafka send error.", errCache)
		return errCache
	}
	if len(stats) == 0 {
		log.Println("~~~ Kafka send 0.")
		return nil
	}

	detail := driver.infoToDetailSpec(stats)
	b, err := json.Marshal(detail)

	driver.producer.Input() <- &kafka.ProducerMessage{
		Topic: driver.topic,
		Value: kafka.StringEncoder(b),
	}
	log.Println("~~~ End of Kafka Send.")
	return err
}

func (self *KafkaStorage) Close() error {
	return self.producer.Close()
}

func New(monitorType string) (*KafkaStorage, error) {
	log.Println("~~~ Enter kafka new.")
	machineName, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return newStorage(machineName, monitorType)
}

func generateTLSConfig() (*tls.Config, error) {
	if *certFile != "" && *keyFile != "" && *caFile != "" {
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			return nil, err
		}

		caCert, err := ioutil.ReadFile(*caFile)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		return &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: *verifySSL,
		}, nil
	}

	return nil, nil
}

func newStorage(machineName string, monitorType string) (*KafkaStorage, error) {
	log.Println("~~~ Enter kafka newStorage.")
	config := kafka.NewConfig()

	tlsConfig, err := generateTLSConfig()
	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}

	config.Producer.RequiredAcks = kafka.WaitForAll

	brokerList := strings.Split(*brokers, ",")
	log.Println("Kafka brokers:%q", brokers)

	producer, err := kafka.NewAsyncProducer(brokerList, config)
	log.Println("Kafka producer:%q", producer)
	if err != nil {
		return nil, err
	}
	ret := &KafkaStorage{
		stop:         make(chan bool, 1),
		sendInterval: *sendInterval,
		producer:     producer,
		topic:        *topic,
		machineName:  machineName,
		monitorType:  monitorType,
	}
	return ret, nil
}
