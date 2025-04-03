package lib_debug

import (
	"application/src/config"
	"fmt"
	"os"
	"path"
	"sync"
	"time"
)

const LogPermission = 0755

var fileMutex struct {
	sync.Mutex
}


func setupDir() error {
	var err error = nil
	if _, err = os.Stat(config.LogDir); err == nil {
		return nil
	}
	fileMutex.Lock()
	defer fileMutex.Unlock()
	err = os.MkdirAll(config.LogDir, LogPermission)
	return err
}

func write(filepath, msg string) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, LogPermission)
	if err != nil {
		return
	}
	defer file.Close()
	fmt.Fprintln(file, msg)
}

func log(name, prefix, msg string, args ...interface{}) {
	err := setupDir()
	if err != nil {
		return
	} else if 0 < len(args) {
		msg = fmt.Sprintf(msg, args...)
	}
	filepath := path.Join(config.LogDir, fmt.Sprintf("%s_%s.log", config.AppCodeName, name))
	timeString := time.Now().Format("2006-01-02 15:04:05")
	s := fmt.Sprintf("%s %s%s", timeString, prefix, msg)
	write(filepath, s)
	if config.GetIsLocal() {
		fmt.Println(s)
	}
}


func Debug(msg string, args ...interface{}) {
	if !config.GetIsDebugMode() {
		return
	}
	log("info", "[DEBUG]", msg, args...)
}

func Info(msg string, args ...interface{}) {
	log("info", "[INFO]", msg, args...)
}

func Error(msg string, args ...interface{}) {
	log("error", "[ERROR]", msg, args...)
}

func Log(filename, msg string, args ...interface{}) {
	log(filename, "", msg, args...)
}

