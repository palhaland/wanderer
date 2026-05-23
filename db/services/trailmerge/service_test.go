package trailmerge

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestFindMaintenanceCandidateTrails(t *testing.T) {
	// Initialize a test app (ephemeral, with temporary DB directory cloned from test-data/pb_data)
	app, err := tests.NewTestApp("../../../test-data/pb_data")
	if err != nil {
		t.Fatalf("Failed to initialize test app: %v", err)
	}
	defer app.Cleanup()

	// First, fetch the collections needed.
	trailsCol, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatalf("Failed to find trails collection: %v", err)
	}

	summitLogsCol, err := app.FindCollectionByNameOrId("summit_logs")
	if err != nil {
		t.Fatalf("Failed to find summit_logs collection: %v", err)
	}

	// Create dummy GPX file contents
	gpxContent1 := []byte(`<gpx version="1.1" creator="test"><trk><trkseg><trkpt lat="45.0" lon="10.0"></trkpt></trkseg></trk></gpx>`)
	gpxContent2 := []byte(`<gpx version="1.1" creator="test"><trk><trkseg><trkpt lat="46.0" lon="11.0"></trkpt></trkseg></trk></gpx>`)
	gpxContent3 := []byte(`<gpx version="1.1" creator="test"><trk><trkseg><trkpt lat="47.0" lon="12.0"></trkpt></trkseg></trk></gpx>`)

	// Find or create an author actor record
	var actorID string
	existingActors, err := app.FindRecordsByFilter("activitypub_actors", "1=1", "", 1, 0)
	if err == nil && len(existingActors) > 0 {
		actorID = existingActors[0].Id
	} else {
		// Create a user record first
		usersCol, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			t.Fatalf("Failed to find users collection: %v", err)
		}
		user := core.NewRecord(usersCol)
		user.Set("email", "test-unique-trail-merge@example.com")
		user.Set("username", "testuser_tm")
		user.Set("password", "password123")
		user.Set("passwordConfirm", "password123")
		if err := app.Save(user); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		// Create activitypub_actors record
		actorCol, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			t.Fatalf("Failed to find activitypub_actors collection: %v", err)
		}
		actor := core.NewRecord(actorCol)
		actor.Set("username", "testuser_actor")
		actor.Set("user", user.Id)
		if err := app.Save(actor); err != nil {
			t.Fatalf("Failed to create activitypub_actor: %v", err)
		}
		actorID = actor.Id
	}

	// Create Trail A (main target)
	trailA := core.NewRecord(trailsCol)
	trailA.Set("name", "Trail A")
	trailA.Set("author", actorID)
	fileA, err := filesystem.NewFileFromBytes(gpxContent1, "trailA.gpx")
	if err != nil {
		t.Fatalf("Failed to create fileA: %v", err)
	}
	trailA.Set("gpx", fileA)
	if err := app.Save(trailA); err != nil {
		t.Fatalf("Failed to save trail A: %v", err)
	}

	// Create Trail B (source/merged trail)
	trailB := core.NewRecord(trailsCol)
	trailB.Set("name", "Trail B")
	trailB.Set("author", actorID)
	fileB, err := filesystem.NewFileFromBytes(gpxContent2, "trailB.gpx")
	if err != nil {
		t.Fatalf("Failed to create fileB: %v", err)
	}
	trailB.Set("gpx", fileB)
	if err := app.Save(trailB); err != nil {
		t.Fatalf("Failed to save trail B: %v", err)
	}

	// Create Trail C (normal trail)
	trailC := core.NewRecord(trailsCol)
	trailC.Set("name", "Trail C")
	trailC.Set("author", actorID)
	fileC, err := filesystem.NewFileFromBytes(gpxContent3, "trailC.gpx")
	if err != nil {
		t.Fatalf("Failed to create fileC: %v", err)
	}
	trailC.Set("gpx", fileC)
	if err := app.Save(trailC); err != nil {
		t.Fatalf("Failed to save trail C: %v", err)
	}

	// Create a Summit Log on Trail A containing Trail B's GPX track file (identical content/hash)
	logRecord := core.NewRecord(summitLogsCol)
	logRecord.Set("trail", trailA.Id)
	logRecord.Set("author", actorID)
	fileBClone, err := filesystem.NewFileFromBytes(gpxContent2, "trailB_cloned.gpx")
	if err != nil {
		t.Fatalf("Failed to create fileBClone: %v", err)
	}
	logRecord.Set("gpx", fileBClone)
	if err := app.Save(logRecord); err != nil {
		t.Fatalf("Failed to save summit log: %v", err)
	}

	// Now call findMaintenanceCandidateTrails
	candidates, err := findMaintenanceCandidateTrails(app, actorID)
	if err != nil {
		t.Fatalf("findMaintenanceCandidateTrails failed: %v", err)
	}

	// We expect:
	// - Trail A is present (it's the target trail, and its GPX isn't in a summit log of a DIFFERENT trail)
	// - Trail B is EXCLUDED (its GPX is in a summit log of Trail A, which is a different trail)
	// - Trail C is present (it hasn't been merged)
	hasA := false
	hasB := false
	hasC := false
	for _, c := range candidates {
		if c.Id == trailA.Id {
			hasA = true
		}
		if c.Id == trailB.Id {
			hasB = true
		}
		if c.Id == trailC.Id {
			hasC = true
		}
	}

	if !hasA {
		t.Errorf("Expected candidate list to include Trail A, but it was missing")
	}
	if hasB {
		t.Errorf("Expected candidate list to EXCLUDE Trail B (already merged), but it was present")
	}
	if !hasC {
		t.Errorf("Expected candidate list to include Trail C, but it was missing")
	}
}

