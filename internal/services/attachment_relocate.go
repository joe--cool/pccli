package services

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

type AttachmentRelocation struct {
	Source      Attachment        `json:"source"`
	From        Scope             `json:"from"`
	To          Scope             `json:"to"`
	Replacement *AttachmentResult `json:"replacement,omitempty"`
	Outcome     string            `json:"outcome"`
	KeyRemoved  bool              `json:"key_removed"`
}

type RelocationOptions struct {
	From, To                      Scope
	SourceID, File, ReplacementID string
	Apply, PruneEmptyKey          bool
	Checkpoint                    func(AttachmentRelocation) error
}

func (l *Library) RelocateAttachment(ctx context.Context, opt RelocationOptions) (AttachmentRelocation, error) {
	out := AttachmentRelocation{From: opt.From, To: opt.To, Outcome: "preview"}
	explicit := func(s string) bool {
		id, _, _, err := ParseSelector(s)
		return err == nil && id != ""
	}
	for _, scope := range []Scope{opt.From, opt.To} {
		if !explicit(scope.Song) || scope.Arrangement != "" && !explicit(scope.Arrangement) ||
			scope.Key != "" && (!explicit(scope.Key) || scope.Arrangement == "") {
			return out, fail("invalid_input", "source and destination require explicit IDs for every selected parent")
		}
	}
	if !explicit(opt.SourceID) || opt.ReplacementID != "" && (!opt.Apply || !explicit(opt.ReplacementID)) ||
		opt.PruneEmptyKey && opt.From.Key == "" || opt.Apply && opt.Checkpoint == nil {
		return out, fail("invalid_input", "use explicit IDs, a source key for pruning, and a journal for apply")
	}
	source, sourcePath, err := l.attachment(ctx, opt.From, opt.SourceID, false)
	if err != nil {
		return out, err
	}
	_, destPath, err := l.attachmentScope(ctx, opt.To, false)
	if err != nil {
		return out, err
	}
	srcParent := strings.TrimSuffix(sourcePath, "/"+source.id)
	if srcParent == destPath {
		return out, fail("invalid_input", "source and destination must be different")
	}
	if err := source.model(&out.Source); err != nil {
		return out, err
	}
	a := out.Source
	if a.Filename == "" || a.PCOType != "AttachmentS3" || !a.Downloadable || a.FileSize <= 0 ||
		a.ContentType != "application/pdf" || a.LicensesPurchased != 0 || a.LicensesRemaining != 0 {
		return out, fail("unsupported_attachment", "relocation supports downloadable, unlicensed uploaded PDFs")
	}
	fingerprint, err := hex.DecodeString(a.RemoteLink)
	if err != nil || len(fingerprint) != md5.Size {
		return out, fail("unsupported_attachment", "uploaded PDF has no usable content fingerprint")
	}
	create := AttachmentCreate{Filename: a.Filename, FilePath: opt.File}
	// Include blank pages so a customized order cannot appear empty.
	var detail planningcenter.Single[json.RawMessage]
	if err := l.api.http.Get(ctx, sourcePath, url.Values{"blank_pages": {"true"}}, &detail); err != nil {
		return out, err
	}
	order, err := decodeRecord(detail.Data, "Attachment")
	if err != nil {
		return out, err
	}
	if order.id != source.id {
		return out, changed("source attachment")
	}
	switch strings.TrimSpace(string(order.attrs["page_order"])) {
	case "", "null", `""`, "[]":
	default:
		return out, fail("unsupported_attachment", "relocation supports PDFs with the default page order")
	}
	if raw := source.attrs["song_part"]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &create.SongPart); err != nil {
			return out, fail("invalid_response", "invalid source song part")
		}
	}
	if raw := source.attrs["import_to_item_details"]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &create.ImportToItemDetails); err != nil {
			return out, fail("invalid_response", "invalid source item-details setting")
		}
	}
	types, err := attachmentTypeIDs(source)
	if err != nil {
		return out, err
	}
	create.AttachmentTypeIDs = strings.Join(types, ",")
	if opt.File != "" || opt.Apply && opt.ReplacementID == "" {
		if err := ValidateAttachmentCreate(create); err != nil {
			return out, err
		}
		info, err := os.Stat(opt.File)
		if err != nil {
			return out, err
		}
		if info.Size() != int64(a.FileSize) {
			return out, fail("invalid_input", "replacement file size differs from the source")
		}
		file, err := os.Open(opt.File)
		if err != nil {
			return out, err
		}
		hash := md5.New()
		_, err = io.Copy(hash, file)
		_ = file.Close()
		if err != nil {
			return out, err
		}
		if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), a.RemoteLink) {
			return out, fail("invalid_input", "local PDF checksum differs from the uploaded source")
		}
	}
	if opt.PruneEmptyKey {
		if err := l.verifyUnscheduled(ctx, opt.From); err != nil {
			return out, err
		}
	}
	rows, err := l.api.collect(ctx, destPath, "Attachment", nil)
	if err != nil {
		return out, err
	}
	for _, row := range rows {
		var f Attachment
		if err := row.model(&f); err != nil {
			return out, err
		}
		matchID := opt.ReplacementID == "id:"+f.ID || opt.ReplacementID == f.ID
		if strings.EqualFold(a.Filename, f.Filename) && !matchID {
			return out, fail("destination_conflict", "another attachment already has the destination filename")
		}
	}
	if !opt.Apply {
		return out, nil
	}
	out.Outcome = "pending"
	if err := opt.Checkpoint(out); err != nil {
		return out, err
	}
	if opt.ReplacementID == "" {
		added, err := l.AddAttachment(ctx, opt.To, create)
		out.Replacement = &added
		if err != nil {
			out.Outcome = added.Outcome
			if checkpointErr := opt.Checkpoint(out); checkpointErr != nil {
				return out, checkpointErr
			}
			return out, err
		}
		opt.ReplacementID = "id:" + added.Attachment.ID
	} else {
		out.Replacement = &AttachmentResult{Outcome: "reconciling", Scope: opt.To}
	}
	out.Outcome = "replacement_pending_verification"
	if err := opt.Checkpoint(out); err != nil {
		return out, err
	}
	replacement, err := l.verifyRelocationReplacement(ctx, opt, a, types)
	if err != nil {
		return out, err
	}
	out.Replacement.Attachment = &replacement
	out.Replacement.Outcome = "verified"
	out.Outcome = "replacement_verified"
	if err := opt.Checkpoint(out); err != nil {
		return out, err
	}
	current, _, err := l.attachment(ctx, opt.From, opt.SourceID, false)
	if err != nil {
		return out, err
	}
	before, err := recordHash(source)
	if err != nil {
		return out, err
	}
	after, err := recordHash(current)
	if err != nil {
		return out, err
	}
	if before != after {
		return out, changed("source attachment")
	}
	if err := l.DeleteAttachment(ctx, opt.From, opt.SourceID); err != nil {
		return out, err
	}
	remaining, err := l.api.collect(ctx, srcParent, "Attachment", nil)
	if err != nil {
		return out, err
	}
	for _, row := range remaining {
		if row.id == source.id {
			return out, fail("verification_failed", "source attachment is still present")
		}
	}
	out.Outcome = "source_retired"
	if err := opt.Checkpoint(out); err != nil {
		return out, err
	}
	if opt.PruneEmptyKey && len(remaining) == 0 {
		if err := l.verifyUnscheduled(ctx, opt.From); err != nil {
			return out, err
		}
		current, err := l.api.collect(ctx, srcParent, "Attachment", nil)
		if err != nil {
			return out, err
		}
		if len(current) != 0 {
			return out, changed("source key attachments")
		}
		if err := l.DeleteKey(ctx, opt.From, opt.From.Key); err != nil {
			return out, err
		}
		keys, err := l.ListKeys(ctx, opt.From, ListOptions{All: true})
		if err != nil {
			return out, err
		}
		for _, key := range keys.Data {
			if opt.From.Key == key.ID || opt.From.Key == "id:"+key.ID {
				return out, fail("verification_failed", "source key is still present")
			}
		}
		out.KeyRemoved = true
	}
	out.Outcome = "verified"
	return out, opt.Checkpoint(out)
}

