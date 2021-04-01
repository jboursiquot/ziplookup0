package main

import (
	"bufio"
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

func init() {
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
	beeline.Init(beeline.Config{
		WriteKey: os.Getenv("HONEYCOMB_API_KEY"),
		Dataset:  "ZipLookup",
	})
	defer beeline.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/{zip:[0-9]+}", getLocationByZip)
	log.Println("listening on :8080...")
	http.ListenAndServe(":8080", hnynethttp.WrapHandler(r))

}

func getLocationByZip(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := beeline.StartSpan(ctx, "lookup")
	defer span.Send()

	zip := chi.URLParam(r, "zip")
	if loc, found := locations[zip]; found {
		w.Write([]byte(spew.Sdump(loc)))
		// beeline.AddField(ctx, "interesting_thing", "banana")
		return
	}
	http.Error(w, http.StatusText(404), 404)
}
