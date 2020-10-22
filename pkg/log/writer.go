package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// writerOptions ...
type writerOptions struct {
	path    string
	name    string
	suffix  string
	cachesz int64
	span    time.Duration
}

func createWriterOptions() *writerOptions {
	ss := strings.Split(os.Args[0], "/")
	proc := ss[len(ss)-1]
	return &writerOptions{
		name:    proc,
		path:    "",
		suffix:  ".log",
		span:    time.Second,
		cachesz: 1024 * 1024 * 1024,
	}
}

// WriterOption ...
type WriterOption func(*writerOptions)

// CacheSize byte
func CacheSize(sz int64) WriterOption {
	return func(w *writerOptions) {
		w.cachesz = sz
	}
}

// Path ...
func Path(path string) WriterOption {
	return func(w *writerOptions) {
		w.path = path
	}
}

// Name ...
func Name(name string) WriterOption {
	return func(w *writerOptions) {
		w.name = name
	}
}

// Suffix ...
func Suffix(suffix string) WriterOption {
	return func(w *writerOptions) {
		w.suffix = suffix
	}
}

// Span ...
func Span(span time.Duration) WriterOption {
	return func(w *writerOptions) {
		w.span = span
	}
}

func createCacheWriter(opts ...WriterOption) io.WriteCloser {
	wt := &cacheWriter{
		opts:     createWriterOptions(),
		writebuf: &bytes.Buffer{},
		readbuf:  &bytes.Buffer{},
		quit:     make(chan bool),
	}
	for _, o := range opts {
		o(wt.opts)
	}
	go wt.run()
	return wt
}

type cacheWriter struct {
	opts     *writerOptions
	writebuf *bytes.Buffer
	readbuf  *bytes.Buffer
	lock     sync.Mutex
	mark     int
	name     string
	file     *os.File
	quit     chan bool
}

func (w *cacheWriter) Write(p []byte) (n int, err error) {
	w.lock.Lock()
	if int64(w.writebuf.Len()) > w.opts.cachesz {
		return 0, fmt.Errorf("no space")
	}
	n, err = w.writebuf.Write(p)
	w.lock.Unlock()
	return n, err
}

func (w *cacheWriter) Close() error {
	w.quit <- true
	<-w.quit
	w.lock.Lock()
	defer w.lock.Unlock()
	f := w.getFile()
	if f != nil {
		f.Write(w.readbuf.Bytes())
		f.Write(w.writebuf.Bytes())
		f.Sync()
		f.Close()
	}
	return nil
}

func (w *cacheWriter) run() {
	for {
		select {
		case <-w.quit:
			w.quit <- true
			return
		default:
		}
		w.lock.Lock()
		w.readbuf, w.writebuf = w.writebuf, w.readbuf
		w.lock.Unlock()

		if f := w.getFile(); f != nil {
			if _, err := f.Write(w.readbuf.Bytes()); err != nil {
				fmt.Printf("cacheWriter write error:%v \n", err)
			}
		}
		w.readbuf.Reset()
		time.Sleep(w.opts.span)
	}
}

func (w *cacheWriter) getFile() *os.File {
	now := time.Now()
	mark := now.YearDay()
	if w.mark != mark {
		w.mark = mark
		w.name = w.fileName(&now)
	}
	if w.file != nil && isExist(w.name) {
		return w.file
	}
	f, err := os.OpenFile(w.name, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("cacheWriter create file error:%v \n", err)
		return nil
	}
	w.file, f = f, w.file
	if f != nil {
		go func() {
			f.Sync()
			f.Close()
		}()
	}
	return w.file
}

func (w *cacheWriter) fileName(tm *time.Time) string {
	tmstr := tm.Format("2006_01_02")
	buf := bytes.Buffer{}
	buf.WriteString(w.opts.path)
	buf.WriteString(w.opts.name)
	buf.WriteByte('_')
	buf.WriteString(tmstr)
	buf.WriteString(w.opts.suffix)
	return buf.String()
}
