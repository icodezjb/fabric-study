package utils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
)

const pkgName = "github.com/icodezjb/fabric-study"

// ReplacePathInFile modify the path in config.yaml, replace `/absolute/path` by path.Join(goPath, "src", pkgName)
func ReplacePathInFile(config string) []byte {
	input, err := ioutil.ReadFile(config)
	if err != nil {
		log.Fatalln("ReadFile err:", err)
	}

	goPath := os.Getenv("GOPATH")

	fmt.Println("first-network in:", path.Join(goPath, "src", pkgName))
	return bytes.ReplaceAll(input, []byte("/absolute/path"), []byte(path.Join(goPath, "src", pkgName)))
}