func TestGeneratePerfectTrack(t *testing.T) {
	app, err := tests.NewTestApp("../../../test-data/pb_data")
	if err != nil {
		t.Fatalf("Failed to initialize test app: %v", err)
	}
	defer app.Cleanup()

	trailsCol, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatalf("Failed to find trails collection: %v", err)
	}

	gpxContent1 := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test">
  <trk>
    <trkseg>
      <trkpt lat="45.0" lon="10.0"><ele>100.0</ele></trkpt>
      <trkpt lat="45.001" lon="10.001"><ele>110.0</ele></trkpt>
      <trkpt lat="45.002" lon="10.002"><ele>120.0</ele></trkpt>
      <trkpt lat="45.003" lon="10.003"><ele>130.0</ele></trkpt>
    </trkseg>
  </trk>
</gpx>`)

	gpxContent2 := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test">
  <trk>
    <trkseg>
      <trkpt lat="45.003" lon="10.003"><ele>130.0</ele></trkpt>
      <trkpt lat="45.002" lon="10.002"><ele>120.0</ele></trkpt>
      <trkpt lat="45.0015" lon="10.0015"><ele>115.0</ele></trkpt>
      <trkpt lat="45.001" lon="10.001"><ele>110.0</ele></trkpt>
      <trkpt lat="45.0" lon="10.0"><ele>100.0</ele></trkpt>
    </trkseg>
  </trk>
</gpx>`)

	var actorID string
	existingActors, err := app.FindRecordsByFilter("activitypub_actors", "1=1", "", 1, 0)
	if err == nil && len(existingActors) > 0 {
		actorID = existingActors[0].Id
	} else {
		usersCol, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			t.Fatalf("Failed to find users collection: %v", err)
		}
		user := core.NewRecord(usersCol)
		user.Set("email", "test-perfect-track@example.com")
		user.Set("username", "testuser_pt")
		user.Set("password", "password123")
		user.Set("passwordConfirm", "password123")
		if err := app.Save(user); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		actorCol, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			t.Fatalf("Failed to find activitypub_actors collection: %v", err)
		}
		actor := core.NewRecord(actorCol)
		actor.Set("username", "testuser_pt_actor")
		actor.Set("user", user.Id)
		if err := app.Save(actor); err != nil {
			t.Fatalf("Failed to create activitypub_actor: %v", err)
		}
		actorID = actor.Id
	}

	trail1 := core.NewRecord(trailsCol)
	trail1.Set("name", "Trail 1")
	trail1.Set("author", actorID)
	file1, err := filesystem.NewFileFromBytes(gpxContent1, "trail1.gpx")
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	trail1.Set("gpx", file1)
	if err := app.Save(trail1); err != nil {
		t.Fatalf("Failed to save trail 1: %v", err)
	}

	trail2 := core.NewRecord(trailsCol)
	trail2.Set("name", "Trail 2")
	trail2.Set("author", actorID)
	file2, err := filesystem.NewFileFromBytes(gpxContent2, "trail2.gpx")
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}
	trail2.Set("gpx", file2)
	if err := app.Save(trail2); err != nil {
		t.Fatalf("Failed to save trail 2: %v", err)
	}

	// Generate perfect track from trail1 and trail2 (trail2 is reversed direction)
	perfectGPX, err := GeneratePerfectTrack(app, []string{trail1.Id, trail2.Id})
	if err != nil {
		t.Fatalf("GeneratePerfectTrack failed: %v", err)
	}

	if len(perfectGPX.Tracks) != 1 {
		t.Fatalf("Expected 1 track, got %d", len(perfectGPX.Tracks))
	}
	if len(perfectGPX.Tracks[0].Segments) != 1 {
		t.Fatalf("Expected 1 segment, got %d", len(perfectGPX.Tracks[0].Segments))
	}

	pts := perfectGPX.Tracks[0].Segments[0].Points
	if len(pts) == 0 {
		t.Fatalf("Expected non-empty perfect track points")
	}

	// Check that it did not collapse (start and end points should be far apart)
	startPt := pts[0]
	endPt := pts[len(pts)-1]

	latDiff := startPt.Latitude - endPt.Latitude
	lonDiff := startPt.Longitude - endPt.Longitude
	distSq := latDiff*latDiff + lonDiff*lonDiff
	if distSq < 0.000005 {
		t.Errorf("Expected perfect track to stretch from end to end, but points collapsed (distance squared %f)", distSq)
	}

	// Check that we don't have collapsed coordinates in the middle (e.g. all points being the same)
	for i := 0; i < len(pts)-1; i++ {
		pA := pts[i]
		pB := pts[i+1]
		if pA.Latitude == pB.Latitude && pA.Longitude == pB.Longitude {
			t.Errorf("Point %d and %d are identical, indicating collapse/loss of resolution", i, i+1)
		}
	}
}
