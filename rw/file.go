package rw

import (
	"fmt"
	"io/ioutil"
	//"log"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

// Putfile - write keys data into the path.
func Putfile(path string, queue <-chan map[string][]byte) {
	for batch := range queue {
		for key, value := range batch {
			log.Debugf("dump filename: %s", path+`/`+key)
			file, err := os.Create(path + `/` + key)
			if err != nil {
				log.Errorf("Putfile %s, err:%s", path+`/`+key, err)
			}
			defer file.Close()

			file.Write(([]byte)(value))
		}
	}
}

// Pullfile Scan path and queue source keys/values from file.
func Pullfile(pathname string, queue chan<- map[string][]byte, match string) {

	files, err := ioutil.ReadDir(pathname)
	if err != nil {
		log.Errorf("Pullfile ReadDir %s", err)
	}

	for _, filename := range files {
		batch := make(map[string][]byte)

		if filename.IsDir() {
			log.Debugf("%s is Directory.", filename.Name())
			continue
		}

		matched, _ := path.Match(match, filename.Name())

		if matched == true {
			buffer, err := ioutil.ReadFile(pathname + `/` + filename.Name())
			if err != nil {
				log.Errorf("Pullfile readfile %s, err: %s", filename.Name(), err)
			}
			batch[filename.Name()] = buffer
			log.Debugf("read dumpfilename: %s", filename.Name())
		}

		if log.GetLevel() != 5 {
			fmt.Print(".")
		}

		queue <- batch
	}
	close(queue)

}
