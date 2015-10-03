package main

import (
	//"github.com/ancientlore/kubismus"
	//"github.com/nu7hatch/gouuid"
	"golang.org/x/net/context"
	//"io"
	//"io/ioutil"
	"log"
	//"mime"
	"net/http"
	"os"
	//"path/filepath"
	//"strconv"
	//"strings"
	"sync"
	"time"
)

func doHttp(ctx context.Context, name string, cfg *FolderCfg, ch <-chan os.FileInfo) {
	var wg sync.WaitGroup

	// create HTTP transport and client
	cfg.transport = &http.Transport{DisableKeepAlives: cfg.NoKeepAlive, MaxIdleConnsPerHost: cfg.Conns, DisableCompression: cfg.NoCompress, ResponseHeaderTimeout: time.Duration(cfg.Timeout)}
	cfg.client = &http.Client{Transport: cfg.transport, Timeout: time.Duration(cfg.Timeout)}

	// create HTTP posting threads
	wg.Add(cfg.Conns)
	for i := 0; i < cfg.Conns; i++ {
		go posterThread(ctx, name, cfg, ch, &wg)
	}

	// Wait for threads to finish
	wg.Wait()
}

func posterThread(ctx context.Context, name string, cfg *FolderCfg, ch <-chan os.FileInfo, wg *sync.WaitGroup) {
	done := ctx.Done()
	defer wg.Done()

	for {
		select {
		case inf, ok := <-ch:
			if !ok {
				return
			}
			log.Print(name, ": ", inf.Name())

			/*
				var f io.ReadCloser
				var err error

				// skip large file, possibly moving it
				if cfg.MaxFileSize > 0 && inf.Size() > cfg.MaxFileSize {
					log.Print(name, ":  ", st.Size(), " byte file, skip/rename: ", inf.Name())
					if cfg.MoveFailedTo != "" {
						_, fname := filepath.Split(inf.Name())
						newFname := filepath.Join(cfg.MoveFailedTo, fname)
						//log.Printf("%s to %s\n", f, newFname)
						err := os.Rename(inf.Name(), newFname)
						if err != nil {
							log.Print(name, ": failed to move file ", inf.Name(), " to ", newFname, ": ", err)
						}
					}
					continue
				}

				if i.Filename != "" {
					f, err = os.Open(i.Filename)
					if err != nil {
						log.Printf(name, ": Unable to open %s", i.Filename)
						continue
					}
				}
				req, err := http.NewRequest(i.Method, i.URL, f)
				if err != nil {
					if f != nil {
						f.Close()
					}
					log.Fatal(err)
				}
				if i.Filename != "" {
					ct := mime.TypeByExtension(filepath.Ext(i.Filename))
					if ct != "" {
						req.Header.Set("Content-Type", ct)
					}
					req.ContentLength = i.Size
				}
				if useRequestId {
					guid, err := uuid.NewV4()
					if err == nil {
						req.Header.Set("X-RequestID", guid.String())
					}
				}
				for _, h := range headers {
					if h.Mode == HdrSet {
						req.Header.Set(h.Key, h.Value)
					} else {
						req.Header.Add(h.Key, h.Value)
					}
				}
				req.Close = false
				// log.Printf("%#v", req)
				t := time.Now()
				resp, err := client.Do(req)
				if f != nil {
					f.Close()
				}
				if err != nil {
					kubismus.Metric("Error", 1, 0)
					log.Print("HTTP error ", i.URL, ": ", err)
					continue
				}
				kubismus.Metric("Sent", 1, float64(i.Size))
				name := urlToFilename(&i)
				// log.Print("File would be ", name)
				var outfile io.WriteCloser
				writeTo := ioutil.Discard
				if !discard {
					outfile, err := os.Create(name)
					if err != nil {
						log.Print("Unable to create file ", name)
					} else {
						writeTo = outfile
					}
				}
				if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
					log.Print("Failed to post to ", i.URL, ", status ", resp.Status)
				}
				if resp.ContentLength > 0 {
					statusRange := resp.StatusCode / 100
					switch statusRange {
					case 1:
						kubismus.Metric("Received100", 1, 0)
					case 2:
						kubismus.Metric("Received200", 1, 0)
					case 3:
						kubismus.Metric("Received300", 1, 0)
					case 4:
						kubismus.Metric("Received400", 1, 0)
					case 5:
						kubismus.Metric("Received500", 1, 0)
					}
					sz, err := io.Copy(writeTo, resp.Body)
					if err == nil {
						kubismus.Metric("Received", 1, float64(sz))
					}
				}
				resp.Body.Close()
				d := time.Since(t)
				kubismus.Metric("ResponseTime", 1, float64(d.Nanoseconds())/float64(time.Second))
				if outfile != nil {
					outfile.Close()
				}
			*/
		case <-done:
			return
		}
	}
}
