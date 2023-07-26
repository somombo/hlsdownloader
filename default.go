// main
package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/grafov/m3u8"
	"io"
	"log"
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

//func writePlaylist(u *url.URL, mpl m3u8.Playlist) {
//	fileName := path.Base(u.Path)
//	out, err := os.Create(OUT_PATH + fileName)
//	if err != nil {
//		log.Fatal("cms3> " + err.Error())
//	}
//	defer out.Close()
//
//	_, err = mpl.Encode().WriteTo(out)
//	if err != nil {
//		log.Fatal("cms4> " + err.Error())
//	}
//}

func writePlaylistFile(u *url.URL, r io.Reader) {
	fileName := OUT_PATH + u.Path
	dir := path.Dir(fileName)
	_, err := os.Stat(dir)
	if err != nil {
		os.MkdirAll(dir, 0755)
	}

	out, err := os.Create(fileName)
	if err != nil {
		log.Fatal("cms3> " + err.Error())
	}
	defer out.Close()

	_, err = io.Copy(out, r)
	if err != nil {
		log.Fatal("cms4> " + err.Error())
	}
	log.Print("cms8 m3u8file:> ", fileName, "\n")

}

func download(u *url.URL) {
	fileName := OUT_PATH + u.Path
	dir := path.Dir(fileName)
	_, err := os.Stat(dir)
	if err != nil {
		os.MkdirAll(dir, 0755)
	}

	out, err := os.Create(fileName)
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

func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return io.NopCloser(&buf), io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func getPlaylist(u *url.URL) {
	content, err := getContent(u)
	defer func() {
		content.Close()
	}()
	if err != nil {
		log.Fatal("cms9> " + err.Error())
	}
	r1, r2, err := drainBody(content)
	if err != nil {
		log.Fatal("cms9> " + err.Error())
	}

	playlist, listType, err := m3u8.DecodeFrom(r1, true)
	if err != nil {
		log.Fatal("cms10> " + err.Error())
	}
	if listType != m3u8.MEDIA && listType != m3u8.MASTER {
		log.Fatal("cms11> " + "Not a valid playlist")
		return
	}

	writePlaylistFile(u, r2)
	log.Print("cms14> "+"Downloaded Playlist: ", path.Base(u.Path), "\n")

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

		//writePlaylist(u, m3u8.Playlist(mediapl))
		//log.Print("cms16> "+"Downloaded Media Playlist: ", path.Base(u.Path))

		//time.Sleep(time.Duration(int64(mediapl.TargetDuration)) * time.Second)

	}

	//time.Sleep(time.Duration(11) * time.Second)

}

// var OUT_PATH string = "/inetpub/wwwroot/cast/media/znbc/"
var OUT_PATH string = "/data/media/"
var IN_URL string = "http://rtmp.ottdemo.rrsat.com/rrsatlive4/rrsat4multi.smil/playlist.m3u8"

// var IN_URL string = "http://makombo.org/cast/media/cmshlstest/master.m3u8"
// var IN_URL string = "http://makombo.org/cast/media/DevBytes%20Google%20Cast%20SDK_withGDLintro_Apple_HLS_h264_SF_16x9_720p/DevBytes%20Google%20Cast%20SDK_withGDLintro_Apple_HLS_h264_SF_16x9_720p.m3u8"
// var IN_URL string = "http://makombo.org/cast/media/DevBytes%20Google%20Cast%20SDK_withGDLintro_Apple_HLS_h264_SF_16x9_720p/stream-2-229952/index.m3u8"
func main() {
	//hi
	flag.Parse()

	os.Stderr.Write([]byte(fmt.Sprintf("HTTP Live Streaming (HLS) downloader\n")))
	os.Stderr.Write([]byte("Copyright (C) 2014 Chisomo Sakala. Licensed for use under the GNU GPL version 3.\n"))

	switch flag.NArg() {
	case 2:
		IN_URL = flag.Arg(0)
		OUT_PATH = flag.Arg(1)
	case 1:
		IN_URL = flag.Arg(0)
	case 0:
		os.Stderr.Write([]byte("Usage: hlsdownloader absolute-url-m3u8-file path-to-output-directory\n"))
		os.Stderr.Write([]byte(fmt.Sprintf("\n...Continuing Under the assumption: \n\thlsdownloader %s %s\n", IN_URL, OUT_PATH)))
		flag.PrintDefaults()
	default:
		os.Stderr.Write([]byte(fmt.Sprintf("Usage: hlsdownloader absolute-url-m3u8-file path-to-output-directory\n")))
		os.Exit(2)
	}

	if !strings.HasPrefix(IN_URL, "http") {
		log.Fatal("cms17> " + "Playlist URL must begin with http/https")
	}

	fmt.Print("\n\n\n")

	theURL, err := url.Parse(IN_URL)
	if err != nil {
		log.Fatal("cms18> " + err.Error())
	}

	getPlaylist(theURL)
}
