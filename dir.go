package main

import (
	"golang.org/x/net/context"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type FI []os.FileInfo

func (fi FI) Len() int           { return len(fi) }
func (fi FI) Swap(i, j int)      { fi[i], fi[j] = fi[j], fi[i] }
func (fi FI) Less(i, j int) bool { return fi[i].Name() < fi[j].Name() }

func readDir(ctx context.Context, folder, filePattern string, sleep time.Duration, maxSize, dirBatchSize int) <-chan os.FileInfo {
	done := ctx.Done()
	out := make(chan os.FileInfo)
	looper := func() {
		defer close(out)
		var lastInfo = make([]os.FileInfo, 0)
		for {
			var wait time.Duration = 0
			fil, err := os.Open(folder)
			if err != nil {
				log.Print("Unable to open folder: ", folder, " ", err)
				return
			}
			info, err := fil.Readdir(dirBatchSize)
			fil.Close()
			if err == io.EOF {
				if info == nil || len(info) == 0 {
					wait = sleep
				}
			} else if err != nil {
				log.Print("Error reading folder: ", folder, " ", err)
				wait = 10 * time.Second
			}

			wait = sleep
			for _, inf := range info {
				if !inf.IsDir() {
					// match file pattern
					if matched, _ := filepath.Match(filePattern, inf.Name()); matched && inf.Size() < int64(maxSize) {
						// check if we saw the file last time
						loc := sort.Search(len(lastInfo), func(i int) bool {
							return lastInfo[i].Name() >= inf.Name()
						})
						if loc >= len(lastInfo) || (loc < len(lastInfo) && lastInfo[loc].Name() != inf.Name()) {
							// send along the file
							select {
							case out <- inf:
								wait = 0
							case <-done:
								return
							}
						}
					}
				}

			}
			lastInfo = info
			sort.Sort(FI(lastInfo))
			if wait > 0 {
				log.Print("Waiting ", wait.String())
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
