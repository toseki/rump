package main

import (
	"flag"
	"fmt"
	"time"
	//"log"
	"os"

	"github.com/garyburd/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/toseki/rump/rw"
)

var version string // set by Makefile

// Report all errors to stdout.
func handle(err error) {
	if err != nil && err != redis.ErrNil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Scan and queue source keys.
func get(conn redis.Conn, queue chan<- map[string][]byte, match string, count int) {
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
			log.Debugf("read redis key dump: %s", key)
			conn.Send("DUMP", key)
		}
		dumps, err := redis.ByteSlices(conn.Do(""))
		handle(err)

		// Build batch map.
		batch := make(map[string][]byte)
		for i, _ := range keys {
			batch[keys[i]] = dumps[i]
		}

		if log.GetLevel() != 5 {
			fmt.Print(".")
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

		// queue current batch.
		queue <- batch
	}
}

// Restore a batch of keys on destination.
func put(conn redis.Conn, queue <-chan map[string][]byte, redisTTL int64) {
	for batch := range queue {
		for key, value := range batch {
			log.Debugf("restore redis key: %s", key)
			conn.Send("RESTORE", key, redisTTL, value)
			rc, err := conn.Do("")
			handle(err)
			if rc != nil {
				log.Debugf("restore redis key response: %s", rc)
			}
		}
	}
}

func main() {
	var showversion = false

	from := flag.String("from", "", "SRC Redis Server. ex) redis://127.0.0.1:6379/0")
	to := flag.String("to", "", "DEST Redis Server. ex) redis://127.0.0.1:6379/0")
	match := flag.String("match", "*", "key match. ex) h*llo")
	count := flag.Int("count", 10, "ex) 100")
	path := flag.String("path", "", "dumpfile path. ex) ./dumpfiles/")
	loglevel := flag.Int("loglevel", 4, "ex) debug=5, info=4, warning=3, error=2, fatal=1, panic=0")
	TTLHour := flag.Duration("ttl", time.Hour*24*180, "Redis Expire TTL. ex) 24h")

	flag.BoolVar(&showversion, "version", false, "show version")

	flag.Parse()

	log.SetLevel(log.Level(uint8(*loglevel)))
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: "2006-01-02T15:04:05.000000"})

	if showversion {
		fmt.Println("version:", version)
		os.Exit(0)
	}

	// Channel where batches of keys will pass.
	queue := make(chan map[string][]byte, *count*30)

	// from redis to filedump
	if *to == "" && *path != "" {
		log.Infof("redis -> files mode.")
		log.Infof("redis:%s, filepath:%s", *from, *path)

		source, err := redis.DialURL(*from)
		handle(err)
		defer source.Close()

		// Scan and send to queue.
		go get(source, queue, *match, *count)

		// each keys data dump to files.
		rw.Putfile(*path, queue)

		fmt.Println("\nProcessing from redis to filedumps was completed.")

	} else if *from == "" && *path != "" { // from filedump to redis
		log.Infof("files -> redis mode.")
		log.Infof("filepath:%s, redis:%s", *path, *to)
		redisTTL := int64(*TTLHour) / int64(time.Millisecond)
		log.Infof("Redis TTL setting: %d sec", redisTTL/1000)

		// keys data dump path scan and send to queue.
		go rw.Pullfile(*path, queue, *match)

		destination, err := redis.DialURL(*to)
		handle(err)
		defer destination.Close()

		// Restore keys as they come into queue.
		put(destination, queue, redisTTL)

		fmt.Println("\nProcessing from filedumps to redis was completed.")

	} else if *from != "" && *to != "" && *path == "" { // from redis to redis
		log.Infof("redis -> redis mode.")
		log.Infof("redis:%s, redis:%s", *from, *to)
		redisTTL := int64(*TTLHour) / int64(time.Millisecond)
		log.Infof("Redis TTL setting: %d sec", redisTTL/1000)

		source, err := redis.DialURL(*from)
		handle(err)
		destination, err := redis.DialURL(*to)
		handle(err)
		defer source.Close()
		defer destination.Close()

		// Scan and send to queue.
		go get(source, queue, *match, *count)

		// Restore keys as they come into queue.
		put(destination, queue, redisTTL)

		fmt.Println("\nProcessing from redis(SRC) to redis(DEST) was completed.")
	} else {
		usagetext := `Usage Example
 ex.1) redis to filedump
	rump -from redis://<redis server>:<port>/0 -path ./dumpfiles/ -match "mat*ch:strings"

 ex.2) filedump to redis
	rump -path ./dumpfiles/ -to redis://<redis server>:<port>/0 -match "mat*ch:strings" -ttl 240h

 ex.3) redis to redis
	rump -from redis://<redis server>:<port>/0 -to redis://<redis server>:<port>/0 -match "mat*ch:strings" -ttl 240h`
		fmt.Println(usagetext)
	}

}
