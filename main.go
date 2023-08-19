package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
)

// isMP3 checks if a file is an MP3. Ignores other file formats not supported by Safari
func isMP3(path string) bool {
	parts := strings.Split(path, ".")
	if len(parts) == 1 {
		return false
	}
	extension := parts[len(parts)-1]
	return extension == "mp3"
}

func randomSong(songs []string) string {
	i := rand.Intn(len(songs))
	return songs[i]
}

func scanSongs(root string) []string {
	var songs []string
	fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if d.IsDir() {
			return nil // skip over directories
		}
		if !isMP3(path) {
			return nil
		}
		songs = append(songs, path)
		return nil
	})
	return songs
}

func main() {
	root := "/home/finnneon/Music/"
	songs := scanSongs(root)
	http.HandleFunc("/random", func(w http.ResponseWriter, r *http.Request) {
		path := root + randomSong(songs)
		f, err := os.Open(path)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
