package main

import (
	"bytes"
	"errors"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

// duration exists for TOML to handle durations
type duration time.Duration

// Unmarshals a duration from a string
func (d *duration) UnmarshalText(text []byte) error {
	dt, err := time.ParseDuration(string(text))
	if err == nil {
		*d = duration(dt)
	}
	return err
}

// Marshals a duration as a string
func (d duration) MarshalText() (text []byte, err error) {
	return []byte(time.Duration(d).String()), nil
}

// Prints a duration as a string
func (d duration) String() string {
	return time.Duration(d).String()
}

// DefaultCfg are configuration items that can be defaulted
type DefaultCfg struct {
	SleepTime    duration `toml:"sleep"`       // Time to wait when no files are found
	Timeout      duration `toml:"timeout"`     // HTTP timeout
	FilesPat     string   `toml:"files"`       // Pattern of files to look for
	Conns        int      `toml:"conns"`       // Number of concurrent HTTP connections
	Method       string   `toml:"method"`      // HTTP method (POST or PUT or PATCH, generally)
	MaxFileSize  int      `toml:"maxsize"`     // Maximum file size - larger files are moved or ignored
	NoCompress   bool     `toml:"nocompress"`  // Disable HTTP compression
	NoKeepAlive  bool     `toml:"nokeepalive"` // Disable HTTP keep-alive (not recommended)
	UseRequestId string   `toml:"requestid"`   // Enable X-RequestID header
	BatchSize    int      `toml:"batchsize"`   // Readdir batch size
	HeaderDelim  string   `toml:"hdrdelim"`    // Header delimiter
	HeaderText   string   `toml:"headers"`     // Text of headers
	FileInfo     bool     `toml:"fileinfo"`    // Whether to pass file info
	Headers      []hdr    `toml:"-"`           // Parsed headers
}

// Init sets the default values for DefaultCfg
func (c *DefaultCfg) Init() {
	c.BatchSize = 32 * 1024
	c.MaxFileSize = 1024 * 1024
	c.Method = "POST"
	c.Conns = 2
	c.FilesPat = "*.*"
	c.Timeout = duration(10 * time.Second)
	c.SleepTime = duration(time.Second)
	c.HeaderDelim = "|"
}

// ParseHeaders parses the header text
func (c *DefaultCfg) ParseHeaders() error {
	var err error
	c.Headers, err = parseHeaders(c.HeaderText, c.HeaderDelim)
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

// Prints config in TOML
func (c *FolderCfg) String() string {
	var b bytes.Buffer
	enc := toml.NewEncoder(&b)
	err := enc.Encode(c)
	if err != nil {
		log.Fatal(err)
	}
	return string(b.Bytes())
}

// SetDefaults fills in any unset defaults from a default config object
func (c *FolderCfg) SetDefaults(from *DefaultCfg) {
	if c.SleepTime == duration(0) {
		c.SleepTime = from.SleepTime
	}
	if c.Timeout == duration(0) {
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
	if c.FileInfo == false {
		c.FileInfo = from.FileInfo
	}
}

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

// conf is the non-default data present in the config file (the folder data only)
type conf struct {
	Folders map[string]*FolderCfg `toml:"folders"`
}

// readConfig reads the folder data from the config file
func readConfig(fn string) (map[string]*FolderCfg, error) {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	var c conf
	if _, err := toml.Decode(string(data), &c); err != nil {
		return nil, err
	}
	return c.Folders, nil
}
