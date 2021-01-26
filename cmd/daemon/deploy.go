package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

const (
	BaseDir = "/var/naglfar/lib/"
)

func upload(w http.ResponseWriter, r *http.Request) {
	fileName := r.Header.Get("fileName")
	err := os.Remove(BaseDir + fileName)
	if err != nil {
		log.Println(err)
	}
	f, err := os.OpenFile(BaseDir+fileName, os.O_WRONLY|os.O_CREATE, 0600|0066)
	if err != nil {
		log.Println(err)
	}
	io.Copy(f, r.Body)
	defer f.Close()
	log.Println("upload successfully", fileName)
}

func delete(w http.ResponseWriter, r *http.Request) {
	fileName := r.Header.Get("fileName")
	err := os.Remove(BaseDir + fileName)
	if err != nil {
		log.Println(err)
	}
	log.Println("delete successfully", fileName)
}

func main() {
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/delete", delete)
	err := os.MkdirAll(BaseDir, os.ModePerm)
	if err != nil {
		log.Println(err)
	}
	err = http.ListenAndServe(":6666", nil)
	if err != nil {
		log.Fatal("listenAndServe: ", err)
	}
}
