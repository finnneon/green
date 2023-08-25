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
	"time"
)

var password string
var root string
var songs []string

func generatePassword() string {
	parts := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var sb strings.Builder
	for i := 0; i < 8; i++ {
		sb.WriteByte(parts[rand.Intn(len(parts))])
	}
	return sb.String()
}

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

func loginWrapper(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("PASSWORD")
		if err == http.ErrNoCookie || cookie.Value != password {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintln(w, "Unauthorized, go away please")
			return
		}
		f(w, r)
	}
}

func randomHandler(w http.ResponseWriter, r *http.Request) {
	path := root + randomSong(songs)
	f, err := os.Open(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		return
	}
	defer f.Close()
	io.Copy(w, f)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		http.SetCookie(w, &http.Cookie{
			Name:  "PASSWORD",
			Value: r.PostFormValue("password"),
		})
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<form action="/login" method="POST"><input type="password" id="password" name="password"></form>`)
}

func main() {
	rand.Seed(time.Now().UnixMicro())
	password = generatePassword()
	fmt.Printf("The password is %s\n", password)
	root = "/home/finnneon/Music/"
	songs = scanSongs(root)
	http.HandleFunc("/random", loginWrapper(randomHandler))
	http.HandleFunc("/login", loginHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
