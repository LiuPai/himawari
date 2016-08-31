package himawari

import (
	"bytes"
	"config"
	"encoding/json"
	"fmt"
	"github.com/valyala/fasthttp"
	"image"
	"image/draw"
	"log"
	"time"
)

const (
	// LatestInfoURL use to retrive latest image timestamp
	LatestInfoURL = "http://himawari8-dl.nict.go.jp/himawari8/img/D531106/latest.json"
	// TileImageURL the real image tail location
	TileImageURL = "http://himawari8.nict.go.jp/img/D531106/%dd/%d/%s_%d_%d.png"
	// TileSize each tile size in pixel
	TileSize = 550
)

type (
	// JSONTime time store in JSON
	JSONTime   struct{ time.Time }
	latestInfo struct {
		Date JSONTime `json:"date"`
		File string   `json:"file"`
	}
	payload struct {
		level      int
		x          int
		y          int
		latestTime time.Time
		url        string
		result     []byte
		err        error
	}
	work struct {
		*payload
		take    time.Time
		useTime time.Duration
		tried   int
		retry   chan bool
	}
)

// UnmarshalJSON unmarshal date in JSON format
func (t *JSONTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	tm, err := time.Parse("2006-01-02 15:04:05", s)
	t.Time = tm
	return err
}

func worker(workChan <-chan *work, reportChan chan<- *work) {
	// take a job
	w := <-workChan
	w.take = time.Now()

	w.url = fmt.Sprintf(TileImageURL,
		w.level,
		TileSize,
		w.latestTime.Format("2006/01/02/150405"),
		w.x, w.y)

	var (
		start time.Time
		end   time.Time
		code  int
		retry = true
	)

	// main loop
	for retry {
		w.tried++

		start = time.Now()

		code, w.result, w.err =
			fasthttp.GetTimeout(nil, w.url, config.HTTPTimesout)

		end = time.Now()
		w.useTime = end.Sub(start)

		if w.err == nil && code != 200 {
			w.err = fmt.Errorf("status code %d", code)
		}
		// report result
		reportChan <- w
		// check if need redo the work
		retry = <-w.retry
		// avoid quick retry
		time.Sleep(time.Second * 5)
	}
}

func buildImage(level int, date time.Time) (image.Image, error) {
	var (
		tileNumber = level * level
		img        = image.NewRGBA(
			image.Rect(0, 0, TileSize*level, TileSize*level))
		workChan   = make(chan *work)
		reportChan = make(chan *work)
		counter    int
	)
	for x := 0; x < level; x++ {
		for y := 0; y < level; y++ {
			go worker(workChan, reportChan)
			workChan <- &work{
				payload: &payload{
					level:      level,
					x:          x,
					y:          y,
					latestTime: date,
				},
				retry: make(chan bool),
			}
		}
	}

	for {
		r := <-reportChan
		// TODO: add network analysis code to determine if should retry
		if r.err != nil {
			if r.tried < config.HTTPRetryTimes {
				r.retry <- true
				continue
			} else {
				r.retry <- false
				println()
				log.Printf("download %s failed after %d tried, err: %v",
					r.url, r.tried, r.err)
				return nil, r.err
			}
		}

		tileImg, _, err := image.Decode(bytes.NewReader(r.result))
		if err != nil {
			log.Printf("image from %s decode failed, err: %v",
				r.url, err)
		}
		draw.Draw(img,
			image.Rect(TileSize*r.x, TileSize*r.y,
				TileSize*(r.x+1), TileSize*(r.y+1)),
			tileImg, image.Pt(0, 0), draw.Src)

		counter++
		fmt.Printf("\rDownloading tiles: %d/%d completed",
			counter, tileNumber)
		if counter == tileNumber {
			fmt.Println()
			break
		}
	}
	return img, nil
}

// LatestImage fetch the latest image from himawari in level
func LatestImage(level int) (image.Image, error) {
	var (
		err    error
		code   int
		try    int
		body   []byte
		latest = new(latestInfo)
	)
	log.Println("Fetching latest image info")
	for (err != nil || code != 200) && try <= config.HTTPRetryTimes {
		try++
		code, body, err = fasthttp.GetTimeout(nil, LatestInfoURL, config.HTTPTimesout)
	}
	if err != nil || code != 200 {
		log.Printf("fetch latest image info from %s failed, code: %d err: %v",
			LatestInfoURL, code, err)
		return nil, fmt.Errorf("fetch latest info failed")
	}
	err = json.Unmarshal(body, latest)
	if err != nil {
		log.Printf("unmarshal JSON %s failed, err: %v",
			string(body), err)
		return nil, fmt.Errorf("decode latest info failed")
	}
	log.Printf("Latest version: %v", latest.Date.Local())
	return buildImage(level, time.Time(latest.Date.Time))
}
