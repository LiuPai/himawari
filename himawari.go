package himawari

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
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
	// CoastlineURL the coast line of image
	CoastlineURL = "http://himawari8-dl.nict.go.jp/himawari8/img/D531106/%dd/%d/coastline/ff0000_%d_%d.png"
	// TileImageURL the real image tail location
	TileImageURL = "http://himawari8.nict.go.jp/img/D531106/%dd/%d/%s_%d_%d.png"
	// TileSize each tile size in pixel
	TileSize = 550
	// HTTPRetryTimes retry times
	HTTPRetryTimes = 5
)

var (
	coastline image.Image
)

type latestInfo struct {
	Date      string    `json:"date"`
	File      string    `json:"file"`
	Timestamp time.Time `json:"-"`
	client    *http.Client
}

func (l *latestInfo) Name() string {
	return "latest image info"
}

func (l *latestInfo) Do() error {
	log.Println("Fetching latest image info")
	resp, err := l.client.Get(LatestInfoURL)
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

type fetchCoastlineSlice struct {
	level  int
	x      int
	y      int
	result draw.Image
	client *http.Client
}

func (f *fetchCoastlineSlice) Name() string {
	return fmt.Sprintf("coastline_%d_%d_%d", f.level, f.x, f.y)
}
func (f *fetchCoastlineSlice) Do() error {
	url := fmt.Sprintf(CoastlineURL,
		f.level,
		TileSize,
		f.x, f.y)
	fileName := fmt.Sprintf("%s/coastline_%d_%d_%d_%d.png",
		os.TempDir(), f.level, TileSize, f.x, f.y)
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
		resp, err := f.client.Get(url)
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
func (f *fetchCoastlineSlice) MaxFailTimes() int {
	return HTTPRetryTimes
}

type fetchSlice struct {
	level     int
	x         int
	y         int
	timestamp *time.Time
	result    draw.Image
	client    *http.Client
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
	fileName := fmt.Sprintf("%s/%d_%d_%d_%d_%d.png",
		os.TempDir(), f.timestamp.Unix(), f.level, TileSize, f.x, f.y)
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
		resp, err := f.client.Get(url)
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
		fileName          = fmt.Sprintf("%s/himawari_%d_%d_%d.png",
			cacheDir, level, TileSize, timestamp.Unix())
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
				client:    new(http.Client),
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
			fileName := fmt.Sprintf("%s/%d_%d_%d_%d_%d.png",
				os.TempDir(), timestamp.Unix(), level, TileSize, x, y)
			_ = os.Remove(fileName)
		}
	}

	return fileName, nil
}

// FetchCoastline fetch coastline and store to cache file
func FetchCoastline(level int, c color.Color, cacheDir string) (string, error) {
	var (
		img = image.NewRGBA(
			image.Rect(0, 0, TileSize*level, TileSize*level))
		fetchSliceManager = NewManager()
		fileName          = fmt.Sprintf("%s/himawari_coastline_%d_%d.png",
			cacheDir, level, TileSize)
	)
	defer func() {
		// set coastline color
		if c != nil {
			for x := 0; x < coastline.Bounds().Dx(); x++ {
				for y := 0; y < coastline.Bounds().Dy(); y++ {
					if r, _, _, _ := coastline.At(x, y).RGBA(); r != 0 {
						coastline.(draw.Image).Set(x, y, c)
					}
				}
			}
		}
	}()
	// check if file exists
	if _, err := os.Stat(fileName); err == nil {
		d, err := ioutil.ReadFile(fileName)
		if err != nil {
			return "", err
		}
		coastline, _, err = image.Decode(bytes.NewReader(d))
		if err != nil {
			_ = os.Remove(fileName)
			return "", err
		}

		return fileName, nil
	}
	for x := 0; x < level; x++ {
		for y := 0; y < level; y++ {
			fetchSliceManager.NewWork(&fetchCoastlineSlice{
				level:  level,
				x:      x,
				y:      y,
				result: img,
				client: new(http.Client),
			})
		}
	}
	if !fetchSliceManager.Done() {
		return "", fmt.Errorf("fetch image %s failed", fileName)
	}
	coastline = img
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
			fileName := fmt.Sprintf("%s/coastline_%d_%d_%d_%d.png",
				os.TempDir(), level, TileSize, x, y)
			_ = os.Remove(fileName)
		}
	}

	return fileName, nil
}

// LatestImage fetch the latest image from himawari in level
func LatestTimestamp() (*time.Time, error) {
	var (
		latestInfoManager = NewManager()
		work              = &latestInfo{
			client: new(http.Client),
		}
	)

	latestInfoManager.NewWork(work)
	if !latestInfoManager.Done() {
		return nil, fmt.Errorf("fetch latest image info failed")
	}
	return &work.Timestamp, nil
}

func MergeCoastline(imgFile string) (fileName string, err error) {
	var (
		imgData []byte
		img     image.Image
	)
	// load file from cache
	_, err = os.Stat(imgFile)
	if err != nil {
		return
	}
	imgData, err = ioutil.ReadFile(imgFile)
	if err != nil {
		return
	}
	img, _, err = image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return
	}

	// set coastline color
	draw.Draw(img.(draw.Image), img.Bounds(), coastline, image.ZP, draw.Over)

	// encode
	var (
		file *os.File
	)
	fileName = imgFile[:len(imgFile)-4] + fmt.Sprintf("_coast.png")
	file, err = os.Create(fileName)
	if err != nil {
		return
	}
	defer file.Close()
	err = png.Encode(file, img)
	if err != nil {
		return
	}
	return fileName, nil
}
