package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/bogem/id3v2/v2"
)

type Song struct {
	Title  string
	Artist string
	Album  string
	Path   string
	Id     string
}

func idTransform(sb *strings.Builder, in string) {
	for _, ch := range in {
		ch = unicode.ToLower(ch)
		if ch == ' ' {
			sb.WriteRune('-')
		} else if ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			sb.WriteRune(ch)
		} // ignore all other characters
	}
}

func (s Song) CreateID() string {
	var sb strings.Builder
	idTransform(&sb, s.Title)
	sb.WriteRune('-')
	idTransform(&sb, s.Album)
	sb.WriteRune('-')
	idTransform(&sb, s.Artist)
	return sb.String()
}

//go:embed list.gohtml
var listTemplate string

var password string
var root string
var songs map[string]Song

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

func randomSong(songs map[string]Song) Song {
	i := rand.Intn(len(songs))
	j := 0
	for _, song := range songs {
		if i == j {
			return song
		}
		j++
	}
	return Song{} // should never happen!
}

func scanSongs(root string) map[string]Song {
	var songs map[string]Song = make(map[string]Song)
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
		tag, err := id3v2.Open(root+path, id3v2.Options{Parse: true})
		if err != nil {
			log.Fatal("Error while opening mp3 file: ", err)
		}
		defer tag.Close()
		song := Song{
			Title:  tag.Title(),
			Artist: tag.Artist(),
			Album:  tag.Album(),
			Path:   root + path,
		}
		song.Id = song.CreateID()
		songs[song.Id] = song
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
	path := randomSong(songs).Path
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

func listHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("foo").Parse(listTemplate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		return
	}
	err = t.Execute(w, songs)
}

func songHandler(w http.ResponseWriter, r *http.Request) {
	val := r.URL.Query().Get("id")
	if val == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "You need to provide a song.")
		return
	}
	v, ok := songs[val]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Song does not exist")
	}
	f, err := os.Open(v.Path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
	}
	defer f.Close()
	io.Copy(w, f)
}

func main() {
	rand.Seed(time.Now().UnixMicro())
	password = generatePassword()
	fmt.Printf("The password is %s\n", password)
	root = "/home/finnneon/Music/"
	songs = scanSongs(root)
	http.HandleFunc("/", loginWrapper(listHandler))
	http.HandleFunc("/random", loginWrapper(randomHandler))
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/song", loginWrapper(songHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
