package rw

import (
	"io/ioutil"
	"log"
	"os"
	"path"
)

// Putfile - write keys data into the path.
func Putfile(path string, queue <-chan map[string]string) {
	for batch := range queue {
		for key, value := range batch {
			file, err := os.Create(path + `/` + key)
			if err != nil {
				log.Printf("error: Putfile() %s", err)
			}
			defer file.Close()

			file.Write(([]byte)(value))
		}
	}
}

// Pullfile Scan path and queue source keys/values from file.
func Pullfile(pathname string, queue chan<- map[string]string, match string) {
	log.Printf("info: target path=%s", pathname)

	files, err := ioutil.ReadDir(pathname)
	if err != nil {
		log.Printf("error: Pullfile ReadDir %s", err)
	}

	for _, filename := range files {
		batch := make(map[string]string)

		if filename.IsDir() {
			log.Printf("info: %s is Directory.", filename.Name())
			continue
		}

		matched, _ := path.Match(match, filename.Name())

		if matched == true {
			buffer, err := ioutil.ReadFile(pathname + `/` + filename.Name())
			if err != nil {
				log.Printf("error: Pullfile ReadFile %s", err)
			}
			batch[filename.Name()] = string(buffer)
			log.Printf("info: filename = %s, data = %#v", filename.Name(), buffer)
		}
		queue <- batch
	}
	log.Printf("info: last queue data=%#v", queue)
	close(queue)

}
