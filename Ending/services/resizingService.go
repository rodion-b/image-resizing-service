package services

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"images-resizing-service/config"
	"images-resizing-service/models"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	jpgresize "github.com/nfnt/resize"
)

const (
	success    = "success"
	inProgress = "inProgress"
	failure    = "failure"
)

type ResizingService struct {
	Cache            *lru.Cache
	ImagesInProgress sync.Map
}

func NewResizingService() *ResizingService {
	return &ResizingService{}
}

// Non-blocking implementation
func (s *ResizingService) SubmitForAsyncProcessing(request models.ResizeRequest) []models.ResizeResult {

	ids := []string{}
	results := make([]models.ResizeResult, 0, len(request.URLs))

	//for each ID starting goruitine
	for _, url := range request.URLs {
		result := models.ResizeResult{}
		imageId := s.genID(url)
		ids = append(ids, imageId)
		key := "/v1/image/" + imageId + ".jpeg"
		newURL := config.Proto + config.Hostport + key

		//we skip the processing if image with such key is already in cash
		if !s.Cache.Contains(key) {
			go s.startResizingProcess(url, request.Width, request.Height, key)
			result.URL = newURL
			result.Result = inProgress
			result.Cached = false
		} else {
			result.URL = newURL
			result.Result = success
			result.Cached = true
		}
		results = append(results, result)
	}
	return results
}

// Legacy blocking implementation
// But is sped by by parralel processing of images
// Can be sped up by resizing several images in parallel with waitgroups
func (s *ResizingService) ProcessResizes(request models.ResizeRequest) ([]models.ResizeResult, error) {
	results := make([]models.ResizeResult, 0, len(request.URLs))
	var wg sync.WaitGroup

	for _, url := range request.URLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			result := models.ResizeResult{}
			id := s.genID(url)
			key := "/v1/image/" + id + ".jpeg"
			newURL := config.Proto + config.Hostport + key

			//we skipp the processing if image is already in there
			if s.Cache.Contains(key) {
				result.URL = newURL
				result.Result = success
				result.Cached = true
				results = append(results, result)
				return
			}

			resizedImage, err := s.startResizingProcess(url, request.Width, request.Height, key)
			if err != nil {
				log.Printf("failed to resize %s: %v", url, err)
				result.Result = failure
				results = append(results, result)
				return
			}

			s.Cache.Add(key, resizedImage)

			result.URL = newURL
			result.Result = success
			result.Cached = true
			results = append(results, result)
		}(url)
	}
	wg.Wait()
	return results, nil
}

func (s *ResizingService) startResizingProcess(imageUrl string, width uint, height uint, key string) ([]byte, error) {

	//Adding image to inprogress collection
	s.ImagesInProgress.Store(key, inProgress)

	//fetching image
	originalImage, err := s.fetch(imageUrl)
	if err != nil {
		log.Printf("failed to resize %s: %v", imageUrl, err)
		return nil, err
	}

	//resizing image
	resizedImage, err := s.resize(originalImage, width, height)
	if err != nil {
		log.Printf("failed to resize %s: %v", imageUrl, err)
		return nil, err
	}

	//upddating cache
	s.Cache.Add(key, resizedImage)

	//Removing image from in progress collection
	s.ImagesInProgress.Delete(key)

	return resizedImage, nil
}

func (s *ResizingService) fetch(url string) ([]byte, error) {
	r, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %v", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status: %d", r.StatusCode)
	}

	data, err := ioutil.ReadAll(io.LimitReader(r.Body, 15*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read fetch data: %v", err)
	}

	return data, nil
}

func (s *ResizingService) resize(data []byte, width uint, height uint) ([]byte, error) {
	// decode jpeg into image.Image
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to jped decode: %v", err)
	}

	var newImage image.Image

	// if either width or height is 0, it will resize respecting the aspect ratio
	newImage = jpgresize.Resize(width, height, img, jpgresize.Lanczos3)

	newData := bytes.Buffer{}
	err = jpeg.Encode(bufio.NewWriter(&newData), newImage, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to jpeg encode resized image: %v", err)
	}

	return newData.Bytes(), nil
}

func (s *ResizingService) genID(url string) string {
	hash := sha256.Sum256([]byte(url))
	return base64.URLEncoding.EncodeToString(hash[:])
}
