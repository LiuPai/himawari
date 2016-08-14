package main

import (
	"desktop"
	"himawari"
	"flag"
	"log"
	"os"
	"time"
	"image/png"
)


var (
	level = flag.Int("level", 4,
		"Image quality and size choose one of [4, 8, 16, 20]")
	timeout = flag.Int("timeout", 30, "HTTP request timeout in second")
	retry = flag.Int("retry", 5, "HTTP request retry times")
	output = flag.String("output", os.TempDir()+"/himiwari.png",
		"Path to the output file")
	wallpaper = flag.Bool("wallpaper", true,
		"If set desktop wallpaper to latest himiwary image")
)


func main() {
	flag.Parse()
	switch *level {
	case 4, 8, 16, 20:
	default:
		log.Fatalf("unsupport level value: %d", *level)
	}
	
	file, err := os.OpenFile(*output, os.O_CREATE|os.O_WRONLY, 0666)
        if err != nil {
                log.Fatalf("image output path open failed, err: %v", err)
        }
        defer file.Close()
	
        h := himawari.New(*retry, time.Duration(*timeout) * time.Second)
	img, err := h.LatestImage(*level)
	if err != nil {
		os.Exit(1)
	}
	
	log.Printf("Saving to %s ...", file.Name())
        err = png.Encode(file, img)
        if err != nil {
                log.Fatalf("store image to %s failed, err: %v",
                        file.Name(), err)
        }
	
	if *wallpaper {
		if desktop.SetBackground(*output) {
			log.Printf("Done!")
		} else {
			log.Fatalf("Your desktop environment %s is not supported.",
				desktop.Environment())
		}
	}
}
