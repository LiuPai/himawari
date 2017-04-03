package himawari

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
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

type latestInfo struct {
	Date      string    `json:"date"`
	File      string    `json:"file"`
	Timestamp time.Time `json:"-"`
}

func (l *latestInfo) Name() string {
	return "latest image info"
}

func (l *latestInfo) Do() error {
	log.Println("Fetching latest image info")
	resp, err := http.Get(LatestInfoURL)
	if err != nil {
		return err
	}

	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(d, l)
	if err != nil {
		return err
	}
	l.Timestamp, err = time.Parse("2006-01-02 15:04:05", l.Date)
	if err != nil {
		return err
	}
	log.Printf("latest image timestamp: %s", l.Timestamp.Local())
	return nil
}

func (l *latestInfo) MaxFailTimes() int {
	return HTTPRetryTimes
}

type fetchSlice struct {
	level     int
	x         int
	y         int
	timestamp *time.Time
	result    draw.Image
}

func (f *fetchSlice) Name() string {
	return fmt.Sprintf("slice_%d_%d_%d", f.level, f.x, f.y)
}
func (f *fetchSlice) Do() error {
	url := fmt.Sprintf(TileImageURL,
		f.level,
		TileSize,
		f.timestamp.Format("2006/01/02/150405"),
		f.x, f.y)
	fileName := fmt.Sprintf("%s/%d_%d_%d.png",
		os.TempDir(), f.timestamp.Unix(), f.x, f.y)

	var slice image.Image
	// load file from cache
	if _, err := os.Stat(fileName); err == nil {
		d, err := ioutil.ReadFile(fileName)
		if err != nil {
			return err
		}

		// decode image
		img, _, err := image.Decode(bytes.NewReader(d))
		if err != nil {
			_ = os.Remove(fileName)
			return err
		}
		slice = img
	} else {
		// load image from web
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		d, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// decode image
		img, _, err := image.Decode(bytes.NewReader(d))
		if err != nil {
			return err
		}
		slice = img
		// save cache
		err = ioutil.WriteFile(fileName, d, 0666)
		if err != nil {
			return err
		}
	}

	draw.Draw(f.result,
		image.Rect(TileSize*f.x, TileSize*f.y,
			TileSize*(f.x+1), TileSize*(f.y+1)),
		slice, image.Pt(0, 0), draw.Src)
	return nil
}

func (f *fetchSlice) MaxFailTimes() int {
	return HTTPRetryTimes
}

func FetchImage(level int, timestamp *time.Time, cacheDir string) (string, error) {
	var (
		img = image.NewRGBA(
			image.Rect(0, 0, TileSize*level, TileSize*level))
		fetchSliceManager = NewManager()
		fileName          = fmt.Sprintf("%s/himawari_%d.png",
			cacheDir, timestamp.Unix())
	)
	// check if file exists
	if _, err := os.Stat(fileName); err == nil {
		return fileName, nil
	}
	for x := 0; x < level; x++ {
		for y := 0; y < level; y++ {
			fetchSliceManager.NewWork(&fetchSlice{
				level:     level,
				x:         x,
				y:         y,
				timestamp: timestamp,
				result:    img,
			})
		}
	}
	if !fetchSliceManager.Done() {
		return "", fmt.Errorf("fetch image %s failed", fileName)
	}
	// save image

	log.Printf("saving to %s ...", fileName)
	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()
	err = png.Encode(file, img)
	if err != nil {
		return "", err
	}
	// remove cache
	for x := 0; x < level; x++ {
		for y := 0; y < level; y++ {
			fileName := fmt.Sprintf("%s/%d_%d_%d.png",
				os.TempDir(), timestamp.Unix(), x, y)
			_ = os.Remove(fileName)
		}
	}

	return fileName, nil
}

// LatestImage fetch the latest image from himawari in level
func LatestTimestamp() (*time.Time, error) {
	var (
		latestInfoManager = NewManager()
		work              = new(latestInfo)
	)

	latestInfoManager.NewWork(work)
	if !latestInfoManager.Done() {
		return nil, fmt.Errorf("fetch latest image info failed")
	}
	return &work.Timestamp, nil
}
