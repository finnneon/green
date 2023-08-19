package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
)

func main() {
	root := "/home/finnneon/Music"
	fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if d.IsDir() {
			return nil // skip over directories
		}
		fmt.Println(path)
		return nil
	})
}
