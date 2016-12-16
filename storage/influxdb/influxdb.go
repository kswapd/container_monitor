package influxdb

import (
	"flag"
	"fmt"
	"log"

	"monitor/container_monitor/container"
	"net/url"
	"os"
	"sync"
	"time"

	"monitor/container_monitor/storage"

	"github.com/fatih/structs"
	influx "github.com/influxdata/influxdb/client/v2"
)

var (
	sendInterval = flag.Duration("influxdb_send_interval", 1*time.Second, "send interval")
)

// func init() {
// 	storage.RegisterStorageDriver("influxdb", New)
// }

type influxdbStorage struct {
	client          influx.Client
	machineName     string
	database        string
	retentionPolicy string
	bufferDuration  time.Duration
	lastWrite       time.Time
	points          []*influx.Point
	lock            sync.Mutex
	readyToFlush    func() bool

	stop         chan bool
	sendInterval time.Duration
	monitorType  string
}

// Field names
const (
	fieldValue  string = "value"
	fieldType   string = "type"
	fieldDevice string = "device"
	fieldLimit  string = "limit"
)

// Tag names
const (
	tagContainerUuid  string = "container_uuid"
	tagEnvironmentId  string = "environment_id"
	tagContainerName  string = "container_name"
	tagNamespace      string = "namespace"
	tagType           string = "type"
	tagSubType        string = "subtype"
	tagFilesystemName string = "container_filesystem_name"
)

func (self *influxdbStorage) filesystem2Points(
	filesystems []container.DetailContainerFilesystem,
	timestamp time.Time,
	tags map[string]string,
	batchPoints influx.BatchPoints,
) {
	tags[tagSubType] = "filesystem"
	for _, filesystem := range filesystems {
		tags[tagFilesystemName] = filesystem.Name
		//使用量
		fields := map[string]interface{}{
			fieldValue: toSignedIfUnsigned(filesystem.Usage),
			fieldLimit: toSignedIfUnsigned(filesystem.Capacity),
		}
		p, err := influx.NewPoint(
			"container_filesystem_usage",
			tags,
			fields,
			timestamp,
		)
		if err == nil {
			batchPoints.AddPoint(p)
		} else {
			log.Println("### makePoint error", err)
		}

		//容量
		fields = map[string]interface{}{
			fieldValue: toSignedIfUnsigned(filesystem.Capacity),
		}
		p, err = influx.NewPoint(
			"container_filesystem_capacity",
			tags,
			fields,
			timestamp,
		)
		if err == nil {
			batchPoints.AddPoint(p)
		} else {
			log.Println("### makePoint error", err)
		}
	}
}

//组织数据
func (self *influxdbStorage) statsToPoints(
	infos []container.DetailContainerInfo,
	batchPoints influx.BatchPoints,
) {

	for _, info := range infos {
		commonTags := map[string]string{
			tagContainerName: info.Container_name,
			tagContainerUuid: info.Container_uuid,
			tagEnvironmentId: info.Environment_id,
			tagNamespace:     info.Namespace,
			tagType:          self.monitorType,
		}
		for _, stat := range info.Stats {
			stats := structs.Map(stat)
			for key, val := range stats {
				if key == "timestamp" || key == "container_filesystem" {
					continue
				}
				fields := map[string]interface{}{
					fieldValue: toSignedIfUnsigned(val),
				}
				p, err := influx.NewPoint(
					key,
					commonTags,
					fields,
					stat.Timestamp,
				)
				if err == nil {
					batchPoints.AddPoint(p)
				} else {
					log.Println("### makePoint error", err)
				}
			}
		}

	}
}

func (self *influxdbStorage) OverrideReadyToFlush(readyToFlush func() bool) {
	self.readyToFlush = readyToFlush
}

func (self *influxdbStorage) defaultReadyToFlush() bool {
	return time.Since(self.lastWrite) >= self.bufferDuration
}

