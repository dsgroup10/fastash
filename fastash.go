package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

func SaveFileName(changeID string) string {
	var sum int64
	for _, shard := range strings.Split(changeID, "-") {
		i64, err := strconv.ParseInt(shard, 10, 63)
		if err != nil {
			panic(err)
		}
		sum += i64
	}
	return filepath.Join("stashes", fmt.Sprint(sum/1e8), changeID+".json.gz")
}

var endpoint string

func MakeRequest(changeID string) *http.Request {
	url := "http://" + endpoint + "/api/public-stash-tabs?id=" + changeID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Accept-Encoding", "gzip")
	return req
}

const nextChangeIDPrefix = `{"next_change_id":"`

// ReadGzipNextChangeID returns the next_change_id and consumed bytes from a json.gz reader
func ReadGzipNextChangeID(r io.Reader) (string, []byte) {
	var buf bytes.Buffer
	tee := io.TeeReader(r, &buf)
	gzr, err := gzip.NewReader(tee)
	if err != nil {
		panic(err)
	}
	var gzhead [160]byte
	io.ReadFull(gzr, gzhead[:])
	nextChangeID := string(gzhead[:])
	if !strings.HasPrefix(nextChangeID, nextChangeIDPrefix) {
		log.Panicf("malformed header %s", nextChangeID)
	}
	nextChangeID = nextChangeID[len(nextChangeIDPrefix):]
	parts := strings.SplitN(nextChangeID, `"`, 2)
	if len(parts) != 2 {
		log.Panicf("malformed header %s", nextChangeID)
	}
	return parts[0], buf.Bytes()
}

func ProcessChangeID(pool *ProxyPool, changeID string) (nextChangeID string, consumedBytes []byte, resp *http.Response) {
	resp = pool.Do(MakeRequest(changeID))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		panic(resp.Status)
	}
	nextChangeID, consumedBytes = ReadGzipNextChangeID(resp.Body)
	log.Println("Get", changeID)
	return
}

func OSMkdirCreate(name string) (*os.File, error) {
	f, err := os.Create(name)
	if err == nil {
		return f, nil
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(name), 0755)
		if err == nil {
			f, err = os.Create(name)
		}
	}
	return f, err
}

func Save(changeID string, consumedBytes []byte, resp *http.Response) {
	defer resp.Body.Close()
	finalname := SaveFileName(changeID)
	tmpname := finalname + ".dirty"
	osf, err := OSMkdirCreate(tmpname)
	if err != nil {
		panic(err)
	}
	_, err = osf.Write(consumedBytes)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(osf, resp.Body)
	if err != nil {
		panic(err)
	}
	defer osf.Close()
	osf.Close()
	os.Rename(tmpname, finalname)
}

type unfinishedResponse struct {
	changeID      string
	consumedBytes []byte
	resp          *http.Response
}

func GetDialer(rawurl string) proxy.ContextDialer {
	url, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	dialer, err := proxy.FromURL(url, proxy.Direct)
	if err != nil {
		panic(err)
	}
	return dialer.(proxy.ContextDialer)
}

func main() {
	endpoint = os.Args[1]
	dialers := []proxy.ContextDialer{proxy.Direct}
	for _, rawurl := range os.Args[2:] {
		dialers = append(dialers, GetDialer(rawurl))
	}
	pool := NewProxyPool(
		1200*time.Millisecond, 2, dialers...,
	)
	responses := make(chan unfinishedResponse)

	// IO goroutines
	var wg sync.WaitGroup
	wg.Add(len(pool.Proxies))
	for range pool.Proxies {
		go func() {
			defer wg.Done()
			for resp := range responses {
				Save(resp.changeID, resp.consumedBytes, resp.resp)
			}
		}()
	}

	changeID := "0"
	for {
		r, err := os.Open(SaveFileName(changeID))
		if err == nil {
			nextChangeID, _ := ReadGzipNextChangeID(r)
			log.Println("Cached", changeID)
			changeID = nextChangeID
			r.Close()
			continue
		}
		nextChangeID, consumedBytes, resp := ProcessChangeID(pool, changeID)
		if nextChangeID == changeID {
			resp.Body.Close()
			break
		}
		responses <- unfinishedResponse{changeID, consumedBytes, resp}
		changeID = nextChangeID
	}

	// wait for everyone to finish writing
	wg.Wait()
}
