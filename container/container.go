package container

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"sync"
	"time"

	"fmt"

	cadvisor_client "github.com/google/cadvisor/client"
	container_info "github.com/google/cadvisor/info/v1"
)

var once sync.Once
var containerClientInstance *cadvisor_client.Client
var environmentId string = ""

const (
	container_uuid_label = "io.rancher.container.uuid"
	container_name_label = "io.rancher.container.name"
	namespace_label      = "io.rancher.stack.name"

	environment_id_label = "caas.hna.environment.id"
	hostUrl              = "http://rancher-metadata/latest/self/host"
	hostBackupUrl        = "http://169.254.169.250/latest/self/host"
)

const (
	DiskStatsAsync = "Async"
	DiskStatsRead  = "Read"
	DiskStatsSync  = "Sync"
	DiskStatsTotal = "Total"
	DiskStatsWrite = "Write"
)

var (
	host = flag.String("cadvisor_host", "127.0.0.1", "cadvisor host")
	port = flag.Uint("cadvisor_port", 8080, "cadvisor port")
)

type DetailContainerFilesystem struct {
	Name     string `json:"container_filesystem_name"`
	Type     string `json:"container_filesystem_type"`
	Capacity uint64 `json:"container_filesystem_capacity"`
	Usage    uint64 `json:"container_filesystem_usage"`
}

type DetailContainerStats struct {
	Timestamp time.Time `json:"timestamp"`

	Cpu_usage_seconds_total  uint64 `json:"container_cpu_usage_seconds_total"`
	Cpu_user_seconds_total   uint64 `json:"container_cpu_user_seconds_total"`
	Cpu_system_seconds_total uint64 `json:"container_cpu_system_seconds_total"`

	Memory_usage_bytes uint64 `json:"container_memory_usage_bytes"`
	Memory_limit_bytes uint64 `json:"container_memory_limit_bytes"`
	Memory_cache       uint64 `json:"container_memory_cache"`
	Memory_rss         uint64 `json:"container_memory_rss"`
	Memory_swap        uint64 `json:"container_memory_swap"`

	Network_receive_bytes_total            uint64 `json:"container_network_receive_bytes_total"`
	Network_receive_packets_total          uint64 `json:"container_network_receive_packets_total"`
	Network_receive_packets_dropped_total  uint64 `json:"container_network_receive_packets_dropped_total"`
	Network_receive_errors_total           uint64 `json:"container_network_receive_errors_total"`
	Network_transmit_bytes_total           uint64 `json:"container_network_transmit_bytes_total"`
	Network_transmit_packets_total         uint64 `json:"container_network_transmit_packets_total"`
	Network_transmit_packets_dropped_total uint64 `json:"container_network_transmit_packets_dropped_total"`
	Network_transmit_errors_total          uint64 `json:"container_network_transmit_errors_total"`

	Filesystem []DetailContainerFilesystem `json:"container_filesystem"`

	Diskio_service_bytes_async uint64 `json:"container_diskio_service_bytes_async"`
	Diskio_service_bytes_read  uint64 `json:"container_diskio_service_bytes_read"`
	Diskio_service_bytes_sync  uint64 `json:"container_diskio_service_bytes_sync"`
	Diskio_service_bytes_total uint64 `json:"container_diskio_service_bytes_total"`
	Diskio_service_bytes_write uint64 `json:"container_diskio_service_bytes_write"`

	Tasks_state_nr_sleeping        uint64 `json:"container_tasks_state_nr_sleeping"`
	Tasks_state_nr_running         uint64 `json:"container_tasks_state_nr_running"`
	Tasks_state_nr_stopped         uint64 `json:"container_tasks_state_nr_stopped"`
	Tasks_state_nr_uninterruptible uint64 `json:"container_tasks_state_nr_uninterruptible"`
	Tasks_state_nr_io_wait         uint64 `json:"container_tasks_state_nr_io_wait"`
}

type DetailContainerInfo struct {
	Timestamp      time.Time              `json:"timestamp"`
	Container_uuid string                 `json:"container_uuid"`
	Environment_id string                 `json:"environment_id"`
	Container_name string                 `json:"container_name"`
	Namespace      string                 `json:"namespace"`
	Stats          []DetailContainerStats `json:"stats"`
}

func GetEvironmentId() (string, error) {
	if len(environmentId) > 0 {
		return environmentId, nil
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", hostUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		// use backup url
		req, err := http.NewRequest("GET", hostBackupUrl, nil)
		if err != nil {
			return "", err
		}
		req.Header.Add("Accept", "application/json")
		resp, err = client.Do(req)
		if err != nil {
			return "", err
		}
	}
	defer resp.Body.Close()
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return "", err
	}

	//get environment id
	if _, ok := body["labels"]; ok {
		labels := body["labels"].(map[string]interface{})
		if _, ok := labels[environment_id_label]; ok {
			envid := labels[environment_id_label].(string)
			environmentId = envid
		} else {
			return "", errors.New("There is no environment_id./n")
		}
	} else {
		return "", errors.New("There is no labels./n")
	}

	return environmentId, nil
}

func getInstance() *cadvisor_client.Client {
	once.Do(func() {
		var url = fmt.Sprintf("http://%s:%d/", *host, *port)
		containerClientInstance, _ = cadvisor_client.NewClient(url)
	})
	return containerClientInstance
}

