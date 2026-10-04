// Package gpx parses source geometry without inferring that it was ridden.
package gpx

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
)

const Version = "gpx-core-v1"

var ErrInvalid = errors.New("invalid or unsupported GPX")

type Point struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Elevation *float64 `json:"elevation_m"`
	RawTime   string   `json:"raw_time,omitempty"`
}
type Path struct {
	Name       string    `json:"name"`
	Structure  string    `json:"structure"`
	Locator    string    `json:"source_locator"`
	Segments   [][]Point `json:"segments"`
	PointCount int       `json:"point_count"`
}
type rawPoint struct {
	Lat       *float64 `xml:"lat,attr"`
	Lon       *float64 `xml:"lon,attr"`
	Elevation *float64 `xml:"ele"`
	Time      string   `xml:"time"`
}
type track struct {
	Name     string `xml:"name"`
	Segments []struct {
		Points []rawPoint `xml:"trkpt"`
	} `xml:"trkseg"`
}
type route struct {
	Name   string     `xml:"name"`
	Points []rawPoint `xml:"rtept"`
}

func Parse(raw []byte) ([]Path, error) {
	if len(raw) > 10*1024*1024 {
		return nil, fmt.Errorf("%w: file exceeds 10 MiB", ErrInvalid)
	}
	d := xml.NewDecoder(bytes.NewReader(raw))
	paths := []Path{}
	rootSeen, rootClosed := false, false
	ns := ""
	tracks, routes, total := 0, 0, 0
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: malformed XML", ErrInvalid)
		}
		switch t := tok.(type) {
		case xml.Directive:
			return nil, fmt.Errorf("%w: XML directives are not supported", ErrInvalid)
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				return nil, fmt.Errorf("%w: unexpected text", ErrInvalid)
			}
		case xml.StartElement:
			if !rootSeen {
				if t.Name.Local != "gpx" || (t.Name.Space != "" && t.Name.Space != "http://www.topografix.com/GPX/1/0" && t.Name.Space != "http://www.topografix.com/GPX/1/1") {
					return nil, fmt.Errorf("%w: expected GPX root", ErrInvalid)
				}
				rootSeen = true
				ns = t.Name.Space
				continue
			}
			if rootClosed {
				return nil, fmt.Errorf("%w: multiple root elements", ErrInvalid)
			}
			if t.Name.Space != ns {
				if err = d.Skip(); err != nil {
					return nil, fmt.Errorf("%w: malformed extension", ErrInvalid)
				}
				continue
			}
			var p Path
			var segments [][]rawPoint
			switch t.Name.Local {
			case "trk":
				var v track
				if err = d.DecodeElement(&v, &t); err != nil {
					return nil, fmt.Errorf("%w: malformed track", ErrInvalid)
				}
				tracks++
				p = Path{Name: v.Name, Structure: "gpx_track", Locator: fmt.Sprintf("/gpx/trk[%d]", tracks)}
				for _, s := range v.Segments {
					segments = append(segments, s.Points)
				}
			case "rte":
				var v route
				if err = d.DecodeElement(&v, &t); err != nil {
					return nil, fmt.Errorf("%w: malformed route", ErrInvalid)
				}
				routes++
				p = Path{Name: v.Name, Structure: "gpx_route", Locator: fmt.Sprintf("/gpx/rte[%d]", routes)}
				segments = [][]rawPoint{v.Points}
			default:
				if err = d.Skip(); err != nil {
					return nil, fmt.Errorf("%w: malformed GPX", ErrInvalid)
				}
				continue
			}
			if len(paths) >= 1000 {
				return nil, fmt.Errorf("%w: more than 1000 paths", ErrInvalid)
			}
			p.Segments = [][]Point{}
			for _, segment := range segments {
				points := []Point{}
				for _, point := range segment {
					if point.Lat == nil || point.Lon == nil || !finite(*point.Lat) || !finite(*point.Lon) || *point.Lat < -90 || *point.Lat > 90 || *point.Lon < -180 || *point.Lon > 180 || (point.Elevation != nil && !finite(*point.Elevation)) {
						return nil, fmt.Errorf("%w: missing or invalid coordinates/elevation", ErrInvalid)
					}
					total++
					if total > 250000 {
						return nil, fmt.Errorf("%w: more than 250000 points", ErrInvalid)
					}
					points = append(points, Point{Latitude: *point.Lat, Longitude: *point.Lon, Elevation: point.Elevation, RawTime: point.Time})
				}
				p.PointCount += len(points)
				p.Segments = append(p.Segments, points)
			}
			paths = append(paths, p)
		case xml.EndElement:
			if rootSeen && t.Name.Local == "gpx" && t.Name.Space == ns {
				rootClosed = true
			}
		}
	}
	if !rootSeen || !rootClosed {
		return nil, fmt.Errorf("%w: incomplete GPX", ErrInvalid)
	}
	if total == 0 {
		return nil, fmt.Errorf("%w: no track or route points found", ErrInvalid)
	}
	return paths, nil
}
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
