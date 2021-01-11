package haproxy

import (
	"github.com/PingCAP-QE/Naglfar/pkg/container"
	"net/http"
	"strconv"
	"strings"
)

const (
	BaseImageName = "docker.io/library/haproxy:"
	MountDir = "/var/naglfar/lib"
)

func TransferBackend(backend string) string{
	return backend
}

func WriteConfigToMachine(machine string,name string,config string) error{
	config=TransferBackend(config)

	req, err := http.NewRequest("POST", "http://"+machine+":+"+strconv.Itoa(container.UploadPort)+"/upload" , strings.NewReader(config))
	if err!= nil {
		return err
	}
	req.Header.Set("fileName", name)
	resp, err := (&http.Client{}).Do(req)
	if err!= nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

