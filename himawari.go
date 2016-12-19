package himawari

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/valyala/fasthttp"
	"image"
	"image/draw"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const (
	// LatestInfoURL use to retrive latest image timestamp
	LatestInfoURL = "http://himawari8-dl.nict.go.jp/himawari8/img/D531106/latest.json"
	// TileImageURL the real image tail location
	TileImageURL = "http://himawari8.nict.go.jp/img/D531106/%dd/%d/%s_%d_%d.png"
	// TileSize each tile size in pixel
	TileSize = 550
	// HTTPRetryTimes retry times
	HTTPRetryTimes = 5
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
		result     image.Image
		err        error
	}
	work struct {
		*payload
		take  time.Time
		tried int
		retry chan bool
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
	// main loop
	for <-w.retry {
		w.tried++

		fileName := fmt.Sprintf("%s/%d_%d_%d.png",
			os.TempDir(), w.latestTime.Unix(), w.x, w.y)
		// load file from cache
		if _, err := os.Stat(fileName); err == nil {
			d, err := ioutil.ReadFile(fileName)
			if err != nil {
				w.err = err
				goto report
			}
			// decode image
			img, _, err := image.Decode(bytes.NewReader(d))
			if err != nil {
				w.err = err
				os.Remove(fileName)
				goto report
			}
			w.payload.result = img
		} else {
			// load image from web
			code, d, err :=
				fasthttp.Get(nil, w.url)

			if err == nil && code != 200 {
				err = fmt.Errorf("status code %d", code)
			}
			if err != nil {
				w.err = err
				goto report
			}
			if w.err != nil {
				goto report
			}
			// decode image
			img, _, err := image.Decode(bytes.NewReader(d))
			if err != nil {
				w.err = err
				goto report
			}
			w.payload.result = img
			// save to cache
			ioutil.WriteFile(fileName, d, 0666)
		}
	report:
		// report result
		reportChan <- w
		// avoid quick retry
		time.Sleep(time.Second * 1)
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
			w := &work{
                                payload: &payload{
                                        level:      level,
                                        x:          x,
                                        y:          y,
                                        latestTime: date,
                                },
                                retry: make(chan bool),
                        }
			workChan <- w
			w.retry <- true
		}
	}

	for {
		r := <-reportChan
		if r.err != nil {
			if r.tried < HTTPRetryTimes {
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

		draw.Draw(img,
			image.Rect(TileSize*r.x, TileSize*r.y,
				TileSize*(r.x+1), TileSize*(r.y+1)),
			r.result, image.Pt(0, 0), draw.Src)

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
	for (err != nil || code != 200) && try <= HTTPRetryTimes {
		try++
		code, body, err = fasthttp.Get(nil, LatestInfoURL)
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
