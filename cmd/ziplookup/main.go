package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
)

type location struct {
	country   string
	zip       string
	city      string
	stateLong string
	state     string
	county    string
	someCode  string
	lat       float64
	long      float64
}

var locations map[string]location
var host string
var port string
var dataDir string

func init() {
	flag.StringVar(&host, "host", "127.0.0.1", "Host to run service on (defaults to 127.0.0.1)")
	flag.StringVar(&port, "port", "8080", "Port to run service on (defaults to 8080)")
	flag.StringVar(&dataDir, "data-dir", "./data", "Data directory (defaults to ./data)")

	locations = make(map[string]location)
	f, err := os.Open("data/US.txt")
	if err != nil {
		log.Fatalln(err)
	}

	defer f.Close()

	log.Println("loading databse...")
	s := bufio.NewScanner(f)
	for s.Scan() {
		l, err := parse(s.Text())
		if err != nil {
			log.Fatalln(err)
		}
		locations[l.zip] = l
	}
	log.Println("loading complete")
}

func parse(s string) (location, error) {
	parts := strings.Split(s, "\t")
	loc := location{
		country:   parts[0],
		zip:       parts[1],
		city:      parts[2],
		stateLong: parts[3],
		state:     parts[4],
		county:    parts[5],
		someCode:  parts[6],
	}

	if l, err := strconv.ParseFloat(parts[9], 64); err != nil {
		return location{}, err
	} else {
		loc.lat = l
	}

	if l, err := strconv.ParseFloat(parts[10], 64); err != nil {
		return location{}, err
	} else {
		loc.long = l
	}
	return loc, nil
}

func main() {
	flag.Parse()

	beeline.Init(beeline.Config{
		WriteKey: os.Getenv("HONEYCOMB_API_KEY"),
		Dataset:  os.Getenv("HONEYCOMB_DATASET"),
	})
	defer beeline.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/{zip:[0-9]+}", getLocationByZip)
	hostPort := fmt.Sprintf("%s:%s", host, port)
	log.Printf("listening on %s...", hostPort)
	if err := http.ListenAndServe(hostPort, hnynethttp.WrapHandler(r)); err != nil {
		log.Fatalln(err)
	}
}

func getLocationByZip(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := beeline.StartSpan(ctx, "lookup")
	defer span.Send()

	zip := chi.URLParam(r, "zip")
	if loc, found := locations[zip]; found {
		if _, err := w.Write([]byte(spew.Sdump(loc))); err != nil {
			log.Println(err)
		}
		// beeline.AddField(ctx, "interesting_thing", "banana")
		return
	}
	http.Error(w, http.StatusText(404), 404)
}