func (self *influxdbStorage) Send(start, end time.Time) (time.Time, error) {
	log.Println("~~~ Enter InfluxDB Send.", start, end)
	stats, last, errCache := container.RecentStats(start, end)

	if errCache != nil {
		log.Println("~~~ Kafka InfluxDB error.", errCache)
		return start, errCache
	}
	if len(stats) == 0 {
		log.Println("~~~ Kafka InfluxDB 0.")
		return start, nil
	}

	if len(stats) > 0 {

		bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
			Database:  self.database,
			Precision: "ns",
		})
		if err != nil {
			log.Println("NewBatchPoints error. ", err)
			return start, nil
		}

		self.statsToPoints(stats, bp)
		err = self.client.Write(bp)
		if err != nil {
			fmt.Errorf("failed to write stats to influxDb - %s", err)
		}
	}

	if last.IsZero() {
		last = start
	}
	return last, nil
}

func (self *influxdbStorage) Close() error {
	self.client = nil
	return nil
}

func (driver *influxdbStorage) Stop() error {
	driver.stop <- true
	return nil
}
func (driver *influxdbStorage) Start() error {
	log.Println("Enter Influxdb Start.")
	go driver.startSend()
	return nil
}
func (driver *influxdbStorage) startSend() {
	last := time.Now()
	for {
		select {
		case <-driver.stop:
			return
		default:
			now := time.Now()
			latest, err := driver.Send(last, now)
			if err != nil {
				log.Println("~~~ Influxdb send error.", err)
			}

			last = latest
			time.Sleep(driver.sendInterval)
		}
	}
}
func New(monitorType string) (*influxdbStorage, error) {
	log.Println("~~~ Enter influxdb new.")
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return newStorage(
		hostname,
		*storage.ArgDbUsername,
		*storage.ArgDbPassword,
		*storage.ArgDbName,
		*storage.ArgDbHost,
		*storage.ArgDbIsSecure,
		*storage.ArgDbBufferDuration,
		monitorType,
	)
}

// machineName: A unique identifier to identify the host that current cAdvisor
// instance is running on.
// influxdbHost: The host which runs influxdb (host:port)
func newStorage(
	machineName,
	username,
	password,
	database,
	influxdbHost string,
	isSecure bool,
	bufferDuration time.Duration,
	monitorType string,
) (*influxdbStorage, error) {
	url := &url.URL{
		Scheme: "http",
		Host:   influxdbHost,
	}
	if isSecure {
		url.Scheme = "https"
	}

	config := &influx.HTTPConfig{
		Addr:     url.String(),
		Username: username,
		Password: password,
		//UserAgent: fmt.Sprintf("%v/%v", "cAdvisor", version.Info["version"]),
	}
	client, err := influx.NewHTTPClient(*config)
	if err != nil {
		return nil, err
	}

	ret := &influxdbStorage{
		client:         client,
		machineName:    machineName,
		database:       database,
		bufferDuration: bufferDuration,
		lastWrite:      time.Now(),

		stop:         make(chan bool, 1),
		points:       make([]*influx.Point, 0),
		monitorType:  monitorType,
		sendInterval: *sendInterval,
	}
	ret.readyToFlush = ret.defaultReadyToFlush
	return ret, nil
}

// Creates a measurement point with a single value field
func makePoint(name string, value interface{}, timestamp time.Time, tags map[string]string) (*influx.Point, error) {
	fields := map[string]interface{}{
		fieldValue: toSignedIfUnsigned(value),
	}
	return influx.NewPoint(
		name,
		tags,
		fields,
		timestamp,
	)
}

// Some stats have type unsigned integer, but the InfluxDB client accepts only signed integers.
func toSignedIfUnsigned(value interface{}) interface{} {
	switch v := value.(type) {
	case uint64:
		return int64(v)
	case uint32:
		return int32(v)
	case uint16:
		return int16(v)
	case uint8:
		return int8(v)
	case uint:
		return int(v)
	}
	return value
}