func (l *Library) verifyUnscheduled(ctx context.Context, scope Scope) error {
	schedules, err := l.ListSongSchedules(ctx, Scope{Song: scope.Song, Visibility: scope.Visibility}, ListOptions{All: true})
	if err != nil {
		return err
	}
	if len(schedules.Data) > 0 {
		return fail("scheduled_song", "cannot automatically prune a key from a song with scheduled uses")
	}
	return nil
}

func attachmentTypeIDs(r record) ([]string, error) {
	var relationships struct {
		Types struct {
			Data []struct{ ID, Type string }
		} `json:"attachment_types"`
	}
	if len(r.rel) > 0 && json.Unmarshal(r.rel, &relationships) != nil {
		return nil, fail("invalid_response", "invalid attachment type relationships")
	}
	ids := []string{}
	for _, row := range relationships.Types.Data {
		if row.Type != "AttachmentType" || !validID(row.ID) {
			return nil, fail("invalid_response", "invalid attachment type")
		}
		ids = append(ids, row.ID)
	}
	slices.Sort(ids)
	return ids, nil
}

func (l *Library) verifyRelocationReplacement(ctx context.Context, opt RelocationOptions, source Attachment, types []string) (Attachment, error) {
	var attachment Attachment
	r, _, err := l.attachment(ctx, opt.To, opt.ReplacementID, false)
	if err != nil {
		return attachment, err
	}
	if err := r.model(&attachment); err != nil {
		return attachment, err
	}
	gotTypes, err := attachmentTypeIDs(r)
	if err != nil {
		return attachment, err
	}
	if attachment.ID == source.ID || attachment.Filename != source.Filename ||
		attachment.ContentType != source.ContentType || attachment.FileSize != source.FileSize ||
		attachment.PCOType != source.PCOType || !strings.EqualFold(attachment.RemoteLink, source.RemoteLink) ||
		!attachment.Downloadable || !slices.Equal(gotTypes, types) {
		return attachment, fail("verification_failed", "replacement did not match the source file, content fingerprint, or access tags")
	}
	return attachment, nil
}
