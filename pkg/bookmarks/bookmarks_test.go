package bookmarks

import (
	"testing"
)

func TestSaveLoadList(t *testing.T) {
	dir := t.TempDir()

	if err := Save(dir, Bookmark{Name: "prod", URL: "mongodb://prod.internal:27017"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := Save(dir, Bookmark{Name: "dev", URL: "mongodb://localhost:27017"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	b, err := Load(dir, "prod")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if b.URL != "mongodb://prod.internal:27017" {
		t.Errorf("URL = %q, want mongodb://prod.internal:27017", b.URL)
	}

	list, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 || list[0].Name != "dev" || list[1].Name != "prod" {
		t.Errorf("List = %+v, want [dev, prod] sorted", list)
	}
}

func TestListMissingDirectory(t *testing.T) {
	list, err := List("/nonexistent/path/for/sure")
	if err != nil {
		t.Fatalf("List on missing dir: %v", err)
	}
	if list != nil {
		t.Errorf("expected nil list for missing directory, got %+v", list)
	}
}

func TestLoadMissingBookmark(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir, "does-not-exist"); err == nil {
		t.Error("expected an error loading a nonexistent bookmark")
	}
}
