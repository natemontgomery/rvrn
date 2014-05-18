package main

import (
  _ "github.com/bmizerany/pq"
  "github.com/astaxie/beedb"

  "os"
  "io/ioutil"
  "path/filepath"
  "database/sql"
  "encoding/json"
  "net/http"
  "html/template"
  "fmt"
  "log"
  "regexp"
  "time"

  "taglib"
)

var song_db beedb.Model

var BASE_URL = "http://developer.echonest.com/api/v4/"
var API_KEY = "make one, they are free :D"

var ROOT_MUSIC_FOLDER = "music"

var templates = template.Must(template.ParseFiles("index.html", "error.html", "library.html"))
var reqMerger = regexp.MustCompile(" ")

type Page struct {
  Title string
  Body []byte
  Artists []Artist
}

type echoResponse struct {
  Response rData "response"
}

type rData struct {
  Status rStatus "status"
  Artists []Artist "artists"
  Songs []Song "songs"
}

type rStatus struct {
  Version string "version"
  Code int "code"
  Message string "message"
}

type Artist struct {
  Name string "name"
  Id string "id"
  Activity float64 "hotttnesss"
  Songs []Song "songs"
}

type Song struct {
  Sid int `beedb:"PK"`
  ArtistId string "artist_id"
  ArtistName string "artist_name"
  Title string "title"
  Id string "id"
  Album string
  Comment string
  Genre string
  Year int
  TrackNumber int
  Length int
  Bitrate int
  Samplerate int
  Channels int
  Created time.Time
}

func grabArtists(artistQuery string) ([]Artist, []byte, error) {
  aResp, _, err := makeRequest(artistQuery, "artist")
  artist_list, rStatus, err := processResponse(aResp)

  //fmt.Print("artistResp: ", processedJSON, "\n")
  //fmt.Print(processedJSON.Response.Artists, "\n")

  return artist_list.Response.Artists, rStatus, err
}

func grabTracks(artist Artist) ([]Song, []byte, error) {
  tResp, _, err := makeRequest(artist.Name, "playlist")
  processedJSON, rStatus, err := processResponse(tResp)
  song_list := processedJSON.Response.Songs

  for i := 0; i < len(song_list); i++ {
    var song = processedJSON.Response.Songs[i]

    fmt.Print(song, "\n")

    t := make(map[string]interface{})
    t["title"] = song_list[i].Title

    song_db.SetTable("song").Where("title == ?", song_list[i].Title).Update(t)
  }

  //fmt.Print("tResp: ", processedJSON, "\n")
  //fmt.Print("tracks: ", processedJSON.Response.Songs, "\n")

  return song_list, rStatus, err
}

func makeRequest(request string, requestType string) ([]byte, []byte, error) {
  var queryString string
  var queryStatus []byte

  switch requestType {
  case "artist":
    queryString = BASE_URL+"artist/search?api_key="+API_KEY+"&name="+request
  case "track":
    queryString = BASE_URL+"song/search?api_key="+API_KEY+"&artist="+request+"&results=1"
  case "playlist":
    queryString = BASE_URL+"playlist/basic?api_key="+API_KEY+"&artist="+request+"&format=json&results=2&type=artist-radio"
  default:
    fmt.Print("failed to create request string, check request type")
  }

  //fmt.Print("\n")
  //fmt.Print(queryString)
  //fmt.Print("\n")

  requestString := string(reqMerger.ReplaceAll([]byte(queryString), []byte("+")))
  echonest_resp, err := http.Get(requestString)

  if err != nil {
    sbody := "Error while sending EchoNest query"
    queryStatus := []byte(sbody)
    return nil, queryStatus, err
  }

  jsonData, err := ioutil.ReadAll(echonest_resp.Body)
  echonest_resp.Body.Close()

  if err != nil {
    sbody := "Error while decoding JSON response"
    queryStatus := []byte(sbody)
    return nil, queryStatus, err
  }

  return jsonData, queryStatus, err
}

func processResponse(resp []byte) (echoResponse, []byte, error) {
  //byte slice for storing status of the request and error messages
  var respStatus []byte

  //var for holding JSON data in response
  var qResp echoResponse

  //decode JSON response and get artists and songs
  err := json.Unmarshal(resp, &qResp)

  //handle errors internal to the nginx server
  if err != nil {
    sbody := "Internal Error in Nginx, check code"
    respStatus := []byte(sbody)
    return qResp, respStatus, err
  }

  return qResp, respStatus, err
}

func search(searchString string) ([]byte, []Artist, error) {
  artistList, body, err := grabArtists(searchString)

  for i := 0; i < len(artistList); i++ {
    artistList[i].Songs, _, err = grabTracks(artistList[i])
  }

  //fmt.Print("finalArtists: ", artistList, "\n")

  return body, artistList, err
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
  title := "Search by Artist"
  sbody := "Type Artist name and hit Submit"
  body := []byte(sbody)
  p := &Page{Title: title, Body: body}
  renderTemplate(w, "index", p)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
  title := "Search by Artist"
  searchString := r.FormValue("name")
  
  body, aList, err := search(searchString)
  p := &Page{Title: title, Body: body, Artists: aList}
  //fmt.Print(err)
  //fmt.Print("\n")
  if err != nil {
    renderTemplate(w, "error", p)
  } else {
    renderTemplate(w, "index", p)
  }
}

func libraryHandler(w http.ResponseWriter, r *http.Request) {
  title := "Library"
  sbody := "Click 'Add' to add tracks"
  body := []byte(sbody)
  p := &Page{Title: title, Body: body}
  renderTemplate(w, "library", p)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
  err := templates.ExecuteTemplate(w, tmpl+".html", p)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}

func saveSongInfo(path string) {
  var song Song

  f := taglib.Open(path)
  if f == nil { return }
  defer f.Close()

  tags := f.GetTags()
  props := f.GetProperties()

  song.Album = tags.Album
  song.Comment = tags.Comment
  song.Genre = tags.Genre
  song.Year = tags.Year
  song.TrackNumber = tags.Track
  song.Length = props.Length
  song.Bitrate = props.Bitrate
  song.Samplerate = props.Samplerate
  song.Channels = props.Channels
  song.ArtistName = tags.Artist
  song.Created = time.Now()

  err := song_db.Save(&song)

  if err != nil {
    fmt.Println(err)
  }
}

func openDbConnection(engine string, arg_string string) {
  db, err := sql.Open(engine, arg_string)

  if err != nil {
      panic(err)
  }

  song_db = beedb.New(db, "pg")
}

func insertSongInfo(path string, f os.FileInfo, err error) error {
  saveSongInfo(path)
  return nil
}

func main() {
  openDbConnection("postgres", "user=nate dbname=rvrn")

  err := filepath.Walk(ROOT_MUSIC_FOLDER, insertSongInfo)

  fmt.Print(err, "\n")

  http.HandleFunc("/", indexHandler)
  http.HandleFunc("/search", searchHandler)
  http.HandleFunc("/library", libraryHandler)
  log.Fatal(http.ListenAndServe(":8080", nil))
}