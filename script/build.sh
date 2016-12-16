docker run --rm -v "$PWD":/go/src/monitor/container_monitor -w /go/src/monitor/container_monitor golang:1.7.3 go build -v -o bin/container_monitor
docker build -t mutemaniac/container-monitor:v0.6.2 .