package device_example

import (
	"context"
	log "github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	nodeutil "k8s.io/component-helpers/node/util"
	"k8s.io/klog/v2"
	"os"
)

var (
	clientset *kubernetes.Clientset
	nodeName  string
)

// func init() {
// 	kubeInit()
// }

func kubeInit() {
	kubeconfigFile := os.Getenv("KUBECONFIG")
	var err error
	var config *rest.Config

	if _, err = os.Stat(kubeconfigFile); err != nil {
		log.V(5).Infof("kubeconfig %s failed to find due to %v", kubeconfigFile, err)
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Failed due to %v", err)
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigFile)
		if err != nil {
			log.Fatalf("Failed due to %v", err)
		}
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed due to %v", err)
	}

	nodeName = os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatalln("Please set env NODE_NAME")
	}

}

func patchDeviceCount(count int) error {
	node, err := clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	klog.Info("patchDeviceCount:", node.Name, node.UID, count)

	if val, ok := node.Status.Capacity[resourceCount]; ok {
		if val.Value() == int64(count) {
			klog.Infof("No need to update Capacity %s", resourceCount)
			return nil
		}
	}

	newNode := node.DeepCopy()
	newNode.Status.Capacity[resourceCount] = *resource.NewQuantity(int64(count), resource.DecimalSI)
	newNode.Status.Allocatable[resourceCount] = *resource.NewQuantity(int64(count), resource.DecimalSI)
	_, _, err = nodeutil.PatchNodeStatus(clientset.CoreV1(), types.NodeName(nodeName), node, newNode)
	if err != nil {
		klog.Infof("Failed to update Capacity %s.", resourceCount)
	} else {
		klog.Infof("Updated Capacity %s successfully.", resourceCount)
	}
	return err
}
