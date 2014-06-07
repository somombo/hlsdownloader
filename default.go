// main
package main

import (
	"fmt"
	"github.com/golang/groupcache/lru"
	"github.com/grafov/m3u8"
	"io"
	"log"
	//"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

const (
	OUT_PATH = "mystream/"
	IN_URL   = "http://rtmp.ottdemo.rrsat.com/rrsatlive4/rrsat4multi.smil/playlist.m3u8"
	//IN_URL   = "http://makombo.org/cast/media/cmshlstest/master.m3u8"
	//IN_URL   = "http://makombo.org/cast/media/DevBytes%20Google%20Cast%20SDK_withGDLintro_Apple_HLS_h264_SF_16x9_720p/DevBytes%20Google%20Cast%20SDK_withGDLintro_Apple_HLS_h264_SF_16x9_720p.m3u8"
	//IN_URL = "http://makombo.org/cast/media/DevBytes%20Google%20Cast%20SDK_withGDLintro_Apple_HLS_h264_SF_16x9_720p/stream-2-229952/index.m3u8"
)

var client = &http.Client{}

func getContent(u *url.URL) (io.ReadCloser, error) {
	var USER_AGENT string

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", USER_AGENT)
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err)
		time.Sleep(time.Duration(2) * time.Second)
	}

	if resp.StatusCode != 200 {
		log.Printf("Received HTTP %v for %v\n", resp.StatusCode, u.String())
	}

	return resp.Body, err

}

func absolutize(rawurl string, u *url.URL) (uri *url.URL, err error) {

	suburl := rawurl
	uri, err = u.Parse(suburl)
	if err != nil {
		return
	}

	if rawurl == u.String() {
		return
	}

	if !uri.IsAbs() { // relative URI
		if rawurl[0] == '/' { // from the root
			suburl = fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, rawurl)
		} else { // last element
			splitted := strings.Split(u.String(), "/")
			splitted[len(splitted)-1] = rawurl

			suburl = strings.Join(splitted, "/")
		}
	}

	suburl, err = url.QueryUnescape(suburl)
	if err != nil {
		return
	}

	uri, err = u.Parse(suburl)
	if err != nil {
		return
	}

	return
}

func writePlaylist(u *url.URL, mpl m3u8.Playlist) {
	fileName := path.Base(u.Path)
	out, err := os.Create(OUT_PATH + fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	_, err = mpl.Encode().WriteTo(out)
	if err != nil {
		log.Fatal(err)
	}
}

func download(u *url.URL) {
	fileName := path.Base(u.Path)

	out, err := os.Create(OUT_PATH + fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	content, err := getContent(u)
	if err != nil {
		log.Print(err)
		//continue
	}
	defer content.Close()

	_, err = io.Copy(out, content)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Downloaded ", fileName, "\n")

}

func getPlaylist(u *url.URL) {

	cache := lru.New(64)

	content, err := getContent(u)
	if err != nil {
		log.Fatal(err)
	}

	playlist, listType, err := m3u8.DecodeFrom(content, true)
	if err != nil {
		log.Fatal(err)
	}
	content.Close()

	if listType != m3u8.MEDIA && listType != m3u8.MASTER {
		log.Fatal("Not a valid playlist")
		return
	}

	if listType == m3u8.MASTER {

		masterpl := playlist.(*m3u8.MasterPlaylist)
		for k, variant := range masterpl.Variants {

			if variant != nil {

				msURL, err := absolutize(variant.URI, u)
				if err != nil {
					log.Fatal(err)
				}
				getPlaylist(msURL)

				log.Print("Downloaded index number ", k)

			}

		}
		writePlaylist(u, m3u8.Playlist(masterpl))
		log.Print("Downloaded Master Playlist: ", path.Base(u.Path), "\n")

		return

	}

	if listType == m3u8.MEDIA {

		mediapl := playlist.(*m3u8.MediaPlaylist)
		for _, segment := range mediapl.Segments {
			if segment != nil {

				msURL, err := absolutize(segment.URI, u)
				if err != nil {
					log.Fatal(err)
				}

				_, hit := cache.Get(msURL.String())
				if !hit {
					cache.Add(msURL.String(), nil)
					download(msURL)
				}

			}
		}

		writePlaylist(u, m3u8.Playlist(mediapl))
		log.Print("Downloaded Media Playlist: ", path.Base(u.Path), "\n")

		//time.Sleep(time.Duration(int64(mediapl.TargetDuration)) * time.Second)

	}

	//time.Sleep(time.Duration(11) * time.Second)

}

func main() {
	log.Print("Welcome to HLS Dowloader By Chisomo Sakala!-----------\n")

	theURL, err := url.Parse(IN_URL)
	if err != nil {
		log.Fatal(err)
	}
	for {

		getPlaylist(theURL)
		log.Print("Refeshed Main Play")
	}

}
