/**************************************************************************************************
** Metadata sync for immich-front-back.
** After stacking, propagates the primary asset's date (date only, preserving sub-asset time),
** tags, and people (best-effort face assignment) to all sub-assets in each stack.
**************************************************************************************************/

package main

import (
	"fmt"
	"time"

	"github.com/sd-leighericksen/immich-front-back/pkg/immich"
	"github.com/sd-leighericksen/immich-front-back/pkg/utils"
	"github.com/sirupsen/logrus"
)

// syncStackMetadata runs after stacking. For every stack it syncs the primary asset's
// metadata (date, tags, people) to all sub-assets, according to the enabled sync flags.
func syncStackMetadata(client *immich.Client, stacks [][]utils.TAsset, logger *logrus.Logger) {
	if !syncMetadataEnabled {
		return
	}
	logger.Infof("=== Metadata Sync ===")
	synced := 0
	for _, stack := range stacks {
		if len(stack) < 2 {
			continue
		}
		primary := stack[0]
		subs := stack[1:]

		// Fetch full details for the primary to get people and tags.
		primaryDetails, err := client.GetAssetDetails(primary.ID)
		if err != nil {
			logger.Warnf("Metadata sync: could not fetch details for primary %s (%s): %v", primary.OriginalFileName, primary.ID, err)
			continue
		}

		if syncDate {
			for _, sub := range subs {
				syncDateForAsset(client, primaryDetails, sub, logger)
			}
		}
		if syncTags {
			syncTagsForStack(client, primaryDetails, subs, logger)
		}
		if syncPeople {
			syncPeopleForStack(client, primaryDetails, subs, logger)
		}
		synced++
	}
	logger.Infof("=== Metadata Sync complete (%d stacks processed) ===", synced)
}

// syncDateForAsset updates a sub-asset's date to match the primary's date (YYYY-MM-DD),
// while preserving the sub-asset's original time (HH:MM:SS).
func syncDateForAsset(client *immich.Client, primary utils.TAsset, sub utils.TAsset, logger *logrus.Logger) {
	primaryTime, err := parseImmichDateTime(primary.LocalDateTime)
	if err != nil {
		logger.Warnf("\tSync date: cannot parse primary LocalDateTime %q: %v", primary.LocalDateTime, err)
		return
	}
	subTime, err := parseImmichDateTime(sub.LocalDateTime)
	if err != nil {
		logger.Warnf("\tSync date: cannot parse sub LocalDateTime %q: %v", sub.LocalDateTime, err)
		return
	}

	// Idempotency: skip if date parts already match.
	if primaryTime.Year() == subTime.Year() &&
		primaryTime.Month() == subTime.Month() &&
		primaryTime.Day() == subTime.Day() {
		logger.Debugf("\tSync date: %s already matches primary date, skipping", sub.OriginalFileName)
		return
	}

	// Combine primary's date with sub's time, keeping sub's timezone.
	combined := time.Date(
		primaryTime.Year(), primaryTime.Month(), primaryTime.Day(),
		subTime.Hour(), subTime.Minute(), subTime.Second(), 0,
		subTime.Location(),
	)
	newDateTime := combined.Format("2006-01-02T15:04:05")
	logger.Infof("\tSync date: %s  %s → %s", sub.OriginalFileName, sub.LocalDateTime[:10], newDateTime[:10])
	if err := client.UpdateAssetDateTime(sub.ID, newDateTime); err != nil {
		logger.Warnf("\tSync date: failed to update %s: %v", sub.OriginalFileName, err)
	}
}

// syncTagsForStack applies all of the primary asset's tags to the sub-assets in one batch per tag.
func syncTagsForStack(client *immich.Client, primary utils.TAsset, subs []utils.TAsset, logger *logrus.Logger) {
	if len(primary.Tags) == 0 {
		logger.Debugf("\tSync tags: primary %s has no tags", primary.OriginalFileName)
		return
	}
	subIDs := make([]string, len(subs))
	for i, s := range subs {
		subIDs[i] = s.ID
	}
	for _, tag := range primary.Tags {
		logger.Infof("\tSync tags: applying tag %q to %d sub-asset(s)", tag.Name, len(subIDs))
		if err := client.TagAssets(tag.ID, subIDs); err != nil {
			logger.Warnf("\tSync tags: failed to apply tag %q: %v", tag.Name, err)
		}
	}
}

// syncPeopleForStack attempts best-effort face assignment: for each sub-asset's unassigned faces,
// assigns the primary's recognized person — only when the counts are unambiguous (1 person, 1 face).
func syncPeopleForStack(client *immich.Client, primary utils.TAsset, subs []utils.TAsset, logger *logrus.Logger) {
	if len(primary.People) == 0 {
		logger.Debugf("\tSync people: primary %s has no recognized people", primary.OriginalFileName)
		return
	}
	for _, sub := range subs {
		faces, err := client.GetAssetFaces(sub.ID)
		if err != nil {
			logger.Warnf("\tSync people: could not fetch faces for %s: %v", sub.OriginalFileName, err)
			continue
		}
		unassigned := make([]utils.TFace, 0, len(faces))
		for _, f := range faces {
			if f.Person == nil {
				unassigned = append(unassigned, f)
			}
		}
		if len(unassigned) == 0 {
			logger.Debugf("\tSync people: %s has no unassigned faces", sub.OriginalFileName)
			continue
		}
		if len(unassigned) == 1 && len(primary.People) == 1 {
			person := primary.People[0]
			logger.Infof("\tSync people: assigning %q to unassigned face on %s", person.Name, sub.OriginalFileName)
			if err := client.AssignPersonToFace(unassigned[0].ID, person.ID); err != nil {
				logger.Warnf("\tSync people: failed to assign person: %v", err)
			}
		} else {
			logger.Debugf("\tSync people: skipping %s — ambiguous (%d unassigned faces, %d people on primary)",
				sub.OriginalFileName, len(unassigned), len(primary.People))
		}
	}
}

// parseImmichDateTime parses an Immich LocalDateTime string, trying multiple formats.
func parseImmichDateTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised datetime format: %q", s)
}
