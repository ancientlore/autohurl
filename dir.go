package main

import (
	"golang.org/x/net/context"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func readDir(ctx context.Context, folder, filePattern string, sleep time.Duration, maxSize int) <-chan os.FileInfo {
	done := ctx.Done()
	out := make(chan os.FileInfo)
	looper := func() {
		defer close(out)
		for {
			var wait time.Duration = 0
			fil, err := os.Open(folder)
			if err != nil {
				log.Print("Unable to open folder: ", folder, " ", err)
				return
			}
			info, err := fil.Readdir(0)
			fil.Close()
			if err == io.EOF {
				if info == nil || len(info) == 0 {
					wait = sleep
				}
			} else if err != nil {
				log.Print("Error reading folder: ", folder, " ", err)
				wait = 10 * time.Second
			}

			for _, inf := range info {
				if !inf.IsDir() {
					if matched, _ := filepath.Match(filePattern, inf.Name()); matched && inf.Size() < int64(maxSize) {
						select {
						case out <- inf:
						case <-done:
							return
						}
					}
				}

			}

			if wait > 0 {
				c := time.After(wait)
				select {
				case <-done:
					return
				case <-c:
				}
			}
		}
	}

	go looper()
	return out
}
