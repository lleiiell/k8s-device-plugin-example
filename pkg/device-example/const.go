package device_example

import pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

const (
	resourceName  = "example.com/device"
	resourceCount = "example.com/device-count"
	serverSock    = pluginapi.DevicePluginPath + "deviceexample.sock"
)
