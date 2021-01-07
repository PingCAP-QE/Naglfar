package tiup

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	// Our controller node and worker nodes share the same insecure_key path
	InsecureKeyPath = "/root/insecure_key"
	ContainerImage  = "docker.io/mahjonp/base-image:latest"
	SshTimeout      = 10 * time.Minute
)

type ErrClusterDuplicated struct {
	ClusterName string
}

type ErrClusterNotExist struct {
	ClusterName string
}

func (e ErrClusterDuplicated) Error() string {
	return fmt.Sprintf("cluster name %s is duplicated", e.ClusterName)
}

func (e ErrClusterNotExist) Error() string {
	return fmt.Sprintf("cluster name %s is not exist", e.ClusterName)
}

// IgnoreClusterDuplicated returns nil on ClusterDuplicated errors
// All other values that are not NotFound errors or nil are returned unmodified.
func IgnoreClusterDuplicated(err error) error {
	if _, ok := err.(ErrClusterDuplicated); ok {
		return nil
	}
	return err
}

// IgnoreClusterNotExist returns nil on IgnoreClusterNotExist errors
// All other values that are not NotFound errors or nil are returned unmodified.
func IgnoreClusterNotExist(err error) error {
	if _, ok := err.(ErrClusterNotExist); ok {
		return nil
	}
	return err
}

func InsecureIgnoreHostKey() ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}
}
