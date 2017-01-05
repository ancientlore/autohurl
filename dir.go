package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type fileInfoSlice []os.FileInfo

func (fi fileInfoSlice) Len() int           { return len(fi) }
func (fi fileInfoSlice) Swap(i, j int)      { fi[i], fi[j] = fi[j], fi[i] }
func (fi fileInfoSlice) Less(i, j int) bool { return fi[i].Name() < fi[j].Name() }

func readDir(ctx context.Context, name string, cfg *FolderCfg) <-chan os.FileInfo {
	done := ctx.Done()
	out := make(chan os.FileInfo)
	looper := func() {
		defer close(out)
		var lastInfo = make([]os.FileInfo, 0)
		for {
			var wait time.Duration
			fil, err := os.Open(cfg.Folder)
			if err != nil {
				log.Print(name, ": Unable to open folder: ", cfg.Folder, " ", err)
				return
			}
			info, err := fil.Readdir(cfg.BatchSize)
			fil.Close()
			if err == io.EOF {
				if info == nil || len(info) == 0 {
					wait = time.Duration(cfg.SleepTime)
				}
			} else if err != nil {
				log.Print(name, ": Error reading folder: ", cfg.Folder, " ", err)
				wait = 10 * time.Second
			} else {
				wait = time.Duration(cfg.SleepTime)
				for _, inf := range info {
					// Don't send directories
					if !inf.IsDir() {
						// match file pattern
						if matched, _ := filepath.Match(cfg.FilesPat, inf.Name()); matched {
							// check if we saw the file last time
							loc := sort.Search(len(lastInfo), func(i int) bool {
								return lastInfo[i].Name() >= inf.Name()
							})
							// new file not in list
							if loc >= len(lastInfo) || (loc < len(lastInfo) && lastInfo[loc].Name() != inf.Name()) {
								// send along the file
								select {
								case out <- inf:
									wait = 0
								case <-done:
									return
								}
							} else if inf.ModTime().Before(time.Now().Add(-time.Minute)) {
								// send along the file - handle very old files that for some reason are still there
								select {
								case out <- inf:
								case <-done:
									return
								}
							}
						}
					}
				}
				lastInfo = info
				sort.Sort(fileInfoSlice(lastInfo))
			}
			if wait > 0 {
				// log.Print(name, ": Waiting ", wait.String())
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
