package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/alecthomas/assert"
)

func TestUpload(t *testing.T) {
	content := []byte("ssss")

	req, err := http.NewRequest("POST", "http://localhost:6666/upload", strings.NewReader(string(content)))
	if err != nil {
		fmt.Println(err)
	}
	assert.Equal(t, nil, err)
	req.Header.Set("fileName", "haproxy.cfg")
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		fmt.Println(err)
	}
	assert.Equal(t, nil, err)
	defer resp.Body.Close()
}
