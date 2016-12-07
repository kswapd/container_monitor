```
docker run --rm -v "$PWD":/go/src/monitor/container_monitor -w /go/src/monitor/container_monitor golang:1.7.3 go build -v -o bin/container_monitor
```

```
./container_monitor -cadvisor_port=8071 -kafka_topic=capability-container -kafka_broker_list=192.168.100.180:8074,192.168.100.181:8074,192.168.100.182:8074
```