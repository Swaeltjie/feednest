package handlers

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/store"
)

type OPMLHandler struct {
	store *store.Queries
}

func NewOPMLHandler(store *store.Queries) *OPMLHandler {
	return &OPMLHandler{store: store}
}

type opmlFeed struct {
	Title    string
	XMLURL   string
	HTMLURL  string
	Category string
}

type opmlDocument struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Body    opmlBody `xml:"body"`
}

type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

type opmlOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr,omitempty"`
	Type     string        `xml:"type,attr,omitempty"`
	XMLURL   string        `xml:"xmlUrl,attr,omitempty"`
	HTMLURL  string        `xml:"htmlUrl,attr,omitempty"`
	Outlines []opmlOutline `xml:"outline,omitempty"`
}

func parseOPML(r io.Reader) ([]opmlFeed, error) {
	var doc opmlDocument
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, err
	}

	var feeds []opmlFeed
	for _, outline := range doc.Body.Outlines {
		if outline.XMLURL != "" {
			feeds = append(feeds, opmlFeed{
				Title:   outline.Text,
				XMLURL:  outline.XMLURL,
				HTMLURL: outline.HTMLURL,
			})
		} else {
			for _, child := range outline.Outlines {
				if child.XMLURL != "" {
					feeds = append(feeds, opmlFeed{
						Title:    child.Text,
						XMLURL:   child.XMLURL,
						HTMLURL:  child.HTMLURL,
						Category: outline.Text,
					})
				}
			}
		}
	}
	return feeds, nil
}

func generateOPML(feeds []opmlFeed) (string, error) {
	categories := make(map[string][]opmlOutline)
	var uncategorized []opmlOutline

	for _, f := range feeds {
		outline := opmlOutline{
			Text:    f.Title,
			Title:   f.Title,
			Type:    "rss",
			XMLURL:  f.XMLURL,
			HTMLURL: f.HTMLURL,
		}
		if f.Category != "" {
			categories[f.Category] = append(categories[f.Category], outline)
		} else {
			uncategorized = append(uncategorized, outline)
		}
	}

	var body []opmlOutline
	for cat, outlines := range categories {
		body = append(body, opmlOutline{
			Text:     cat,
			Title:    cat,
			Outlines: outlines,
		})
	}
	body = append(body, uncategorized...)

	doc := opmlDocument{
		Version: "2.0",
		Body:    opmlBody{Outlines: body},
	}

	data, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return xml.Header + string(data), nil
}

func (h *OPMLHandler) Import(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)

	// Limit upload size to 5MB
	r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024)

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file upload required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	feeds, err := parseOPML(file)
	if err != nil {
		http.Error(w, `{"error":"invalid OPML file"}`, http.StatusBadRequest)
		return
	}

	imported := 0
	for _, f := range feeds {
		var categoryID *int64
		if f.Category != "" {
			cat, err := h.store.CreateCategory(userID, f.Category, 0)
			if err != nil {
				cats, _ := h.store.ListCategories(userID)
				for _, c := range cats {
					if c.Name == f.Category {
						categoryID = &c.ID
						break
					}
				}
			} else {
				categoryID = &cat.ID
			}
		}

		_, err := h.store.CreateFeed(userID, f.XMLURL, f.Title, f.HTMLURL, "", categoryID)
		if err == nil {
			imported++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"imported":%d,"total":%d}`, imported, len(feeds))
}

func (h *OPMLHandler) Export(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)

	feedList, err := h.store.ListFeeds(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to list feeds"}`, http.StatusInternalServerError)
		return
	}

	cats, _ := h.store.ListCategories(userID)
	catMap := make(map[int64]string)
	for _, c := range cats {
		catMap[c.ID] = c.Name
	}

	var opmlFeeds []opmlFeed
	for _, f := range feedList {
		of := opmlFeed{
			Title:   f.Title,
			XMLURL:  f.URL,
			HTMLURL: f.SiteURL,
		}
		if f.CategoryID != nil {
			of.Category = catMap[*f.CategoryID]
		}
		opmlFeeds = append(opmlFeeds, of)
	}

	output, err := generateOPML(opmlFeeds)
	if err != nil {
		http.Error(w, `{"error":"failed to generate OPML"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=feednest-export.opml")
	w.Write([]byte(output))
}
