package himawari

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/valyala/fasthttp"
	"image"
	"image/draw"
	"log"
	"time"
)

const (
	latestInfoURL = "http://himawari8-dl.nict.go.jp/himawari8/img/D531106/latest.json"
	tileImageURL  = "http://himawari8.nict.go.jp/img/D531106/%dd/%d/%s_%d_%d.png"
	tileWidth     = 550
	tileHeight    = 550
)

type (
	// JSONTime time store in JSON
	jsonTime   struct{ time.Time }
	latestInfo struct {
		Date jsonTime `json:"date"`
		File string   `json:"file"`
	}
	tile struct {
		x, y int
		data []byte
	}
	// Himawari Image downloader
	Himawari struct {
		retry   int
		timeout time.Duration
	}
)

// UnmarshalJSON unmarshal date in JSON format
func (t *jsonTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	tm, err := time.Parse("2006-01-02 15:04:05", s)
	t.Time = tm
	return err
}

// New initialize himiwari downloader parameters
func New(retry int, timeout time.Duration) *Himawari {
	return &Himawari{
		retry:   retry,
		timeout: timeout,
	}
}

func (h *Himawari)downloader(level, x, y int, latestTime time.Time,
	tileChan chan<- *tile) {
	url := fmt.Sprintf(tileImageURL, level, tileWidth,
		latestTime.Format("2006/01/02/150405"), x, y)
	var (
		err  error
		code int
		try  int
		body []byte
	)
	for (err != nil || code != 200) && try <= h.retry {
		try++
		code, body, err = fasthttp.GetTimeout(nil, url, h.timeout)
		if err != nil || code != 200 {
			continue
		}
	}
	if err != nil || code != 200 {
		log.Printf("\nfetch image from %s failed, code: %d err: %v",
			url, code, err)
		close(tileChan)
	}
	tileChan <- &tile{
                x:    x,
                y:    y,
                data: body,
        }
}

func (h *Himawari)buildImage(level int, date time.Time) (image.Image, error) {
	var (
		tileNumber = level * level
		img = image.NewRGBA(
			image.Rect(0, 0, tileWidth*level, tileHeight*level))
		tileChan = make(chan *tile)
		counter  int
	)
	
	for x := 0; x < level; x++ {
		for y := 0; y < level; y++ {
			go h.downloader(level, x, y, date, tileChan)
		}
	}

	for {
		t, ok := <-tileChan
		if !ok {
			return nil, fmt.Errorf("fetch image fail")
		}
		tileImg, _, err := image.Decode(bytes.NewReader(t.data))
		if err != nil {
			log.Printf("image tile[%d][%d] decode failed, err: %v",
				t.x, t.y, err)
			return nil, err
		}
		draw.Draw(img,
			image.Rect(tileWidth*t.x, tileHeight*t.y,
				tileWidth*(t.x+1), tileHeight*(t.y+1)),
			tileImg, image.Pt(0, 0), draw.Src)
		counter++
		fmt.Printf("\rDownloading tiles: %d/%d completed",
			counter, tileNumber)
		if counter == tileNumber {
			fmt.Println()
			break
		}
	}
	close(tileChan)
	return img, nil
}

// LatestImage fetch the latest image from himawari in level
func (h *Himawari)LatestImage(level int) (image.Image, error) {
	var (
		err error
		code   int
		try    int
		body   []byte
		latest = new(latestInfo)
	)
	log.Println("Fetching latest image info")
	for (err != nil || code != 200) && try <= h.retry {
		try++
		code, body, err = fasthttp.GetTimeout(nil, latestInfoURL, h.timeout)
		if err != nil || code != 200 {
			continue
		}
	}
	if err != nil || code != 200 {
		log.Printf("fetch latest image info from %s failed, code: %d err: %v",
			latestInfoURL, code, err)
		return nil, fmt.Errorf("fetch latest info failed")
	}
	err = json.Unmarshal(body, latest)
	if err != nil {
		log.Printf("unmarshal JSON %s failed, err: %v",
			string(body), err)
		return nil, fmt.Errorf("decode latest info failed")
	}
	log.Printf("Latest version: %v", latest.Date.Local())
	return h.buildImage(level, time.Time(latest.Date.Time))
}
