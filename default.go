// main
package main

import (
	"flag"
	"fmt"
	//"github.com/golang/groupcache/lru"
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

var client = &http.Client{}

func getContent(u *url.URL) (io.ReadCloser, error) {
	var USER_AGENT string

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		log.Fatal("cms1> " + err.Error())
	}

	req.Header.Set("User-Agent", USER_AGENT)
	resp, err := client.Do(req)
	if err != nil {
		log.Print("cms2> " + err.Error())
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
		log.Fatal("cms3> " + err.Error())
	}
	defer out.Close()

	_, err = mpl.Encode().WriteTo(out)
	if err != nil {
		log.Fatal("cms4> " + err.Error())
	}
}

func download(u *url.URL) {
	fileName := path.Base(u.Path)

	out, err := os.Create(OUT_PATH + fileName)
	if err != nil {
		log.Fatal("cms5> " + err.Error())
	}
	defer out.Close()

	content, err := getContent(u)
	if err != nil {
		log.Print("cms6> " + err.Error())
		//continue
	}
	defer content.Close()

	_, err = io.Copy(out, content)
	if err != nil {
		log.Print("cms7> " + err.Error() + "Failed to download " + fileName + "\n")
	}

	log.Print("cms8> "+"Downloaded ", fileName, "\n")

}

func getPlaylist(u *url.URL) {

	//cache := lru.New(64)

	content, err := getContent(u)
	if err != nil {
		log.Fatal("cms9> " + err.Error())
	}

	playlist, listType, err := m3u8.DecodeFrom(content, true)
	if err != nil {
		log.Fatal("cms10> " + err.Error())
	}
	content.Close()

	if listType != m3u8.MEDIA && listType != m3u8.MASTER {
		log.Fatal("cms11> " + "Not a valid playlist")
		return
	}

	if listType == m3u8.MASTER {

		masterpl := playlist.(*m3u8.MasterPlaylist)
		for k, variant := range masterpl.Variants {

			if variant != nil {

				msURL, err := absolutize(variant.URI, u)
				if err != nil {
					log.Fatal("cms12> " + err.Error())
				}
				getPlaylist(msURL)

				log.Print("cms13> "+"Downloaded chunklist number ", k+1, "\n\n")
				//break
			}

		}
		writePlaylist(u, m3u8.Playlist(masterpl))
		log.Print("cms14> "+"Downloaded Master Playlist: ", path.Base(u.Path), "\n")

		return

	}

	if listType == m3u8.MEDIA {

		mediapl := playlist.(*m3u8.MediaPlaylist)
		for _, segment := range mediapl.Segments {
			if segment != nil {

				msURL, err := absolutize(segment.URI, u)
				if err != nil {
					log.Fatal("cms15> " + err.Error())
				}

				//_, hit := cache.Get(msURL.String())
				//if !hit {
				//	cache.Add(msURL.String(), nil)
				download(msURL)
				//}

			}
		}

		writePlaylist(u, m3u8.Playlist(mediapl))
		log.Print("cms16> "+"Downloaded Media Playlist: ", path.Base(u.Path))

		//time.Sleep(time.Duration(int64(mediapl.TargetDuration)) * time.Second)

	}

	//time.Sleep(time.Duration(11) * time.Second)

}

var OUT_PATH string = "/inetpub/wwwroot/cast/media/znbc/"
var IN_URL string = "http://rtmp.ottdemo.rrsat.com/rrsatlive4/rrsat4multi.smil/playlist.m3u8"

//var IN_URL string = "http://makombo.org/cast/media/cmshlstest/master.m3u8"
//var IN_URL string = "http://makombo.org/cast/media/DevBytes%20Google%20Cast%20SDK_withGDLintro_Apple_HLS_h264_SF_16x9_720p/DevBytes%20Google%20Cast%20SDK_withGDLintro_Apple_HLS_h264_SF_16x9_720p.m3u8"
//var IN_URL string = "http://makombo.org/cast/media/DevBytes%20Google%20Cast%20SDK_withGDLintro_Apple_HLS_h264_SF_16x9_720p/stream-2-229952/index.m3u8"
func main() {
	//hi
	flag.Parse()

	os.Stderr.Write([]byte(fmt.Sprintf("HTTP Live Streaming (HLS) downloader\n")))
	os.Stderr.Write([]byte("Copyright (C) 2014 Chisomo Sakala. Licensed for use under the GNU GPL version 3.\n"))

	if flag.NArg() < 2 {
		os.Stderr.Write([]byte("Usage: hlsdownloader absolute-url-m3u8-file path-to-output-directory\n"))
		os.Stderr.Write([]byte(fmt.Sprintf("\n...Continuing Under the assumption: \n\thlsdownloader %s %s\n", IN_URL, OUT_PATH)))

		flag.PrintDefaults()
		//os.Exit(2)
	} else {
		IN_URL = flag.Arg(0)
		OUT_PATH = flag.Arg(1)
	}

	if !strings.HasPrefix(IN_URL, "http") {
		log.Fatal("cms17> " + "Playlist URL must begin with http/https")
	}

	fmt.Print("\n\n\n")

	theURL, err := url.Parse(IN_URL)
	if err != nil {
		log.Fatal("cms18> " + err.Error())
	}
	for {

		getPlaylist(theURL)
		log.Print("cms19> " + "Refeshed Master Playlist\n\n")
	}

}
