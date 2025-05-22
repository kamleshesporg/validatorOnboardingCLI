package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/hashicorp/go-getter"
)

func main() {

	url := "git::https://github.com/kamleshesporg/mrmintchain/mrkamlesh"
	destination := "chain/mrmintd"
	if !exists(destination) {
		err := getter.GetFile(destination, url)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("✅ File downloaded successfully to:", destination)
		// Optionally, check file permissions and make it executable
		err = os.Chmod(destination, 0755)
		if err != nil {
			log.Println("Error setting file permissions:", err)
		}
	} else {
		fmt.Println("✅ File already downloaded to:", destination)
	}

}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return false
}
