package device_example

import (
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	log "github.com/golang/glog"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// DevicePlugin implements the Kubernetes device plugin API
type DevicePlugin struct {
	devs   []*pluginapi.Device
	socket string
	stop   chan struct{}
	health chan *pluginapi.Device

	server *grpc.Server
	sync.RWMutex
}

// NewDevicePlugin returns an initialized DevicePlugin
func NewDevicePlugin() (*DevicePlugin, error) {
	devs := getDevices()

	return &DevicePlugin{
		devs:   devs,
		socket: serverSock,
		stop:   make(chan struct{}),
		health: make(chan *pluginapi.Device),
	}, nil
}

func (m *DevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

func (m *DevicePlugin) GetPreferredAllocation(context.Context, *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

// dial establishes the gRPC communication with the registered device plugin.
func dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

// Start starts the gRPC server of the device plugin
func (m *DevicePlugin) Start() error {
	err := m.cleanup()
	if err != nil {
		return err
	}

	sock, err := net.Listen("unix", m.socket)
	if err != nil {
		return err
	}

	m.server = grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(m.server, m)

	go m.server.Serve(sock)

	// Wait for server to start by launching a blocking connexion
	conn, err := dial(m.socket, 5*time.Second)
	if err != nil {
		return err
	}
	_ = conn.Close()

	return nil
}

// Stop stops the gRPC server
func (m *DevicePlugin) Stop() error {
	if m.server == nil {
		return nil
	}
	m.server.Stop()
	m.server = nil
	close(m.stop)
	return m.cleanup()
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (m *DevicePlugin) Register(kubeletEndpoint, resourceName string) error {
	conn, err := dial(kubeletEndpoint, 5*time.Second)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()
	registrationClient := pluginapi.NewRegistrationClient(conn)
	registerRequest := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(m.socket),
		ResourceName: resourceName,
	}
	_, err = registrationClient.Register(context.Background(), registerRequest)
	if err != nil {
		return err
	}
	return nil
}

// ListAndWatch lists devices and update that list according to the health status
func (m *DevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	_ = s.Send(&pluginapi.ListAndWatchResponse{Devices: m.devs})
	for {
		select {
		case <-m.stop:
			return nil
		case d := <-m.health:
			// FIXME: there is no way to recover from the Unhealthy state.
			d.Health = pluginapi.Unhealthy
			_ = s.Send(&pluginapi.ListAndWatchResponse{Devices: m.devs})
		}
	}
}

func (m *DevicePlugin) unhealthy(dev *pluginapi.Device) {
	m.health <- dev
}

func (m *DevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (m *DevicePlugin) cleanup() error {
	if err := os.Remove(m.socket); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Serve starts the gRPC server and register the device plugin to Kubelet
func (m *DevicePlugin) Serve() error {
	err := m.Start()
	if err != nil {
		log.Infof("Could not start device plugin: %s", err)
		return err
	}
	log.Infoln("Starting to serve on", m.socket)

	err = m.Register(pluginapi.KubeletSocket, resourceName)
	if err != nil {
		log.Infof("Could not register device plugin: %s", err)
		_ = m.Stop()
		return err
	}
	log.Infoln("Registered device plugin with Kubelet")

	return nil
}

// Allocate which return list of devices.
func (m *DevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {

	var responses pluginapi.AllocateResponse

	devs := getDevices()
	for _, req := range r.ContainerRequests {
		response := &pluginapi.ContainerAllocateResponse{}
		for _, requestID := range req.DevicesIDs {
			var dev *pluginapi.Device
			for _, v := range devs {
				if v.ID == requestID {
					dev = v
				}
			}
			if dev == nil {
				return nil, fmt.Errorf("invalid allocation request with non-existing device %s", requestID)
			}

			if dev.Health != pluginapi.Healthy {
				return nil, fmt.Errorf("invalid allocation request with unhealthy device: %s", requestID)
			}

			// create fake device file
			fpath := filepath.Join("/tmp", dev.ID)

			// clean first
			if err := os.RemoveAll(fpath); err != nil {
				return nil, fmt.Errorf("failed to clean fake device file from previous run: %s", err)
			}

			f, err := os.Create(fpath)
			if err != nil && !os.IsExist(err) {
				return nil, fmt.Errorf("failed to create fake device file: %s", err)
			}

			f.Close()

			response.Mounts = append(response.Mounts, &pluginapi.Mount{
				ContainerPath: fpath,
				HostPath:      fpath,
			})
		}
		responses.ContainerResponses = append(responses.ContainerResponses, response)
	}

	return &responses, nil
}
