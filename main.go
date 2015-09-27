package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/ancientlore/flagcfg"
	"github.com/ancientlore/kubismus"
	"github.com/facebookgo/flagenv"
	"golang.org/x/net/context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
)

// github.com/ancientlore/binder is used to package the web files into the executable.
//go:generate binder -package main -o webcontent.go media/*.png

type hdrMode int

const (
	HdrSet hdrMode = iota
	HdrAdd
)

type hdr struct {
	Key   string
	Value string
	Mode  hdrMode
}

var (
	client       *http.Client
	transport    *http.Transport
	addr         string        = ":8080"
	conns        int           = 10
	timeout      time.Duration = 10 * time.Second
	method       string        = "GET"
	filesPat     string        = "*.*"
	batchSize    int           = 32 * 1024
	cpuProfile   string
	memProfile   string
	cpus         int
	workingDir   string
	noCompress   bool
	noKeepAlive  bool
	useRequestId bool
	headerDelim  string = "|"
	headerText   string
	version      bool
	help         bool
	headers      []hdr
	sleepTime    time.Duration = time.Second
	maxFileSize  int           = 1024 * 1024
)

func init() {

	// http service/status address
	flag.StringVar(&addr, "addr", addr, "HTTP service address for monitoring.")

	// http post settings
	flag.IntVar(&conns, "conns", conns, "Number of concurrent HTTP connections.")
	flag.DurationVar(&timeout, "timeout", timeout, "HTTP timeout.")
	flag.StringVar(&filesPat, "files", filesPat, "Pattern of files to post, like *.xml.")
	flag.StringVar(&method, "method", method, "HTTP method.")
	flag.BoolVar(&useRequestId, "requestid", useRequestId, "Send X-RequestID header.")
	flag.BoolVar(&noCompress, "nocompress", noCompress, "Disable HTTP compression.")
	flag.BoolVar(&noKeepAlive, "nokeepalive", noKeepAlive, "Disable HTTP keep-alives.")

	// processing
	flag.DurationVar(&sleepTime, "sleep", sleepTime, "Interval to wait when no files are found.")
	flag.IntVar(&maxFileSize, "maxsize", maxFileSize, "Maximum file size to post.")
	flag.IntVar(&batchSize, "batchsize", batchSize, "Readdir batch size.")

	// headers
	flag.StringVar(&headerDelim, "hdrdelim", headerDelim, "Delimiter for HTTP headers specified with -header.")
	flag.StringVar(&headerText, "headers", headerText, "HTTP headers, delimited by -hdrdelim.")

	// profiling
	flag.StringVar(&cpuProfile, "cpuprofile", cpuProfile, "Write CPU profile to given file.")
	flag.StringVar(&memProfile, "memprofile", memProfile, "Write memory profile to given file.")

	// runtime
	flag.IntVar(&cpus, "cpu", cpus, "Number of CPUs to use.")
	flag.StringVar(&workingDir, "wd", workingDir, "Set the working directory.")

	// help
	flag.BoolVar(&version, "version", false, "Show version.")
	flag.BoolVar(&help, "help", false, "Show help.")
}

func showHelp() {
	fmt.Println(`
    __    __  ______  __ 
   / /_  / / / / __ \/ / 
  / __ \/ / / / /_/ / /  
 / / / / /_/ / _, _/ /___
/_/ /_/\____/_/ |_/_____/

A tool to continuously post files found in a folder.

Usage:
  hurl [options] url1 [url2 ... urlN]

Example:
  hurl -method POST -files "*.xml" -conns 10 http://localhost/svc/foo http://localhost/svc/bar

Options:`)
	flag.PrintDefaults()
	fmt.Println(`
All of the options can be set via environment variables prefixed with "AUTOHURL_" - for instance,
AUTOHURL_TIMEOUT can be set to "30s" to increase the default timeout.

Options can also be specified in a TOML configuration file named "autohurl.config". The location
of the file can be overridden with the AUTOHURL_CONFIG environment variable.`)
}

func showVersion() {
	fmt.Printf("autohURL version %s\n", AUTOHURL_VERSION)
}

func parseHeaders() error {
	headers = make([]hdr, 0)
	headerText = strings.TrimSpace(headerText)
	if headerText != "" {
		arr := strings.Split(headerText, headerDelim)
		found := make(map[string]bool)
		for _, h := range arr {
			harr := strings.SplitN(h, ":", 2)
			if len(harr) != 2 {
				return errors.New("Unable to parse header: " + h)
			}
			newHdr := hdr{Key: strings.TrimSpace(harr[0]), Value: strings.TrimSpace(harr[1])}
			_, ok := found[newHdr.Key]
			if !ok {
				found[newHdr.Key] = true
				newHdr.Mode = HdrSet
			} else {
				newHdr.Mode = HdrAdd
			}
			headers = append(headers, newHdr)
		}
	}
	return nil
}

