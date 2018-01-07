package utils

import (
	"log"
	"os"
	"path/filepath"
)

// utils is the package that defines the various subroutines used by Big Disk API. they are all functions.

// Pwd finds the directory of the main process (which would be ../) so that Prometheus can find ../public
// Mainly, this is necessary so that Prometheus can be started in rc.local. The directory becomes relative to the root when started as a startup process. Hence, the ./public folder will no longer be locatable through relative positioning. Pwd ensures you don't have to hardcode the path of the program directory.
func Pwd() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}
