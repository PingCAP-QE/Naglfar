package haproxy

import (
	"github.com/PingCAP-QE/Naglfar/pkg/container"
	"net/http"
	"strconv"
	"strings"
)

const (
	BaseImageName = "docker.io/library/haproxy:"
	SourceMount = "/var/naglfar/lib/haproxy.cfg"
	TargetMount = "/usr/local/etc/haproxy/haproxy.cfg"
	FileName = "haproxy.cfg"
)

func TransferBackend(backend string) string{
	return backend
}

func WriteConfigToMachine(machine string,name string,config string) error{
	config = "global\n\tdaemon\n\tmaxconn 256\n \ndefaults\n\tmode tcp\n\ttimeout connect 5000ms\n\ttimeout client 6000000ms\n\ttimeout server 6000000ms\n\nfrontend http-in\n\tbind *:9999\n\tdefault_backend tidbs\n\nbackend tidbs\n\tserver server1 172.18.0.3:4000 maxconn 64 \n\tserver server2 172.18.0.4:4000 maxconn 64\n\tserver server3 172.18.0.5:4000 maxconn 64\n"
	req, err := http.NewRequest("POST", "http://"+machine+":"+strconv.Itoa(container.UploadPort)+"/upload" , strings.NewReader(config))
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

