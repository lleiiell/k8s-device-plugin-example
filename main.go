package main

import (
	"flag"
	device_example "github.com/lleiiell/k8s-device-plugin-example/pkg/device-example"
	"k8s.io/klog/v2"
)

func main() {
	flag.Parse()

	klog.Infoln("Start device plugin")

	dp, err := device_example.NewDevicePlugin()
	if err != nil {
		klog.Fatalf("Failed due to %v", err)
	}

	err = dp.Serve()

	if err != nil {
		klog.Fatalf("Failed due to %v", err)
	}

	select {}
}
