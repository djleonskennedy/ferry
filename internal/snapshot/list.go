package snapshot

import (
	"time"

	"github.com/djleonskennedy/ferry/internal/manifest"
)

type ListEntry struct {
	Version   int
	CreatedAt time.Time
	KeyID     string
	Message   string
	FileCount int
	IsLatest  bool
}

// List returns all snapshots for a project, oldest first.
func List(project string) ([]ListEntry, error) {
	versions, err := listVersions(project)
	if err != nil {
		return nil, err
	}
	latest, _ := ReadLatest(project)
	out := make([]ListEntry, 0, len(versions))
	for _, v := range versions {
		m, err := manifest.Read(ManifestPath(project, v))
		if err != nil {
			return nil, err
		}
		out = append(out, ListEntry{
			Version:   v,
			CreatedAt: m.CreatedAt,
			KeyID:     m.KeyID,
			Message:   m.Message,
			FileCount: len(m.Files),
			IsLatest:  v == latest,
		})
	}
	return out, nil
}
