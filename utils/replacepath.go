package utils

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
)

const pkgName = "github.com/icodezjb/fabric-study"

// ReplacePathInFile modify the path in config.yaml, replace `/absolute/path` by path.Join(goPath, "src", pkgName)
func ReplacePathInFile(config string) []byte {
	input, err := ioutil.ReadFile(config)
	if err != nil {
		Fatalf("ReadFile err:", err)
	}

	goPath := os.Getenv("GOPATH")

	fmt.Println("first-network in:", path.Join(goPath, "src", pkgName))
	return bytes.ReplaceAll(input, []byte("/absolute/path"), []byte(path.Join(goPath, "src", pkgName)))
}

// Fatalf formats a message to standard error and exits the program.
// The message is also printed to standard output if standard error
// is redirected to a different file.
func Fatalf(format string, args ...interface{}) {
	w := io.MultiWriter(os.Stdout, os.Stderr)
	if runtime.GOOS == "windows" {
		// The SameFile check below doesn't work on Windows.
		// stdout is unlikely to get redirected though, so just print there.
		w = os.Stdout
	} else {
		outf, _ := os.Stdout.Stat()
		errf, _ := os.Stderr.Stat()
		if outf != nil && errf != nil && os.SameFile(outf, errf) {
			w = os.Stderr
		}
	}
	fmt.Fprintf(w, "Fatal: "+format+"\n", args...)
	os.Exit(1)
}
