```
docker run --rm -v "$PWD":/go/src/monitor/container_monitor -w /go/src/monitor/container_monitor golang:1.7.3 go build -v -o bin/container_monitor
```

```
./container_monitor -cadvisor_port=8071 -kafka_topic=capability-container -kafka_broker_list=192.168.100.180:8074,192.168.100.181:8074,192.168.100.182:8074
```


```
docker pull mutemaniac/container-monitor:v0.1.1
sudo docker run \
  --volume=/:/rootfs:ro \
  --volume=/var/run:/var/run:rw \
  --volume=/sys:/sys:ro \
  --volume=/var/lib/docker/:/var/lib/docker:ro \
  --detach=true \
  --name=container-monitor \
  mutemaniac/container-monitor:v0.1.1 \
  192.168.100.180:8074,192.168.100.181:8074,192.168.100.182:8074
```

sudo docker run \
  --volume=/:/rootfs:ro \
  --volume=/var/run:/var/run:rw \
  --volume=/sys:/sys:ro \
  --volume=/var/lib/docker/:/var/lib/docker:ro \
  --detach=true \
  --name=container-monitor \
  registry.time-track.cn/monitor/container-monitor:v0.5 \
  223.202.32.59:8065