func RecentStats(start, end time.Time, maxStats int) ([]DetailContainerInfo, error) {
	client := getInstance()

	reqParams := container_info.ContainerInfoRequest{
		NumStats: -1,
		Start:    start,
		End:      end,
	}
	conts, err := client.AllDockerContainers(&reqParams)
	if err != nil {
		return nil, err
	}
	//log.Println("### AllDockerContainers", len(conts), conts)
	var containerInfos []DetailContainerInfo
	for _, v := range conts {
		//
		info, err := convertContainerInfo(&v)
		if err != nil {
			log.Println("", err)
			continue
		}
		if len(info.Stats) > 0 {
			containerInfos = append(containerInfos, info)
		} else {
			//og.Println("### no Stats", v)
		}
	}
	log.Println("### containerInfos", len(containerInfos))
	return containerInfos, nil
}

func convertContainerInfo(containerInfo *container_info.ContainerInfo) (DetailContainerInfo, error) {
	var info DetailContainerInfo
	info.Stats = []DetailContainerStats{}
	//TODO 过滤系统容器。filter systerm container

	var err error
	info.Environment_id, err = GetEvironmentId()
	if err != nil {
		log.Println(err)
	}
	// FIXME for test
	// if info.Environment_id == "" {
	// 	info.Environment_id = "1234"
	// }
	info.Container_uuid = containerInfo.Spec.Labels[container_uuid_label]
	info.Container_name = containerInfo.Spec.Labels[container_name_label]
	info.Namespace = containerInfo.Spec.Labels[namespace_label]
	info.Timestamp = time.Now()

	var memoryLimit uint64
	if containerInfo.Spec.HasMemory {
		memoryLimit = containerInfo.Spec.Memory.Limit
	}
	for _, stats := range containerInfo.Stats {
		detailStats, e := convertContainerStats(stats)
		if e != nil {
			log.Println(e)
			continue
		}
		detailStats.Memory_limit_bytes = memoryLimit
		info.Stats = append(info.Stats, detailStats)
	}
	return info, nil
}

// func convertContainerSpec(containerInfo *info.ContainerSpec) (map[string]interface{}, error) {
// 	infos := make(map[string]interface{})

// 	return infos, nil
// }

func convertContainerStats(containerStats *container_info.ContainerStats) (DetailContainerStats, error) {
	var stats DetailContainerStats
	stats.Timestamp = containerStats.Timestamp
	stats.Cpu_usage_seconds_total = containerStats.Cpu.Usage.Total
	stats.Cpu_user_seconds_total = containerStats.Cpu.Usage.User
	stats.Cpu_system_seconds_total = containerStats.Cpu.Usage.System

	stats.Memory_usage_bytes = containerStats.Memory.Usage
	//stats.Memory_limit_bytes = containerStats.Memory.Usage
	stats.Memory_cache = containerStats.Memory.Cache
	stats.Memory_rss = containerStats.Memory.RSS
	stats.Memory_swap = containerStats.Memory.Swap

	stats.Network_receive_bytes_total = containerStats.Network.RxBytes
	stats.Network_receive_packets_total = containerStats.Network.RxPackets
	stats.Network_receive_packets_dropped_total = containerStats.Network.RxDropped
	stats.Network_receive_errors_total = containerStats.Network.RxErrors
	stats.Network_transmit_bytes_total = containerStats.Network.TxBytes
	stats.Network_transmit_packets_total = containerStats.Network.TxPackets
	stats.Network_transmit_packets_dropped_total = containerStats.Network.TxDropped
	stats.Network_transmit_errors_total = containerStats.Network.TxErrors

	stats.Filesystem = converFsStats(containerStats.Filesystem)

	stats.Diskio_service_bytes_async = sumDiskStats(containerStats.DiskIo.IoServiceBytes, DiskStatsAsync)
	stats.Diskio_service_bytes_read = sumDiskStats(containerStats.DiskIo.IoServiceBytes, DiskStatsRead)
	stats.Diskio_service_bytes_sync = sumDiskStats(containerStats.DiskIo.IoServiceBytes, DiskStatsSync)
	stats.Diskio_service_bytes_total = sumDiskStats(containerStats.DiskIo.IoServiceBytes, DiskStatsTotal)
	stats.Diskio_service_bytes_write = sumDiskStats(containerStats.DiskIo.IoServiceBytes, DiskStatsWrite)

	stats.Tasks_state_nr_sleeping = containerStats.TaskStats.NrSleeping
	stats.Tasks_state_nr_running = containerStats.TaskStats.NrRunning
	stats.Tasks_state_nr_stopped = containerStats.TaskStats.NrStopped
	stats.Tasks_state_nr_uninterruptible = containerStats.TaskStats.NrUninterruptible
	stats.Tasks_state_nr_io_wait = containerStats.TaskStats.NrIoWait
	return stats, nil
}

func converFsStats(fsStats []container_info.FsStats) []DetailContainerFilesystem {
	var fss = make([]DetailContainerFilesystem, len(fsStats))
	for i, fsStat := range fsStats {
		detail := DetailContainerFilesystem{
			Name:     fsStat.Device,
			Type:     fsStat.Type,
			Capacity: fsStat.Limit,
			Usage:    fsStat.Usage,
		}
		fss[i] = detail
	}
	return fss
}

func sumDiskStats(diskStats []container_info.PerDiskStats, key string) uint64 {
	var result uint64

	for _, perDiskStats := range diskStats {
		val, has := perDiskStats.Stats[key]
		if has {
			result += val
		}
	}
	return result
}
