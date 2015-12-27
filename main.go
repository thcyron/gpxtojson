package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/thcyron/go-gpx"
)

const progname = "gpxtojson"

func main() {
	if len(os.Args) != 2 {
		usage()
	}

	var r io.Reader

	if os.Args[1] == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(os.Args[1])
		if err != nil {
			die("%s", err)
		}
		defer f.Close()
		r = f
	}

	doc, err := parse(r)
	if err != nil {
		die("%s", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(doc); err != nil {
		die("%s", err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s <gpx file>\n", progname)
	os.Exit(2)
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", progname, fmt.Sprintf(format, args...))
	os.Exit(1)
}

type Doc struct {
	Version  string    `json:"version,omitempty"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Distance float64   `json:"distance"`
	Duration uint      `json:"duration"`
	Tracks   []Track   `json:"tracks"`
}

type Track struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Distance float64   `json:"distance"`
	Duration uint      `json:"duration"`
	Speed    float64   `json:"speed"`
	Segments []Segment `json:"segments"`
}

type Segment struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Distance float64   `json:"distance"`
	Duration uint      `json:"duration"`
	Speed    float64   `json:"speed"`
	Points   []Point   `json:"points"`
}

type Point struct {
	Lat                float64   `json:"lat"`
	Lon                float64   `json:"lon"`
	Elevation          float64   `json:"elevation"`
	Time               time.Time `json:"time"`
	Distance           float64   `json:"distance"`
	Duration           uint      `json:"duration"`
	Speed              float64   `json:"speed"`
	CumulativeDistance float64   `json:"cumulative_distance"`
	CumulativeDuration uint      `json:"cumulative_duration"`
}

func (p Point) DistanceTo(q Point) float64 {
	return haversine(p.Lat, p.Lon, q.Lat, q.Lon)
}

func parse(r io.Reader) (doc Doc, err error) {
	gpxdoc, err := gpx.NewDecoder(r).Decode()
	if err != nil {
		return doc, err
	}

	doc.Version = gpxdoc.Version
	doc.Tracks = make([]Track, 0, len(gpxdoc.Tracks))

	for _, track := range gpxdoc.Tracks {
		t := convertTrack(track)
		doc.Tracks = append(doc.Tracks, t)
		doc.Duration += t.Duration
		doc.Distance += t.Distance
	}

	if len(doc.Tracks) > 0 {
		t1, t2 := doc.Tracks[0], doc.Tracks[len(doc.Tracks)-1]
		doc.Start = t1.Start
		doc.End = t2.End
	}

	return doc, nil
}

func convertTrack(track gpx.Track) (t Track) {
	t.Segments = make([]Segment, 0, len(track.Segments))

	for _, segment := range track.Segments {
		s := convertSegment(segment)
		t.Segments = append(t.Segments, s)
		t.Duration += s.Duration
		t.Distance += s.Distance
	}

	if len(t.Segments) > 0 {
		s1, s2 := t.Segments[0], t.Segments[len(t.Segments)-1]
		t.Start = s1.Start
		t.End = s2.End
	}

	if t.Duration > 0 {
		t.Speed = float64(t.Distance/1000) / (float64(t.Duration) / 60 / 60)
	}

	return
}

func convertSegment(segment gpx.Segment) (s Segment) {
	s.Points = make([]Point, 0, len(segment.Points))

	for i, point := range segment.Points {
		p := convertPoint(point)
		if i == 0 {
			p.CumulativeDistance = 0
			p.CumulativeDuration = 0
		} else {
			q := s.Points[i-1]
			p.Distance = p.DistanceTo(q)
			p.Duration = uint(p.Time.Sub(q.Time).Seconds())
			p.CumulativeDistance = q.CumulativeDistance + p.Distance
			p.CumulativeDuration = q.CumulativeDuration + p.Duration

			if p.Duration > 0 {
				p.Speed = float64(p.Distance/1000) / (float64(p.Duration) / 60 / 60)
			}
		}
		s.Points = append(s.Points, p)
	}

	if len(s.Points) > 0 {
		p := s.Points[len(s.Points)-1]
		q := s.Points[0]

		s.Start = q.Time
		s.End = p.Time
		s.Distance = p.CumulativeDistance
		s.Duration = p.CumulativeDuration

		if s.Duration > 0 {
			s.Speed = float64(s.Distance/1000) / (float64(s.Duration) / 60 / 60)
		}
	}

	return
}

func convertPoint(point gpx.Point) (p Point) {
	p.Lat = point.Latitude
	p.Lon = point.Longitude
	p.Elevation = point.Elevation
	p.Time = point.Time

	return
}
