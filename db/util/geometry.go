package util

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pocketbase/pocketbase/core"
	"github.com/tkrajina/gpxgo/gpx"
)

type TrailGeometryMetrics struct {
	MeanDistanceMeters  float64
	MaxDistanceMeters   float64
	StartDistanceMeters float64
	EndDistanceMeters   float64
}

type TrailPoint struct {
	Lat float64
	Lon float64
	Ele float64
}

func TrailCoordinates(app core.App, r *core.Record) ([][2]float64, error) {
	points, err := TrailCoordinatesPoints(app, r)
	if err != nil {
		return nil, err
	}

	coords := make([][2]float64, 0, len(points))
	for _, pt := range points {
		coords = append(coords, [2]float64{pt.Lat, pt.Lon})
	}

	return coords, nil
}

func TrailCoordinatesPoints(app core.App, r *core.Record) ([]TrailPoint, error) {
	gpxPath := r.GetString("gpx")
	if gpxPath == "" {
		return nil, nil
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, err
	}
	defer fsys.Close()

	reader, err := fsys.GetReader(r.BaseFilesPath() + "/" + gpxPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	content := new(bytes.Buffer)
	if _, err := io.Copy(content, reader); err != nil {
		return nil, err
	}

	gpxData, err := gpx.Parse(content)
	if err != nil {
		return nil, err
	}

	points := make([]TrailPoint, 0)
	for _, trk := range gpxData.Tracks {
		for _, seg := range trk.Segments {
			for _, pt := range seg.Points {
				points = append(points, TrailPoint{
					Lat: pt.Latitude,
					Lon: pt.Longitude,
					Ele: pt.Elevation.Value(),
				})
			}
		}
	}

	return points, nil
}

func TrailGeometrySimilarity(app core.App, a *core.Record, b *core.Record) (*TrailGeometryMetrics, error) {
	aCoords, err := TrailCoordinates(app, a)
	if err != nil {
		return nil, fmt.Errorf("load source geometry: %w", err)
	}
	bCoords, err := TrailCoordinates(app, b)
	if err != nil {
		return nil, fmt.Errorf("load target geometry: %w", err)
	}

	return CompareTrailCoordinates(aCoords, bCoords)
}

func CompareTrailCoordinates(aCoords [][2]float64, bCoords [][2]float64) (*TrailGeometryMetrics, error) {
	if len(aCoords) < 2 || len(bCoords) < 2 {
		return nil, fmt.Errorf("missing geometry")
	}

	a := resampleTrailCoordinates(aCoords, 64)
	b := resampleTrailCoordinates(bCoords, 64)
	if len(a) < 2 || len(b) < 2 {
		return nil, fmt.Errorf("missing geometry")
	}

	return compareSampledTrails(a, b), nil
}

func compareSampledTrails(a []TrailPoint, b []TrailPoint) *TrailGeometryMetrics {
	count := min(len(a), len(b))
	if count == 0 {
		return &TrailGeometryMetrics{}
	}

	sum := 0.0
	maxDistance := 0.0
	for i := 0; i < count; i++ {
		distance := HaversineDistanceMeters(a[i].Lat, a[i].Lon, b[i].Lat, b[i].Lon)
		sum += distance
		if distance > maxDistance {
			maxDistance = distance
		}
	}

	return &TrailGeometryMetrics{
		MeanDistanceMeters:  sum / float64(count),
		MaxDistanceMeters:   maxDistance,
		StartDistanceMeters: HaversineDistanceMeters(a[0].Lat, a[0].Lon, b[0].Lat, b[0].Lon),
		EndDistanceMeters:   HaversineDistanceMeters(a[count-1].Lat, a[count-1].Lon, b[count-1].Lat, b[count-1].Lon),
	}
}

func resampleTrailCoordinates(coords [][2]float64, targetPoints int) []TrailPoint {
	points := make([]TrailPoint, 0, len(coords))
	for _, coord := range coords {
		points = append(points, TrailPoint{Lat: coord[0], Lon: coord[1]})
	}
	return resampleTrailPoints(points, targetPoints)
}

func resampleTrailPoints(points []TrailPoint, targetPoints int) []TrailPoint {
	if targetPoints <= 0 {
		return nil
	}

	if len(points) == 0 {
		return nil
	}

	// If we have only one point or target is 1, return the point repeated
	if len(points) == 1 || targetPoints == 1 {
		resampled := make([]TrailPoint, targetPoints)
		for i := 0; i < targetPoints; i++ {
			resampled[i] = points[0]
		}
		return resampled
	}

	cumulative := make([]float64, len(points))
	total := 0.0
	for i := 1; i < len(points); i++ {
		total += HaversineDistanceMeters(points[i-1].Lat, points[i-1].Lon, points[i].Lat, points[i].Lon)
		cumulative[i] = total
	}

	resampled := make([]TrailPoint, targetPoints)
	if total == 0 {
		for i := 0; i < targetPoints; i++ {
			resampled[i] = points[0]
		}
		return resampled
	}

	for i := 0; i < targetPoints; i++ {
		targetDistance := (float64(i) / float64(targetPoints-1)) * total
		resampled[i] = interpolateTrailPoint(points, cumulative, targetDistance)
	}

	return resampled
}

func interpolateTrailPoint(points []TrailPoint, cumulative []float64, targetDistance float64) TrailPoint {
	if targetDistance <= 0 {
		return points[0]
	}
	lastIndex := len(points) - 1
	if targetDistance >= cumulative[lastIndex] {
		return points[lastIndex]
	}

	for i := 1; i < len(points); i++ {
		if cumulative[i] < targetDistance {
			continue
		}

		prevDistance := cumulative[i-1]
		nextDistance := cumulative[i]
		if nextDistance == prevDistance {
			return points[i]
		}

		ratio := (targetDistance - prevDistance) / (nextDistance - prevDistance)
		return TrailPoint{
			Lat: points[i-1].Lat + (points[i].Lat-points[i-1].Lat)*ratio,
			Lon: points[i-1].Lon + (points[i].Lon-points[i-1].Lon)*ratio,
			Ele: points[i-1].Ele + (points[i].Ele-points[i-1].Ele)*ratio,
		}
	}

	return points[lastIndex]
}

func AverageTrailPoints(allPoints [][]TrailPoint) []TrailPoint {
	if len(allPoints) == 0 {
		return nil
	}
	if len(allPoints) == 1 {
		return allPoints[0]
	}

	// 1. Choose a reference track using the Medoid approach with coarse tracks (100 points).
	coarseTracks := make([][]TrailPoint, len(allPoints))
	for i, track := range allPoints {
		coarseTracks[i] = resampleTrailPoints(track, 100)
	}

	bestIdx := 0
	minTotalDist := -1.0
	for i, tA := range coarseTracks {
		totalDist := 0.0
		for j, tB := range coarseTracks {
			if i == j {
				continue
			}
			totalDist += trackDistance(tA, tB)
		}
		if minTotalDist < 0 || totalDist < minTotalDist {
			minTotalDist = totalDist
			bestIdx = i
		}
	}

	// 2. Resample the reference track and other tracks to 1000 points for high-quality, fast averaging.
	refTrack := resampleTrailPoints(allPoints[bestIdx], 1000)

	alignedTrails := make([][]TrailPoint, len(allPoints))
	alignedTrails[bestIdx] = refTrack

	for k := 0; k < len(allPoints); k++ {
		if k == bestIdx {
			continue
		}
		track := resampleTrailPoints(allPoints[k], 1000)
		if len(track) == 0 {
			continue
		}

		refStart := refTrack[0]
		refEnd := refTrack[len(refTrack)-1]
		trackStart := track[0]
		trackEnd := track[len(track)-1]

		dStartStart := HaversineDistanceMeters(refStart.Lat, refStart.Lon, trackStart.Lat, trackStart.Lon)
		dEndEnd := HaversineDistanceMeters(refEnd.Lat, refEnd.Lon, trackEnd.Lat, trackEnd.Lon)
		dStartEnd := HaversineDistanceMeters(refStart.Lat, refStart.Lon, trackEnd.Lat, trackEnd.Lon)
		dEndStart := HaversineDistanceMeters(refEnd.Lat, refEnd.Lon, trackStart.Lat, trackStart.Lon)

		if (dStartEnd + dEndStart) < (dStartStart + dEndEnd) {
			// Reverse track to align orientation
			reversed := make([]TrailPoint, len(track))
			for i := 0; i < len(track); i++ {
				reversed[i] = track[len(track)-1-i]
			}
			alignedTrails[k] = reversed
		} else {
			alignedTrails[k] = track
		}
	}

	// 3. For each point in the reference track, find the closest spatial point in each of the other tracks and average them.
	// Since both reference and other tracks are resampled to 1000 points, finding the closest point is extremely fast.
	result := make([]TrailPoint, len(refTrack))
	for i, refPt := range refTrack {
		sumLat := refPt.Lat
		sumLon := refPt.Lon
		sumEle := refPt.Ele
		count := 1.0

		for k := 0; k < len(alignedTrails); k++ {
			if k == bestIdx {
				continue
			}
			otherTrack := alignedTrails[k]
			if len(otherTrack) == 0 {
				continue
			}

			closestPt := findClosestPoint(refPt, otherTrack)
			sumLat += closestPt.Lat
			sumLon += closestPt.Lon
			sumEle += closestPt.Ele
			count++
		}

		result[i] = TrailPoint{
			Lat: sumLat / count,
			Lon: sumLon / count,
			Ele: sumEle / count,
		}
	}

	return result
}

func trackDistance(trackA, trackB []TrailPoint) float64 {
	if len(trackA) == 0 || len(trackB) == 0 {
		return 0
	}
	sumDist := 0.0
	for _, pt := range trackA {
		minDist := -1.0
		for _, opt := range trackB {
			dist := HaversineDistanceMeters(pt.Lat, pt.Lon, opt.Lat, opt.Lon)
			if minDist < 0 || dist < minDist {
				minDist = dist
			}
		}
		sumDist += minDist
	}
	return sumDist / float64(len(trackA))
}

func findClosestPoint(pt TrailPoint, track []TrailPoint) TrailPoint {
	closest := track[0]
	minDist := HaversineDistanceMeters(pt.Lat, pt.Lon, closest.Lat, closest.Lon)

	for i := 1; i < len(track); i++ {
		dist := HaversineDistanceMeters(pt.Lat, pt.Lon, track[i].Lat, track[i].Lon)
		if dist < minDist {
			minDist = dist
			closest = track[i]
		}
	}
	return closest
}
