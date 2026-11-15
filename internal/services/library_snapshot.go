package services

import (
	"context"
	"sync"
)

const LibrarySnapshotSchema = "pccli/library-v1"

type LibraryInclude struct {
	Arrangements bool `json:"arrangements"`
	Keys         bool `json:"keys"`
	Attachments  bool `json:"attachments"`
	Tags         bool `json:"tags"`
}

type LibrarySnapshot struct {
	Schema          string         `json:"schema"`
	Complete        bool           `json:"complete"`
	Include         LibraryInclude `json:"include"`
	Visibility      Visibility     `json:"visibility"`
	IncludeArchived bool           `json:"include_archived"`
	Songs           []LibrarySong  `json:"songs"`
}

type LibrarySong struct {
	Song         Song                 `json:"song"`
	Arrangements []LibraryArrangement `json:"arrangements"`
	Attachments  []Attachment         `json:"attachments"`
	Tags         []Tag                `json:"tags"`
}

type LibraryArrangement struct {
	Arrangement Arrangement  `json:"arrangement"`
	Keys        []LibraryKey `json:"keys"`
	Attachments []Attachment `json:"attachments"`
	Tags        []Tag        `json:"tags"`
}

type LibraryKey struct {
	Key         Key          `json:"key"`
	Attachments []Attachment `json:"attachments"`
}

func (l *Library) attachmentRows(ctx context.Context, path string) ([]Attachment, error) {
	rows := []Attachment{}
	_, err := walkModels(ctx, l, path+"/attachments", "Attachment", nil, nil,
		func(a Attachment) error {
			rows = append(rows, a)
			return nil
		})
	return rows, err
}

func (l *Library) libraryArrangement(ctx context.Context, songID string, arrangement Arrangement, include LibraryInclude, catalog []Tag) (LibraryArrangement, error) {
	path := arrangementPath(songID, arrangement.ID)
	out := LibraryArrangement{Arrangement: arrangement}
	var err error
	if include.Attachments {
		out.Attachments, err = l.attachmentRows(ctx, path)
		if err != nil {
			return out, err
		}
	}
	if include.Keys {
		out.Keys = []LibraryKey{}
		_, err = walkModels(ctx, l, path+"/keys", "Key", nil, func(r record) (bool, error) {
			return true, r.scope("arrangement", "Arrangement", arrangement.ID)
		}, func(key Key) error {
			entry := LibraryKey{Key: key}
			if include.Attachments {
				var err error
				entry.Attachments, err = l.attachmentRows(ctx, keyPath(songID, arrangement.ID, key.ID))
				if err != nil {
					return err
				}
			}
			out.Keys = append(out.Keys, entry)
			return nil
		})
		if err != nil {
			return out, err
		}
	}
	if include.Tags {
		tags, err := l.tagsAt(ctx, path, "arrangement", catalog, ListOptions{All: true})
		if err != nil {
			return out, err
		}
		out.Tags = tags.Data
	}
	return out, nil
}

func (l *Library) librarySong(ctx context.Context, song Song, include LibraryInclude, archived bool, catalog []Tag) (LibrarySong, error) {
	out := LibrarySong{Song: song}
	path := songPath(song.ID)
	var err error
	if include.Attachments {
		out.Attachments, err = l.attachmentRows(ctx, path)
		if err != nil {
			return out, err
		}
	}
	if include.Arrangements {
		out.Arrangements = []LibraryArrangement{}
		_, err = walkModels(ctx, l, arrangementsPath(song.ID, archived), "Arrangement", nil,
			func(r record) (bool, error) {
				return !isArchived(r) || archived, r.scope("song", "Song", song.ID)
			}, func(a Arrangement) error {
				entry, err := l.libraryArrangement(ctx, song.ID, a, include, catalog)
				if err == nil {
					out.Arrangements = append(out.Arrangements, entry)
				}
				return err
			})
		if err != nil {
			return out, err
		}
	}
	if include.Tags {
		tags, err := l.tagsAt(ctx, path, "song", catalog, ListOptions{All: true})
		if err != nil {
			return out, err
		}
		out.Tags = tags.Data
	}
	return out, nil
}

// ExportLibrary traverses selected scopes once per resource and returns only a
// fully paginated snapshot. Progress callbacks are serialized.
func (l *Library) ExportLibrary(ctx context.Context, visibility Visibility, archived bool, include LibraryInclude, workers int, progress func(int, int)) (LibrarySnapshot, error) {
	v, err := visible(visibility)
	if err != nil {
		return LibrarySnapshot{}, err
	}
	if workers < 1 || workers > 4 || include.Keys && !include.Arrangements || include.Attachments && (!include.Keys || !include.Arrangements) {
		return LibrarySnapshot{}, fail("invalid_input", "workers must be 1–4; keys require arrangements, and attachments require arrangements and keys")
	}
	var songs []Song
	_, err = l.WalkSongs(ctx, SongListOptions{Visibility: v, Order: "title"}, func(s Song) error {
		songs = append(songs, s)
		return nil
	})
	if err != nil {
		return LibrarySnapshot{}, err
	}
	var catalog []Tag
	if include.Tags {
		catalog, _, err = l.tagCatalog(ctx)
		if err != nil {
			return LibrarySnapshot{}, err
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	out := LibrarySnapshot{Schema: LibrarySnapshotSchema, Include: include, Visibility: v,
		IncludeArchived: archived, Songs: make([]LibrarySong, len(songs))}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	done := 0
	for worker := range min(workers, len(songs)) {
		wg.Go(func() {
			for index := worker; index < len(songs) && ctx.Err() == nil; index += workers {
				entry, err := l.librarySong(ctx, songs[index], include, archived, catalog)
				mu.Lock()
				if err != nil {
					if firstErr == nil {
						firstErr = err
						cancel()
					}
				} else {
					out.Songs[index] = entry
					done++
					if progress != nil {
						progress(done, len(songs))
					}
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if firstErr != nil {
		return LibrarySnapshot{}, firstErr
	}
	if err := ctx.Err(); err != nil {
		return LibrarySnapshot{}, err
	}
	out.Complete = true
	return out, nil
}

func (l *Library) ShowArrangementFiles(ctx context.Context, s Scope, target string) (LibraryArrangement, error) {
	s.Arrangement = target
	p, err := l.parents(ctx, s, true, false)
	if err != nil {
		return LibraryArrangement{}, err
	}
	var arrangement Arrangement
	if err := p.arr.inPath(arrangementPath(p.song.id, p.arr.id)).model(&arrangement); err != nil {
		return LibraryArrangement{}, err
	}
	return l.libraryArrangement(ctx, p.song.id, arrangement, LibraryInclude{Keys: true, Attachments: true}, nil)
}
