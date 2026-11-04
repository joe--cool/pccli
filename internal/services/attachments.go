package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

func attachmentPath(p resolvedParents, key string) string {
	path := songPath(p.song.id)
	if p.arr.id != "" {
		path = arrangementPath(p.song.id, p.arr.id)
	}
	if key != "" {
		path = keyPath(p.song.id, p.arr.id, key)
	}
	return path + "/attachments"
}
func (l *Library) attachmentScope(ctx context.Context, s Scope, write bool) (resolvedParents, string, error) {
	if s.Key != "" && s.Arrangement == "" {
		return resolvedParents{}, "", fail("invalid_input", "key requires arrangement")
	}
	p, e := l.parents(ctx, s, s.Arrangement != "", write)
	if e != nil {
		return p, "", e
	}
	key := ""
	if s.Key != "" {
		r, e := l.key(ctx, p, s.Key, write)
		if e != nil {
			return p, "", e
		}
		key = r.id
	}
	return p, attachmentPath(p, key), nil
}
func (l *Library) ListAttachments(ctx context.Context, s Scope, o ListOptions) (ListResult[Attachment], error) {
	if _, e := bound([]Attachment{}, o); e != nil {
		return ListResult[Attachment]{}, e
	}
	_, path, e := l.attachmentScope(ctx, s, false)
	if e != nil {
		return ListResult[Attachment]{}, e
	}
	rows, e := l.api.collect(ctx, path, "Attachment", nil)
	if e != nil {
		return ListResult[Attachment]{}, e
	}
	out := []Attachment{}
	for _, r := range rows {
		var x Attachment
		if e = r.model(&x); e != nil {
			return ListResult[Attachment]{}, e
		}
		out = append(out, x)
	}
	return bound(out, o)
}

// ValidateAttachmentCreate checks supplied values and local file access without
// resolving a destination or starting an upload.
func ValidateAttachmentCreate(m AttachmentCreate) error {
	sources := 0
	for _, x := range []string{m.FilePath, m.FileUploadID, m.RemoteLink, m.Content} {
		if x != "" {
			sources++
		}
	}
	if sources != 1 {
		return fail("invalid_input", "provide exactly one of file, upload ID, remote link, or content")
	}
	if m.RemoteLink != "" {
		u, e := url.Parse(m.RemoteLink)
		if e != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
			return fail("invalid_input", "remote link must be an absolute HTTP or HTTPS URL without credentials")
		}
	}

	if m.FileUploadID != "" && !validReceiptID(m.FileUploadID) {
		return fail("invalid_input", "upload ID cannot contain whitespace or control characters")
	}
	if m.FilePath != "" {
		file, err := os.Open(m.FilePath)
		if err != nil {
			return &Error{Code: "invalid_input", Message: "cannot open upload file: " + err.Error(), Cause: err}
		}
		info, statErr := file.Stat()
		closeErr := file.Close()
		if statErr != nil {
			return &Error{Code: "invalid_input", Message: "cannot inspect upload file", Cause: statErr}
		}
		if !info.Mode().IsRegular() {
			return fail("invalid_input", "upload requires a regular file")
		}
		if closeErr != nil {
			return &Error{Code: "invalid_input", Message: "cannot close upload file", Cause: closeErr}
		}
	}
	return nil
}
func (l *Library) AddAttachment(ctx context.Context, s Scope, m AttachmentCreate) (AttachmentResult, error) {
	result := AttachmentResult{Outcome: "not_started", Scope: s}
	if err := ValidateAttachmentCreate(m); err != nil {
		return result, err
	}
	parents, path, e := l.attachmentScope(ctx, s, true)
	if e != nil {
		return result, e
	}
	result.Scope.Song = parents.song.id
	result.Scope.Arrangement = parents.arr.id
	result.Scope.Visibility, _ = visible(s.Visibility)
	if s.Key != "" {
		parts := strings.Split(path, "/")
		result.Scope.Key = parts[len(parts)-2]
	}
	if m.FileUploadID != "" {
		result.Upload = &UploadedFile{ID: m.FileUploadID, Name: m.Filename}
	}
	attrs := map[string]any{}
	if m.FilePath != "" {
		var response planningcenter.Collection[json.RawMessage]
		if e = l.api.http.UploadFile(ctx, m.FilePath, &response); e != nil {
			return result, e
		}
		if len(response.Data) != 1 || !validReceiptID(response.Data[0].ID) || response.Data[0].Type != "File" {
			return result, fail("invalid_response", "upload must return exactly one file receipt")
		}
		upload := UploadedFile{ID: response.Data[0].ID}
		result.Upload = &upload
		if e = json.Unmarshal(response.Data[0].Attributes, &upload); e != nil {
			result.Outcome = "not_attached"
			return result, &Error{Code: "partial_success", Message: "upload returned an invalid receipt; preserve its ID", Recovery: &result, Cause: e}
		}
		upload.ID = response.Data[0].ID
		m.FileUploadID = upload.ID
		if m.Filename == "" {
			m.Filename = upload.Name
			if m.Filename == "" {
				m.Filename = filepath.Base(m.FilePath)
			}
		}
	}
	for k, v := range map[string]string{"file_upload_identifier": m.FileUploadID, "filename": m.Filename, "remote_link": m.RemoteLink, "content": m.Content, "attachment_type_ids": m.AttachmentTypeIDs, "page_order": m.PageOrder, "song_part": m.SongPart} {
		if v != "" {
			attrs[k] = v
		}
	}
	if m.ImportToItemDetails != nil {
		attrs["import_to_item_details"] = *m.ImportToItemDetails
	}
	// Uploads can take time. Revalidate scoped selection before attaching the known receipt.
	if result.Upload != nil {
		_, refreshedPath, checkErr := l.attachmentScope(ctx, s, true)
		if checkErr != nil || refreshedPath != path {
			result.Outcome = "not_attached"
			return result, &Error{Code: "partial_success", Message: "upload completed but destination could not be revalidated; preserve receipt", Recovery: &result, Cause: checkErr}
		}
	}
	r, e := l.api.mutate(ctx, "POST", path, "Attachment", attrs)
	if e != nil {
		result.Outcome = "unknown"
		var apiErr planningcenter.APIError
		if errors.As(e, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			result.Outcome = "rejected"
		}
		code := "outcome_unknown"
		if result.Outcome == "rejected" {
			code = "api_error"
			if result.Upload != nil || m.FileUploadID != "" {
				code = "partial_success"
			}
		}
		return result, &Error{Code: code, Message: "attachment creation " + result.Outcome + "; preserve the upload receipt and inspect destination before retrying", Recovery: &result, Cause: e}
	}
	var attachment Attachment
	if e = r.model(&attachment); e != nil {
		result.Outcome = "unknown"
		return result, &Error{Code: "outcome_unknown", Message: "attachment submitted but response invalid", Recovery: &result, Cause: e}
	}
	result.Attachment = &attachment
	result.Outcome = "created"
	return result, nil
}

func validReceiptID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
