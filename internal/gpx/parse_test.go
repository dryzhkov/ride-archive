package gpx

import "testing"

func TestMultiplePathsAndSegments(t *testing.T) {
	raw := []byte(`<gpx xmlns="http://www.topografix.com/GPX/1/1"><trk><name>Recorded &amp; raw</name><trkseg><trkpt lat="45" lon="-120"><ele>42</ele><time>2026-09-05T10:00:00Z</time></trkpt></trkseg><trkseg><trkpt lat="46" lon="-121"/></trkseg></trk><rte><name>Supplied route</name><rtept lat="47" lon="-122"/></rte></gpx>`)
	paths, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[0].Name != "Recorded & raw" || paths[0].PointCount != 2 || len(paths[0].Segments) != 2 || paths[1].Structure != "gpx_route" {
		t.Fatal(paths)
	}
	if paths[0].Segments[0][0].RawTime != "2026-09-05T10:00:00Z" || paths[1].Segments[0][0].RawTime != "" || paths[1].Segments[0][0].Elevation != nil {
		t.Fatal(paths)
	}
}
func TestInvalidInputs(t *testing.T) {
	for _, raw := range []string{
		`<html/>`, `<gpx/>`, `<gpx><wpt lat="1" lon="2"/></gpx>`,
		`<gpx><trk><trkseg><trkpt lon="2"/></trkseg></trk></gpx>`,
		`<gpx><rte><rtept lat="91" lon="2"/></rte></gpx>`,
		`<gpx><rte><rtept lat="NaN" lon="2"/></rte></gpx>`,
		`<gpx><rte><rtept lat="1" lon="2"><ele>Inf</ele></rtept></rte></gpx>`,
		`<gpx><rte><rtept lat="1" lon="2"/></rte></gpx><gpx/>`,
		`<gpx><rte><rtept lat="1" lon="2"/></rte>`,
		`<!DOCTYPE gpx [<!ENTITY a SYSTEM "file:///tmp/no">]><gpx/>`,
	} {
		if _, err := Parse([]byte(raw)); err == nil {
			t.Errorf("accepted invalid: %s", raw)
		}
	}
}
func TestUnknownTimeRetainedAndEmptySegmentsPreserved(t *testing.T) {
	paths, err := Parse([]byte(`<gpx><trk><trkseg/><trkseg><trkpt lat="0" lon="0"><time>not-a-known-time</time></trkpt></trkseg></trk></gpx>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths[0].Segments) != 2 || len(paths[0].Segments[0]) != 0 || paths[0].Segments[1][0].RawTime != "not-a-known-time" {
		t.Fatal(paths)
	}
}
