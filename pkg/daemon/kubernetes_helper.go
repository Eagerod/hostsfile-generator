package daemon

import (
	"fmt"
	"log"
	"os"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func WriteHostsFileAndRestartPihole(config *rest.Config, clientset *kubernetes.Clientset, podName string, hostsfile string) error {
	log.Println("Updating kube.list in pod:", podName)
	if err := CopyFileToPod(config, clientset, podName, "/etc/pihole/kube.list", hostsfile); err != nil {
		return err
	}

	log.Println("Restarting DNS service in pod:", podName)
	if err := ExecInPod(config, clientset, podName, []string{"pihole", "restartdns"}); err != nil {
		return err
	}

	log.Println("Successfully restarted DNS service in pod:", podName)
	return nil
}

func CopyFileToPod(config *rest.Config, clientset *kubernetes.Clientset, podName string, filepath string, contents string) error {
	// There's certainly a more correct way of doing this, but that's a lot of
	//   extra code.
	script := fmt.Sprintf("cat <<EOF > %s\n%s\nEOF", filepath, contents)
	return ExecInPod(config, clientset, podName, []string{"sh", "-c", script})
}

func ExecInPod(config *rest.Config, clientset *kubernetes.Clientset, podName string, command []string) error {
	api := clientset.CoreV1()

	execResource := api.RESTClient().Post().Resource("pods").Name(podName).
		Namespace("default").SubResource("exec").Param("container", "pihole")

	podExecOptions := &v1.PodExecOptions{
		Command: command,
		Stdin:   true,
		Stdout:  true,
		Stderr:  true,
		TTY:     true,
	}

	execResource.VersionedParams(
		podExecOptions,
		scheme.ParameterCodec,
	)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", execResource.URL())
	if err != nil {
		return err
	}

	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}
