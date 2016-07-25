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
	flag.DurationVar((*time.Duration)(&defaultCfg.Timeout), "timeout", time.Duration(defaultCfg.Timeout), "HTTP timeout.")
	flag.StringVar(&defaultCfg.FilesPat, "files", defaultCfg.FilesPat, "Pattern of files to post, like *.xml.")
	flag.StringVar(&defaultCfg.Method, "method", defaultCfg.Method, "HTTP method.")
	flag.StringVar(&defaultCfg.UseRequestId, "requestid", defaultCfg.UseRequestId, "Name of header to send a random GUID.")
	flag.BoolVar(&defaultCfg.NoCompress, "nocompress", defaultCfg.NoCompress, "Disable HTTP compression.")
	flag.BoolVar(&defaultCfg.NoKeepAlive, "nokeepalive", defaultCfg.NoKeepAlive, "Disable HTTP keep-alives.")
	flag.BoolVar(&defaultCfg.FileInfo, "fileinfo", defaultCfg.FileInfo, "Whether to send file information headers.")

	// processing
	flag.DurationVar((*time.Duration)(&defaultCfg.SleepTime), "sleep", time.Duration(defaultCfg.SleepTime), "Interval to wait when no files are found.")
	flag.Int64Var(&defaultCfg.MaxFileSize, "maxsize", defaultCfg.MaxFileSize, "Maximum file size to post.")
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
               __        __    __  ______  __
  ________  __/ /_____  / /_  / / / / __ \/ /
 / __  / / / / __/ __ \/ __ \/ / / / /_/ / /
/ /_/ / /_/ / /_/ /_/ / / / / /_/ / _, _/ /___
\__,_/\__,_/\__/\____/_/ /_/\____/_/ |_/_____/

A tool to continuously post files found in a folder.

Usage:
  autohurl [options] url1 [url2 ... urlN]

Example:
  autohurl -method POST -files "*.xml" -conns 10

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

	// Print settings
	fmt.Printf("addr = \"%s\"\n", addr)
	fmt.Printf("cpu = %d\n", cpus)
	fmt.Printf("wd = \"%s\"\n", workingDir)
	fmt.Printf("cpuprofile = \"%s\"\n", cpuProfile)
	fmt.Printf("memprofile = \"%s\"\n", memProfile)
	fmt.Println()

	// Print default folder settings
	fmt.Printf("[folders.*]\n%s\n", defaultCfg.String())

	// read folders
	var cfg map[string]*FolderCfg
	cfg, err = readConfig(flagcfg.Filename())
	if err != nil {
		log.Fatal(err)
	}
	if len(cfg) == 0 {
		log.Fatal("No folders configured to watch")
	}
	// set up default settings
	for i, _ := range cfg {
		cfg[i].SetDefaults(&defaultCfg)
		err = cfg[i].ParseHeaders()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("[folders.%s]\n%s\n", i, cfg[i].String())
		kubismus.Note("folders."+i, cfg[i].String())
		kubismus.Define(i+"_Errors", kubismus.COUNT, i+": Errors")
		kubismus.Define(i+"_Sent", kubismus.COUNT, i+": HTTP Posts")
		kubismus.Define(i+"_Sent", kubismus.SUM, i+": Bytes Sent")
		kubismus.Define(i+"_Received", kubismus.SUM, i+": Bytes Received")
		kubismus.Define(i+"_ResponseTime", kubismus.AVERAGE, i+": Average Time (s)")
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
	for name, fldr := range cfg {
		ch1 := readDir(ctx, name, fldr)
		go doHttp(ctx, name, fldr, ch1)
	}

	// write memory profile if configured
	if memProfile != "" {
		defer func() {
			f, err := os.Create(memProfile)
			if err != nil {
				log.Print(err)
			} else {
				log.Print("Writing memory profile to ", memProfile)
				pprof.WriteHeapProfile(f)
				f.Close()
			}
		}()
	}

	// status web site
	log.Fatal(http.ListenAndServe(addr, nil))
}
