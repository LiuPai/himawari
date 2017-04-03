package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/LiuPai/himawari"
)

var (
	level = flag.Int("level", 4,
		"Image quality and size choose one of [4, 8, 16, 20]")
	cache = flag.String("cache", os.TempDir(),
		"Path to the cache file directory")
	defaultImageFile = fmt.Sprintf("%s/himawari.png", os.TempDir())
	output           = flag.String("output", defaultImageFile,
		"The link of current himawari image")
	daemon = flag.Bool("daemon", false,
		"Run himawari as daemon")
	tick = flag.Uint("tick", 300,
		"Duration to check himawari latest timestamp in seconds")
	pidFile = flag.String("pid", "",
		"Himawari unix like system pid file")
	latestTimestamp *time.Time
)

func checkLatestImage() (err error) {
	latestTimestamp, err = himawari.LatestTimestamp()
	if err != nil {
		return err
	}

	imageFile, err := himawari.FetchImage(*level,
		latestTimestamp,
		*cache)
	if err != nil {
		return err
	}
	_ = os.Remove(*output)
	err = os.Symlink(imageFile, *output)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	switch *level {
	case 4, 8, 16, 20:
	default:
		log.Fatalf("unsupport level value: %d", *level)
	}

	// oneshot fetch current himawari image
	if !*daemon {
		err := checkLatestImage()
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// store pid file
	if *pidFile != "" {
		err := ioutil.WriteFile(*pidFile,
			[]byte(fmt.Sprintf("%d", os.Getpid())),
			0644)
		if err != nil {
			log.Fatalf("failed to write pid file %s, err: %v",
				*pidFile, err)
		}
	}

	// daemon ticker
	ticker := time.NewTicker(time.Second * time.Duration(*tick))
	log.Println(*level, *tick)
	err := checkLatestImage()
	if err != nil {
		log.Print(err)
	}
	// main loop
	for _ = range ticker.C {
		timestamp, err := himawari.LatestTimestamp()
		if err != nil {
			log.Print(err)
		}
		// check latest timestamp
		if timestamp.Unix() != latestTimestamp.Unix() {
			latestTimestamp = timestamp
			imageFile, err := himawari.FetchImage(*level,
				latestTimestamp,
				*output)
			if err != nil {
				log.Print(err)
			}
			_ = os.Remove(*output)
			err = os.Symlink(imageFile, *output)
			if err != nil {
				log.Print(err)
			}
		}
	}
}