package main

import "net/http"

import (
	"io"
	"log"
	"os"
)

const (
	BaseDir = "/var/naglfar/lib/"
)

func upload(w http.ResponseWriter, r *http.Request) {
	fileName := r.Header.Get("fileName")
	err := os.Remove(BaseDir+fileName)
	if err != nil {
		log.Println(err)
	}
	f, err := os.OpenFile(BaseDir+fileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println(err)
	}
	io.Copy(f, r.Body)
	defer f.Close()
	log.Println("upload successfully", fileName)
}

func main() {
	http.HandleFunc("/upload", upload)
	err:=os.MkdirAll(BaseDir,os.ModePerm)
	if err!=nil{
		log.Println(err)
	}
	err = http.ListenAndServe(":6666", nil)
	if err != nil {
		log.Fatal("listenAndServe: ", err)
	}
}