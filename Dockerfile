FROM debian:stretch-slim

WORKDIR /

COPY k8s-device-plugin-example /usr/local/bin

CMD ["k8s-device-plugin-example", "-alsologtostderr"]