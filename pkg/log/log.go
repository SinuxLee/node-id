package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	prefix     string
	fileWriter logFileWriter
	filePath   = ""
	nameFormat = "2006_01_02"
)

// Init 模块初始化
func Init() bool {
	// set output
	ss := strings.Split(strings.Replace(os.Args[0], "\\", "/", -1), "/")
	prefix = ss[len(ss)-1]
	f := fileWriter.createFile(prefix, time.Now())
	if f == nil {
		return false
	}
	fileWriter.f = f
	fileWriter.stdout = true
	log.Logger = log.Output(&fileWriter)

	// time format
	zerolog.TimeFieldFormat = time.RFC3339 // "2006-01-02 15:04:05.000"

	// check
	go fileWriter.checkFileChange()
	go fileWriter.checkFileExists()

	// log
	log.Info().Msg("log init success!")

	return true
}

// InitWithPathAndFormat ...
func InitWithPathAndFormat(path, nameFmt string) bool {
	// set output
	filePath = path
	nameFormat = nameFmt
	ss := strings.Split(os.Args[0], "/")
	prefix = ss[len(ss)-1]
	f := fileWriter.createFile(prefix, time.Now())
	if f == nil {
		return false
	}
	fileWriter.f = f
	fileWriter.stdout = true
	log.Logger = log.Output(&fileWriter)

	// time format
	zerolog.TimeFieldFormat = time.RFC3339 // "2006-01-02 15:04:05.000"

	// check
	go fileWriter.checkFileChange()
	go fileWriter.checkFileExists()

	// log
	log.Info().Msg("log init success!")

	return true
}

// InitLogWithCacheWriter ...
// 返回的closer,在程序退出时需要调用，否则可能会有日志打不出来
func InitLogWithCacheWriter(path string) io.Closer {
	writer := createCacheWriter(Path(path))
	if writer == nil {
		return nil
	}

	zerolog.TimeFieldFormat = time.RFC3339 // "2006-01-02 15:04:05.000"
	log.Logger = log.Output(writer)

	log.Info().Msg("log init success!")
	return writer
}

// EnableStdOut 可以控制台输出
func EnableStdOut() {
	fileWriter.stdout = true
}

// DisableStdOut 不可以输出到控制台
func DisableStdOut() {
	fileWriter.stdout = false
}

// SetLevel 设置级别
func SetLevel(lv int) {
	log.Logger = log.Level(zerolog.Level(lv))
}

// Close 关闭日志系统
func Close() {
	fileWriter.Close()
}

type logFileWriter struct {
	mut    sync.Mutex
	f      *os.File
	stdout bool
}

func (l *logFileWriter) Write(p []byte) (n int, err error) {
	// 控制台输出
	if l.stdout {
		os.Stderr.Write(p)
	}
	// 文件输出
	tf := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&l.f)))
	f := (*os.File)(tf)
	if f != nil {
		l.mut.Lock()
		n, err := f.Write(p)
		l.mut.Unlock()
		return n, err
	}
	return 0, nil
}

func (l *logFileWriter) Close() {
	tf := atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&l.f)), nil)
	f := (*os.File)(tf)
	if f != nil {
		f.Sync()
		f.Close()
	}
}

func (l *logFileWriter) createFile(prefix string, now time.Time) *os.File {
	tstr := now.Format(nameFormat)
	name := fmt.Sprintf("%s%s_%s.log", filePath, prefix, tstr)
	file, _ := os.OpenFile(name, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	return file
}

func (l *logFileWriter) checkFileChange() {
	for {
		nextday := time.Now().Add(time.Hour * 24)
		nextday = time.Date(nextday.Year(), nextday.Month(), nextday.Day(), 0, 0, 0, 0, nextday.Location())
		tm := time.NewTimer(nextday.Sub(time.Now()))
		select {
		case <-tm.C:
			{
				f := l.createFile(prefix, nextday)
				if f != nil {
					oldf := l.f
					atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&l.f)), unsafe.Pointer(f))
					time.Sleep(10 * time.Second)
					oldf.Sync()
					oldf.Close()
				}
			}
		}
	}
}

func (l *logFileWriter) checkFileExists() {
	for {
		tm := time.NewTimer(time.Second)
		select {
		case <-tm.C:
			{
				now := time.Now()
				if !isLogExists(prefix, now) {
					f := fileWriter.createFile(prefix, now)
					if f != nil {
						atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&l.f)), unsafe.Pointer(f))
					}
				}
			}
		}
	}
}

func isLogExists(prefix string, now time.Time) bool {
	tstr := now.Format("2006_01_02")
	name := fmt.Sprintf("%s_%s.log", prefix, tstr)
	return isExist(name)
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// Info ...
func Info() *zerolog.Event {
	return log.Info()
}

// Debug ...
func Debug() *zerolog.Event {
	return log.Debug()
}

// Error ...
func Error() *zerolog.Event {
	return log.Error()
}

// Warn ...
func Warn() *zerolog.Event {
	return log.Warn()
}

// Fatal ...
func Fatal() *zerolog.Event {
	return log.Fatal()
}

// Print ...
func Print(v ...interface{}) {
	log.Print(v...)
}

// Printf ...
func Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}