func main() {
	// Parse flags from command-line
	flag.Parse()

	// Parser flags from config
	flagcfg.AddDefaults()
	flagcfg.Parse()

	// Parse flags from environment (using github.com/facebookgo/flagenv)
	flagenv.Prefix = "AUTOHURL_"
	flagenv.Parse()

	if help {
		showHelp()
		return
	}

	if version {
		showVersion()
		return
	}

	err := parseHeaders()
	if err != nil {
		log.Fatal(err)
	}
	// log.Printf("%#v", headers)

	// setup number of CPUs
	runtime.GOMAXPROCS(cpus)

	// setup cpu profiling if desired
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer func() {
			log.Print("Writing CPU profile to ", cpuProfile)
			pprof.StopCPUProfile()
			f.Close()
		}()
	}

	// create HTTP transport and client
	transport = &http.Transport{DisableKeepAlives: noKeepAlive, MaxIdleConnsPerHost: conns, DisableCompression: noCompress, ResponseHeaderTimeout: timeout}
	client = &http.Client{Transport: transport, Timeout: timeout}

	// setup Kubismus
	kubismus.Setup("autohURL", "/media/logo36.png")
	kubismus.Define("Sent", kubismus.COUNT, "HTTP Posts")
	kubismus.Define("Sent", kubismus.SUM, "Bytes Sent")
	kubismus.Define("Received", kubismus.SUM, "Bytes Received")
	kubismus.Define("Received100", kubismus.COUNT, "1xx Responses")
	kubismus.Define("Received200", kubismus.COUNT, "2xx Responses")
	kubismus.Define("Received300", kubismus.COUNT, "3xx Responses")
	kubismus.Define("Received400", kubismus.COUNT, "4xx Responses")
	kubismus.Define("Received500", kubismus.COUNT, "5xx Responses")
	kubismus.Define("Error", kubismus.COUNT, "Communication Errors")
	kubismus.Define("ResponseTime", kubismus.AVERAGE, "Average Time (s)")
	kubismus.Note("Concurrent Connections", strconv.Itoa(conns))
	kubismus.Note("HTTP Method", method)
	kubismus.Note("HTTP Timeout", timeout.String())
	kubismus.Note("Processors", fmt.Sprintf("%d of %d", runtime.GOMAXPROCS(0), runtime.NumCPU()))
	kubismus.Note("Data files", strings.Join(strings.Split(filesPat, ","), "\n"))
	kubismus.Note("URLs", strings.Join(flag.Args(), "\n"))
	kubismus.Note("Keep-alive", strconv.FormatBool(!noKeepAlive))
	kubismus.Note("Compress", strconv.FormatBool(!noCompress))
	kubismus.Note("Use Request ID", strconv.FormatBool(useRequestId))
	kubismus.Note("Sleep interval", sleepTime.String())
	kubismus.Note("Max file size", strconv.FormatInt(int64(maxFileSize), 10))
	http.Handle("/", http.HandlerFunc(kubismus.ServeHTTP))
	http.HandleFunc("/media/", ServeHTTP)

	// switch to working dir
	if workingDir != "" {
		err := os.Chdir(workingDir)
		if err != nil {
			log.Fatal(err)
		}
	}
	wd, err := os.Getwd()
	if err == nil {
		kubismus.Note("Working Directory", wd)
	}

	// setup the thread context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// spawn a function that updates the number of goroutines shown in the status page
	go func() {
		done := ctx.Done()
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				kubismus.Note("Goroutines", fmt.Sprintf("%d", runtime.NumGoroutine()))
			}
		}
	}()

	// spawn the status web site
	go func() {
		log.Fatal(http.ListenAndServe(addr, nil))
	}()

	// handle kill signals
	go func() {
		// Set up channel on which to send signal notifications.
		// We must use a buffered channel or risk missing the signal
		// if we're not ready to receive when the signal is sent.
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)

		// Block until a signal is received.
		s := <-c
		log.Print("Got signal ", s, ", canceling work")
		cancel()
	}()

	// Build pipeline
	/*
		var patList []string
		if filesPat != "" {
			patList = strings.Split(filesPat, ",")
		}
	*/
	ch1 := readDir(ctx, ".", filesPat, sleepTime, maxFileSize, batchSize)

	done := ctx.Done()
	for {
		select {
		case i, ok := <-ch1:
			if !ok {
				break
			}
			log.Print(i.Name())
		case <-done:
			break
		}
	}
	/*
		ch2 := loopUrls(ctx, method, flag.Args(), ch1)
		ch3 := loopFiles(ctx, patList, ch2)
		doHttp(ctx, conns, ch3)
	*/

	// write memory profile if configured
	if memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			log.Print(err)
		} else {
			log.Print("Writing memory profile to ", memProfile)
			pprof.WriteHeapProfile(f)
			f.Close()
		}
	}
}
