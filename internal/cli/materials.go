package cli

import (
	"fmt"

	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func newAttachmentsCommand(j *bool) *cobra.Command {
	g := group("attachments", "Manage song, arrangement, or key files and links")
	var s services.Scope
	var o services.ListOptions
	c := dataCommand("list", "List materials at an explicit scope", 0, func(c *cobra.Command, _ []string) error {
		if e := checkScope(s, false); e != nil {
			return e
		}
		var e error
		o, e = listOptions(c, o)
		if e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if *j && o.All {
			return writeStreamList(c, func(yield func(services.Attachment) error) (services.ListMeta, error) {
				return a.library.WalkAttachments(c.Context(), s, yield)
			})
		}
		r, e := a.library.ListAttachments(c.Context(), s, o)
		if e != nil {
			return e
		}
		return writeList(c, *j, a, r)
	})
	selection(c, &s, true, true)
	listFlags(c, &o)
	g.AddCommand(c)
	var sh services.Scope
	c = dataCommand("show ATTACHMENT", "Inspect one file or link", 1, func(c *cobra.Command, x []string) error {
		if e := checkScope(sh, false); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ShowAttachment(c.Context(), sh, x[0])
		if e != nil {
			return e
		}
		return writeData(c, *j, a, r, nil)
	})
	selection(c, &sh, true, true)
	g.AddCommand(c)
	var cs services.Scope
	var m services.AttachmentCreate
	var item bool
	c = dataCommand("add", "Upload a file or attach a link/content/upload ID", 0, func(c *cobra.Command, _ []string) error {
		if e := checkScope(cs, false); e != nil {
			return e
		}
		if c.Flags().Changed("item-details") {
			m.ImportToItemDetails = &item
		}
		n := 0
		for _, v := range []string{m.FilePath, m.FileUploadID, m.RemoteLink, m.Content} {
			if v != "" {
				n++
			}
		}
		if n != 1 {
			return usageError{fmt.Errorf("provide exactly one of --file, --upload-id, --url, or --content")}
		}
		if e := services.ValidateAttachmentCreate(m); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.AddAttachment(c.Context(), cs, m)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, nil)
	})
	selection(c, &cs, true, true)
	c.Flags().StringVar(&m.FilePath, "file", "", "local regular file to upload")
	c.Flags().StringVar(&m.FileUploadID, "upload-id", "", "reuse a known upload receipt UUID")
	c.Flags().StringVar(&m.RemoteLink, "url", "", "HTTP(S) remote link")
	c.Flags().StringVar(&m.Content, "content", "", "inline file content")
	c.Flags().StringVar(&m.Filename, "filename", "", "destination filename")
	c.Flags().StringVar(&m.AttachmentTypeIDs, "attachment-type-ids", "", "comma-separated file access type IDs")
	c.Flags().StringVar(&m.PageOrder, "page-order", "", "page ordering")
	c.Flags().StringVar(&m.SongPart, "song-part", "", "song section label")
	c.Flags().BoolVar(&item, "item-details", false, "set import to item details state")
	g.AddCommand(c)
	var us services.Scope
	var p services.AttachmentPatch
	b := newBindings()
	c = dataCommand("update ATTACHMENT", "Change supplied file metadata or replace content", 1, func(c *cobra.Command, x []string) error {
		if e := b.apply(c); e != nil {
			return e
		}
		if e := checkScope(us, false); e != nil {
			return e
		}
		if e := services.ValidateAttachmentPatch(p); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.UpdateAttachment(c.Context(), us, x[0], p)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, p)
	})
	selection(c, &us, true, true)
	b.str(c, "filename", "destination filename", &p.Filename)
	b.str(c, "attachment-type-ids", "file access type IDs", &p.AttachmentTypeIDs)
	b.str(c, "page-order", "page ordering", &p.PageOrder)
	b.str(c, "content", "replacement inline content", &p.Content)
	b.str(c, "url", "replacement remote link", &p.RemoteLink)
	b.str(c, "upload-id", "replacement known upload UUID", &p.FileUploadID)
	b.str(c, "song-part", "song section label", &p.SongPart)
	b.boolean(c, "item-details", "set import to item details state", &p.ImportToItemDetails)
	g.AddCommand(c)
	var ds services.Scope
	c = dataCommand("delete ATTACHMENT", "Permanently delete a file attachment or link", 1, func(c *cobra.Command, x []string) error {
		if e := confirm(c); e != nil {
			return e
		}
		if e := checkScope(ds, false); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if e = a.library.DeleteAttachment(c.Context(), ds, x[0]); e != nil {
			return e
		}
		return writeMutationData(c, *j, a, receipt("attachment", x[0]), nil)
	})
	selection(c, &ds, true, true)
	deleteFlag(c)
	g.AddCommand(c)
	return g
}
func newTagsCommand(j *bool) *cobra.Command {
	g := group("tags", "Discover and replace song or arrangement tag assignments")
	var s services.Scope
	var o services.ListOptions
	c := dataCommand("list", "List assigned tags or the available song/arrangement catalog", 0, func(c *cobra.Command, _ []string) error {
		if s.Arrangement != "" && s.Song == "" {
			return usageError{fmt.Errorf("--arrangement requires --song")}
		}
		var e error
		o, e = listOptions(c, o)
		if e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if *j && o.All {
			return writeStreamList(c, func(yield func(services.Tag) error) (services.ListMeta, error) {
				return a.library.WalkTags(c.Context(), s, yield)
			})
		}
		r, e := a.library.ListTags(c.Context(), s, o)
		if e != nil {
			return e
		}
		return writeList(c, *j, a, r)
	})
	selection(c, &s, true, false)
	c.Flags().Lookup("song").Usage = "optional song ID or name; omit to discover catalog"
	listFlags(c, &o)
	g.AddCommand(c)
	var rs services.Scope
	var tags []string
	var clear bool
	c = dataCommand("replace", "Replace the complete tag set on a song or arrangement", 0, func(c *cobra.Command, _ []string) error {
		if e := checkScope(rs, false); e != nil {
			return e
		}
		if clear == c.Flags().Changed("tags") {
			return usageError{fmt.Errorf("provide --tags or --clear, exclusively")}
		}
		if !clear && len(tags) == 0 {
			return usageError{fmt.Errorf("use --clear to remove all tags")}
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ReplaceTags(c.Context(), rs, tags)
		if e != nil {
			return e
		}
		if e := writeList(c, *j, a, r); e != nil {
			return &committedOutputError{Result: r.Data, Cause: e}
		}
		return nil
	})
	selection(c, &rs, true, false)
	c.Flags().StringSliceVar(&tags, "tags", nil, "complete replacement set of tag IDs or exact names")
	c.Flags().BoolVar(&clear, "clear", false, "explicitly remove all assigned tags")
	g.AddCommand(c)
	return g
}
func newCatalogCommand(name string, j *bool) *cobra.Command {
	g := group(name, "Discover library classification metadata")
	var o services.ListOptions
	c := dataCommand("list", "List available classifications", 0, func(c *cobra.Command, _ []string) error {
		var e error
		o, e = listOptions(c, o)
		if e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if name == "tag-groups" {
			if *j && o.All {
				return writeStreamList(c, func(yield func(services.TagGroup) error) (services.ListMeta, error) {
					return a.library.WalkTagGroups(c.Context(), yield)
				})
			}
			r, e := a.library.ListTagGroups(c.Context(), o)
			if e != nil {
				return e
			}
			return writeList(c, *j, a, r)
		}
		if *j && o.All {
			return writeStreamList(c, func(yield func(services.AttachmentType) error) (services.ListMeta, error) {
				return a.library.WalkAttachmentTypes(c.Context(), yield)
			})
		}
		r, e := a.library.ListAttachmentTypes(c.Context(), o)
		if e != nil {
			return e
		}
		return writeList(c, *j, a, r)
	})
	listFlags(c, &o)
	g.AddCommand(c)
	return g
}
