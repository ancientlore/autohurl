package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ancientlore/kubismus"
	"github.com/nu7hatch/gouuid"
)

func doHTTP(ctx context.Context, name string, cfg *FolderCfg, ch <-chan os.FileInfo) {
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
			//log.Print(name, ": ", inf.Name())
			processFile(name, cfg, inf)
		case <-done:
			return
		}
	}
}

type postError struct {
	err error
}

func (e postError) Error() string {
	return e.err.Error()
}

func processFile(name string, cfg *FolderCfg, inf os.FileInfo) {
	// unlikely
	if inf.Name() == "" {
		log.Printf("%s: File not named", name)
		return
	}

	// Give very new files a chance to finish up. Sort of hacky.
	if inf.ModTime().After(time.Now().Add(-time.Second)) {
		time.Sleep(time.Second)
	}

	fname := filepath.Join(cfg.Folder, inf.Name())

	// skip large file, possibly moving it
	if cfg.MaxFileSize > 0 && inf.Size() > cfg.MaxFileSize {
		//log.Print(name, ":  ", inf.Size(), " byte file, skip/rename: ", fname)
		if cfg.MoveFailedTo != "" {
			_, fn := filepath.Split(inf.Name())
			newFname := filepath.Join(cfg.MoveFailedTo, fn)
			//log.Printf("%s to %s\n", f, newFname)
			err := os.Rename(fname, newFname)
			if err != nil {
				log.Print(name, ": failed to move oversized file ", fname, " to ", newFname, ": ", err)
			}
		}
		return
	}

	err := postFile(name, cfg, inf)
	if err == nil {
		if cfg.MoveTo == "" {
			err = os.Remove(fname)
			if err != nil {
				log.Print(name, ": failed to remove file ", fname, ": ", err)
			}
		} else {
			_, fn := filepath.Split(inf.Name())
			newFname := filepath.Join(cfg.MoveTo, fn)
			//log.Printf("%s to %s\n", f, newFname)
			err := os.Rename(fname, newFname)
			if err != nil {
				log.Print(name, ": failed to move file ", fname, " to ", newFname, ": ", err)
			}
		}
	} else {
		switch err.(type) {
		case postError:
			if cfg.MoveFailedTo != "" {
				_, fn := filepath.Split(inf.Name())
				newFname := filepath.Join(cfg.MoveFailedTo, fn)
				//log.Printf("%s to %s\n", f, newFname)
				err := os.Rename(fname, newFname)
				if err != nil {
					log.Print(name, ": failed to move failed file ", fname, " to ", newFname, ": ", err)
				}
			}
		}
	}
}

func postFile(name string, cfg *FolderCfg, inf os.FileInfo) error {
	var f io.ReadCloser
	var err error

	fname := filepath.Join(cfg.Folder, inf.Name())

	// open file
	f, err = os.Open(fname)
	if err != nil {
		log.Printf("%s: Unable to open %s", name, fname)
		return err
	}
	// defer f.Close()

	// create request
	req, err := http.NewRequest(cfg.Method, cfg.URL, f)
	if err != nil {
		f.Close()
		log.Fatal(err)
	}

	// set content type if possible
	ct := mime.TypeByExtension(filepath.Ext(fname))
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	// set content length
	req.ContentLength = inf.Size()

	// Set request ID header if desired
	if cfg.UseRequestID != "" {
		guid, err := uuid.NewV4()
		if err == nil {
			req.Header.Set(cfg.UseRequestID, guid.String())
		}
	}

	// set file info headers
	if cfg.FileInfo {
		req.Header.Set("X-Autohurl-Name", inf.Name())
		req.Header.Set("X-Autohurl-Size", fmt.Sprintf("%d", inf.Size()))
		req.Header.Set("X-Autohurl-Modtime", inf.ModTime().Format(time.RFC3339Nano))
	}

	// set headers
	for _, h := range cfg.Headers {
		if h.Mode == HdrSet {
			req.Header.Set(h.Key, h.Value)
		} else {
			req.Header.Add(h.Key, h.Value)
		}
	}
	req.Close = false

	// log.Printf("%#v", req)
	t := time.Now()
	resp, err := cfg.client.Do(req)
	f.Close()
	if err != nil {
		kubismus.Metric(name+"_Errors", 1, 0)
		log.Print(name, ": HTTP error ", cfg.URL, ": ", err)
		return postError{err: err}
	}
	kubismus.Metric(name+"_Sent", 1, float64(inf.Size()))

	if resp.ContentLength > 0 {
		sz, err := io.Copy(ioutil.Discard, resp.Body)
		if err == nil {
			kubismus.Metric(name+"_Received", 1, float64(sz))
		} else {
			// ignoring it because HTTP code was a success
		}
	}
	resp.Body.Close()
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		kubismus.Metric(name+"_Errors", 1, 0)
		log.Print(name, ": Failed to post to ", cfg.URL, ", status ", resp.Status)
		return postError{err: errors.New(fmt.Sprint(name, ": Failed to post to ", cfg.URL, ", status ", resp.Status))}
	}
	d := time.Since(t)
	kubismus.Metric(name+"_ResponseTime", 1, float64(d.Nanoseconds())/float64(time.Second))

	return nil
}
