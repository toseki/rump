package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/garyburd/redigo/redis"
	"github.com/toseki/rump/rw"
)

// Report all errors to stdout.
func handle(err error) {
	if err != nil && err != redis.ErrNil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Scan and queue source keys.
func get(conn redis.Conn, queue chan<- map[string]string, match string, count int) {
	var (
		cursor int64
		keys   []string
	)

	for {
		// Scan a batch of keys.
		values, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", match, "COUNT", count))
		handle(err)
		values, err = redis.Scan(values, &cursor, &keys)
		handle(err)

		// Get pipelined dumps.
		for _, key := range keys {
			conn.Send("DUMP", key)
		}
		dumps, err := redis.Strings(conn.Do(""))
		handle(err)

		// Build batch map.
		batch := make(map[string]string)
		for i, _ := range keys {
			batch[keys[i]] = dumps[i]
		}

		// Last iteration of scan.
		if cursor == 0 {
			// queue last batch.
			select {
			case queue <- batch:
			}
			close(queue)
			break
		}

		fmt.Printf(">")
		// queue current batch.
		queue <- batch
	}
}

// Restore a batch of keys on destination.
func put(conn redis.Conn, queue <-chan map[string]string) {
	for batch := range queue {
		log.Printf("info: put redis batch data = %#v", batch)
		for key, value := range batch {
			conn.Send("RESTORE", key, "0", value)
		}
		_, err := conn.Do("")
		handle(err)

		fmt.Printf(".")
	}
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	from := flag.String("from", "", "SRC Redis Server. ex) redis://127.0.0.1:6379/0")
	to := flag.String("to", "", "DEST Redis Server. ex) redis://127.0.0.1:6379/0")
	match := flag.String("match", "*", "key match. ex) h*llo")
	count := flag.Int("count", 10, "ex) 100")
	path := flag.String("path", "", "dumpfile path. ex) ./dumpfiles/")

	flag.Parse()

	// Channel where batches of keys will pass.
	queue := make(chan map[string]string, *count*10)

	if *to == "" && *path != "" {
		fmt.Println("redis -> files mode.")
		source, err := redis.DialURL(*from)
		handle(err)
		defer source.Close()

		// Scan and send to queue.
		go get(source, queue, *match, *count)
		rw.Putfile(*path, queue)

	} else if *from == "" && *path != "" {
		// DirScan and send to queue.
		go rw.Pullfile(*path, queue, *match)

		fmt.Println("files -> redis mode.")
		destination, err := redis.DialURL(*to)
		handle(err)
		defer destination.Close()

		// Restore keys as they come into queue.
		put(destination, queue)

	}

	fmt.Println("Sync done.")
}
