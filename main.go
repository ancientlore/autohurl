package main

import (
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
	"time"
)

// github.com/ancientlore/binder is used to package the web files into the executable.
//go:generate binder -package main -o webcontent.go media/*.png

var (
	addr       string = ":8080"
	cpuProfile string
	memProfile string
	cpus       int
	workingDir string
	version    bool
	help       bool

	defaultCfg DefaultCfg
)

func init() {

	// Set initial settings
	defaultCfg.Init()

	// http service/status address
	flag.StringVar(&addr, "addr", addr, "HTTP service address for monitoring.")

	// http post settings
	flag.IntVar(&defaultCfg.Conns, "conns", defaultCfg.Conns, "Number of concurrent HTTP connections.")
	flag.DurationVar(&defaultCfg.Timeout, "timeout", defaultCfg.Timeout, "HTTP timeout.")
	flag.StringVar(&defaultCfg.FilesPat, "files", defaultCfg.FilesPat, "Pattern of files to post, like *.xml.")
	flag.StringVar(&defaultCfg.Method, "method", defaultCfg.Method, "HTTP method.")
	flag.StringVar(&defaultCfg.UseRequestId, "requestid", defaultCfg.UseRequestId, "Name of header to send a random GUID.")
	flag.BoolVar(&defaultCfg.NoCompress, "nocompress", defaultCfg.NoCompress, "Disable HTTP compression.")
	flag.BoolVar(&defaultCfg.NoKeepAlive, "nokeepalive", defaultCfg.NoKeepAlive, "Disable HTTP keep-alives.")

	// processing
	flag.DurationVar(&defaultCfg.SleepTime, "sleep", defaultCfg.SleepTime, "Interval to wait when no files are found.")
	flag.IntVar(&defaultCfg.MaxFileSize, "maxsize", defaultCfg.MaxFileSize, "Maximum file size to post.")
	flag.IntVar(&defaultCfg.BatchSize, "batchsize", defaultCfg.BatchSize, "Readdir batch size.")

	// headers
	flag.StringVar(&defaultCfg.HeaderDelim, "hdrdelim", defaultCfg.HeaderDelim, "Delimiter for HTTP headers specified with -header.")
	flag.StringVar(&defaultCfg.HeaderText, "headers", defaultCfg.HeaderText, "HTTP headers, delimited by -hdrdelim.")

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

	// parse default headers - make sure they work
	var err error
	err = defaultCfg.ParseHeaders()
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
	kubismus.Note("Processors", fmt.Sprintf("%d of %d", runtime.GOMAXPROCS(0), runtime.NumCPU()))
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

	// read folders

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
	ch1 := readDir(ctx, ".", defaultCfg.FilesPat, defaultCfg.SleepTime, defaultCfg.MaxFileSize, defaultCfg.BatchSize)

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
