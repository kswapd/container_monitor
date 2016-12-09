FROM google/cadvisor:latest

MAINTAINER tang

ADD bin/container_monitor /usr/bin/container_monitor
ADD bin/docker-entrypoint.sh /usr/bin/docker-entrypoint.sh

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["brokers"]