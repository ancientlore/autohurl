package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DefaultCfg are configuration items that can be defaulted
type DefaultCfg struct {
	SleepTime    time.Duration `toml:"sleep"`       // Time to wait when no files are found
	Timeout      time.Duration `toml:"timeout"`     // HTTP timeout
	FilesPat     string        `toml:"files"`       // Pattern of files to look for
	Conns        int           `toml:"conns"`       // Number of concurrent HTTP connections
	Method       string        `toml:"method"`      // HTTP method (POST or PUT or PATCH, generally)
	MaxFileSize  int           `toml:"maxsize"`     // Maximum file size - larger files are moved or ignored
	NoCompress   bool          `toml:"nocompress"`  // Disable HTTP compression
	NoKeepAlive  bool          `toml:"nokeepalive"` // Disable HTTP keep-alive (not recommended)
	UseRequestId string        `toml:"requestid"`   // Enable X-RequestID header
	BatchSize    int           `toml:"batchsize"`   // Readdir batch size
	HeaderDelim  string        `toml:"hdrdelim"`    // Header delimiter
	HeaderText   string        `toml:"headers"`     // Text of headers
	headers      []hdr         `toml:"-"`           // Parsed headers
}

// Init sets the default values for DefaultCfg
func (c *DefaultCfg) Init() {
	c.BatchSize = 32 * 1024
	c.MaxFileSize = 1024 * 1024
	c.Method = "POST"
	c.Conns = 2
	c.FilesPat = "*.*"
	c.Timeout = 10 * time.Second
	c.SleepTime = time.Second
	c.HeaderDelim = "|"
}

// ParseHeaders parses the header text
func (c *DefaultCfg) ParseHeaders() error {
	var err error
	c.headers, err = parseHeaders(c.HeaderText, c.HeaderDelim)
	return err
}

// FolderCfg are config items for a folder
type FolderCfg struct {
	DefaultCfg                   // defaultable config settings
	Folder       string          `toml:"folder"`       // folder to watch
	URL          string          `toml:"url"`          // URL to post to
	MoveTo       string          `toml:"moveto"`       // folder to move files to after posting (otherwise deletes)
	MoveFailedTo string          `toml:"movefailedto"` // folder to move files that we cannot post
	client       *http.Client    `toml:"-"`            // http client for this folder
	transport    *http.Transport `toml:"-"`            // http transport for this folder
}

func (c *FolderCfg) String() string {
	return fmt.Sprintf(`folder: %s
files: %s
moveto: %s
movefailedto: %s
url: %s
timeout: %s
conns: %d
method: %s
compress: %b
keepalive: %b
requestid: %s
sleep: %s
maxfilesize: %d bytes
batchsize: %d
headers:
`, c.FilesPat, c.MoveTo, c.MoveFailedTo, c.URL, c.Timeout.String(), c.Conns, c.Method, !c.NoCompress, !c.NoKeepAlive, c.UseRequestId, c.SleepTime.String(), c.MaxFileSize, c.BatchSize)
}

// SetDefaults fills in any unset defaults from a default config object
func (c *FolderCfg) SetDefaults(from *DefaultCfg) {
	if c.SleepTime == 0 {
		c.SleepTime = from.SleepTime
	}
	if c.Timeout == 0 {
		c.Timeout = from.Timeout
	}
	if c.FilesPat == "" {
		c.FilesPat = from.FilesPat
	}
	if c.Conns == 0 {
		c.Conns = from.Conns
	}
	if c.Method == "" {
		c.Method = from.Method
	}
	if c.MaxFileSize == 0 {
		c.MaxFileSize = from.MaxFileSize
	}
	if c.NoCompress == false {
		c.NoCompress = from.NoCompress
	}
	if c.NoKeepAlive == false {
		c.NoKeepAlive = from.NoKeepAlive
	}
	if c.UseRequestId == "" {
		c.UseRequestId = from.UseRequestId
	}
	if c.BatchSize == 0 {
		c.BatchSize = from.BatchSize
	}
	if c.HeaderDelim == "" {
		c.HeaderDelim = from.HeaderDelim
	}
	if c.HeaderText == "" {
		c.HeaderText = from.HeaderText
	}
}

// create HTTP transport and client
// transport = &http.Transport{DisableKeepAlives: noKeepAlive, MaxIdleConnsPerHost: conns, DisableCompression: noCompress, ResponseHeaderTimeout: timeout}
// client = &http.Client{Transport: transport, Timeout: timeout}

// header mode
type hdrMode int

// header mode definitions
const (
	HdrSet hdrMode = iota
	HdrAdd
)

// header structure
type hdr struct {
	Key   string
	Value string
	Mode  hdrMode
}

// parseHeaders parses the header text with the given delimiter
func parseHeaders(headerText, headerDelim string) ([]hdr, error) {
	headers := make([]hdr, 0)
	headerText = strings.TrimSpace(headerText)
	if headerText != "" {
		arr := strings.Split(headerText, headerDelim)
		found := make(map[string]bool)
		for _, h := range arr {
			harr := strings.SplitN(h, ":", 2)
			if len(harr) != 2 {
				return nil, errors.New("Unable to parse header: " + h)
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
	return headers, nil
}

func readConfig(fn string) ([]FolderCfg, error) {
	return nil, nil
}
