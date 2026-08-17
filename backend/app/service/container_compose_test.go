package service

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
)

func TestValidateComposeName(t *testing.T) {
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"blog", false},
		{"Blog_1.test", false},
		{"", true},
		{"-bad", true},
		{"has space", true},
		{"../etc", true},
		{"a/b", true},
	}
	for _, tc := range cases {
		err := validateComposeName(tc.name)
		if tc.wantErr && err == nil {
			t.Fatalf("name %q: want error", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("name %q: unexpected %v", tc.name, err)
		}
	}
}

func TestValidateComposePath(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(ok, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := validateComposePath(ok); err != nil {
		t.Fatalf("existing absolute path: %v", err)
	}
	if _, err := validateComposePath("relative.yml"); err == nil {
		t.Fatal("relative path should fail")
	}
	if _, err := validateComposePath(filepath.Join(dir, "missing.yml")); err == nil {
		t.Fatal("missing file should fail")
	}
	sneaky := filepath.Join(dir, "sub", "..", "docker-compose.yml")
	cleaned, err := validateComposePath(sneaky)
	if err != nil {
		t.Fatalf("cleanable path: %v", err)
	}
	if cleaned != ok {
		t.Fatalf("cleaned = %q, want %q", cleaned, ok)
	}
}

func TestShouldRemoveComposeDir(t *testing.T) {
	root := "/var/lib/xpanel/compose"
	created := filepath.Join(root, "blog", "docker-compose.yml")
	if !shouldRemoveComposeDir("created", created, root) {
		t.Fatal("created project under root should remove dir")
	}
	if shouldRemoveComposeDir("attached", created, root) {
		t.Fatal("attached must never remove dir")
	}
	if shouldRemoveComposeDir("created", "/opt/blog/docker-compose.yml", root) {
		t.Fatal("created but outside root must not remove dir")
	}
	if shouldRemoveComposeDir("created", root+"-other/blog/docker-compose.yml", root) {
		t.Fatal("prefix sibling must not remove dir")
	}
}

func TestParseComposeLSArray(t *testing.T) {
	raw := `[{"Name":"blog","Status":"running(2)","ConfigFiles":"/opt/blog/docker-compose.yml"}]`
	items, err := parseComposeLS(raw)
	if err != nil || len(items) != 1 || items[0].Name != "blog" ||
		items[0].Status != "running(2)" || items[0].ConfigFiles != "/opt/blog/docker-compose.yml" {
		t.Fatalf("array parse = %#v, err=%v", items, err)
	}
}

func TestParseComposeLSNDJSON(t *testing.T) {
	raw := `{"Name":"a","Status":"running(1)","ConfigFiles":"/a/compose.yml"}
{"Name":"b","Status":"exited(1)","ConfigFiles":"/b/compose.yml"}`
	items, err := parseComposeLS(raw)
	if err != nil || len(items) != 2 || items[0].Name != "a" || items[1].Name != "b" {
		t.Fatalf("ndjson parse = %#v, err=%v", items, err)
	}
}

func TestParseComposeLSEmpty(t *testing.T) {
	items, err := parseComposeLS("[]")
	if err != nil || len(items) != 0 {
		t.Fatalf("empty = %#v, err=%v", items, err)
	}
	items, err = parseComposeLS(" \n")
	if err != nil || len(items) != 0 {
		t.Fatalf("blank = %#v, err=%v", items, err)
	}
}

func TestFirstConfigFile(t *testing.T) {
	got := firstConfigFile(" /a.yml, /b.yml ")
	if got != "/a.yml" {
		t.Fatalf("firstConfigFile = %q, want %q", got, "/a.yml")
	}
}

type memoryComposeRepo struct {
	seq          uint
	items        map[uint]model.ComposeProject
	getByNameErr error
}

func newMemoryComposeRepo() *memoryComposeRepo {
	return &memoryComposeRepo{items: map[uint]model.ComposeProject{}}
}

func (r *memoryComposeRepo) Create(item *model.ComposeProject) error {
	for _, existing := range r.items {
		if existing.Name == item.Name {
			return fmt.Errorf("duplicate name")
		}
	}
	r.seq++
	item.ID = r.seq
	r.items[item.ID] = *item
	return nil
}

func (r *memoryComposeRepo) Get(id uint) (*model.ComposeProject, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	copy := item
	return &copy, nil
}

func (r *memoryComposeRepo) GetByName(name string) (*model.ComposeProject, error) {
	if r.getByNameErr != nil {
		return nil, r.getByNameErr
	}
	for _, item := range r.items {
		if item.Name == name {
			copy := item
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (r *memoryComposeRepo) List() ([]model.ComposeProject, error) {
	out := make([]model.ComposeProject, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out, nil
}

func (r *memoryComposeRepo) Delete(id uint) error {
	delete(r.items, id)
	return nil
}

func testComposeService(repo *memoryComposeRepo, run func(name, path string, args ...string) (string, error), list func() ([]composeLSItem, error), root string) *ComposeService {
	if run == nil {
		run = func(name, path string, args ...string) (string, error) {
			return "", nil
		}
	}
	if list == nil {
		list = func() ([]composeLSItem, error) { return nil, nil }
	}
	return &ComposeService{
		repo: repo,
		run:  run,
		list: list,
		root: func() string { return root },
	}
}

func TestCreateComposeWritesPanelFileAndUps(t *testing.T) {
	root := t.TempDir()
	repo := newMemoryComposeRepo()
	var calls []composeRunCall
	svc := testComposeService(repo, func(name, path string, args ...string) (string, error) {
		calls = append(calls, composeRunCall{name: name, path: path, args: append([]string{}, args...)})
		return "", nil
	}, nil, root)

	content := "services:\n  web:\n    image: nginx\n"
	if err := svc.CreateCompose(dto.ComposeCreate{Name: "blog", Content: content}); err != nil {
		t.Fatal(err)
	}

	wantPath := filepath.Join(root, "blog", "docker-compose.yml")
	got, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("panel file missing: %v", err)
	}
	if string(got) != content {
		t.Fatalf("file content = %q, want %q", got, content)
	}

	row, err := repo.GetByName("blog")
	if err != nil {
		t.Fatal(err)
	}
	if row.Source != "created" || row.Path != wantPath {
		t.Fatalf("row = %#v, want source=created path=%s", row, wantPath)
	}
	if len(calls) != 1 || calls[0].name != "blog" || calls[0].path != wantPath || !reflect.DeepEqual(calls[0].args, []string{"up", "-d"}) {
		t.Fatalf("run calls = %#v, want up -d on %s", calls, wantPath)
	}
}

func TestCreateComposeLocksDuringUp(t *testing.T) {
	root := t.TempDir()
	repo := newMemoryComposeRepo()
	started := make(chan struct{})
	release := make(chan struct{})
	var startOnce sync.Once
	svc := testComposeService(repo, func(name, path string, args ...string) (string, error) {
		startOnce.Do(func() { close(started) })
		<-release
		return "", nil
	}, nil, root)

	done := make(chan error, 1)
	go func() {
		done <- svc.CreateCompose(dto.ComposeCreate{Name: "locked", Content: "services: {}\n"})
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("create up did not start")
	}

	// Row must already exist while first up is in progress.
	row, err := repo.GetByName("locked")
	if err != nil {
		t.Fatalf("row should exist after persist: %v", err)
	}

	// Concurrent operate must see busy while create holds the lock.
	err = svc.OperateCompose(dto.ComposeOperate{ID: row.ID, Operation: "stop"})
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("operate during create up = %v, want busy", err)
	}

	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("create: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("create did not finish")
	}
}

func TestCreateComposeAttachDoesNotWriteFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(t.TempDir(), "existing-compose.yml")
	original := []byte("services:\n  db:\n    image: postgres\n")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	called := false
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		called = true
		return "", nil
	}, nil, root)

	if err := svc.CreateCompose(dto.ComposeCreate{Name: "attached-app", Path: path}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("attach create must not call runner")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, original) {
		t.Fatalf("file changed: %q", got)
	}
	row, err := repo.GetByName("attached-app")
	if err != nil {
		t.Fatal(err)
	}
	if row.Source != "attached" || row.Path != path {
		t.Fatalf("row = %#v", row)
	}
}

func TestCreateComposeRejectsContentAndPath(t *testing.T) {
	repo := newMemoryComposeRepo()
	svc := testComposeService(repo, func(name, path string, args ...string) (string, error) {
		t.Fatal("runner must not be called")
		return "", nil
	}, nil, t.TempDir())

	err := svc.CreateCompose(dto.ComposeCreate{
		Name:    "both",
		Path:    "/tmp/docker-compose.yml",
		Content: "services: {}\n",
	})
	if err == nil {
		t.Fatal("expected error when both content and path are set")
	}
	if items, _ := repo.List(); len(items) != 0 {
		t.Fatalf("repo should stay empty, got %#v", items)
	}
}

func TestCreateComposeRejectsNeither(t *testing.T) {
	repo := newMemoryComposeRepo()
	svc := testComposeService(repo, func(name, path string, args ...string) (string, error) {
		t.Fatal("runner must not be called")
		return "", nil
	}, nil, t.TempDir())

	err := svc.CreateCompose(dto.ComposeCreate{Name: "empty"})
	if err == nil {
		t.Fatal("expected error when both content and path are empty")
	}
	if items, _ := repo.List(); len(items) != 0 {
		t.Fatalf("repo should stay empty, got %#v", items)
	}
}

func TestOperateUpdatePullsThenUps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	item := &model.ComposeProject{Name: "blog", Path: path, Source: "created"}
	if err := repo.Create(item); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		calls = append(calls, append([]string{}, args...))
		return "", nil
	}, nil, t.TempDir())

	if err := svc.OperateCompose(dto.ComposeOperate{ID: item.ID, Operation: "update"}); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"pull"}, {"up", "-d"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestOperateUpdateStopsIfPullFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	item := &model.ComposeProject{Name: "blog", Path: path, Source: "created"}
	if err := repo.Create(item); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		calls = append(calls, append([]string{}, args...))
		if len(args) == 1 && args[0] == "pull" {
			return "", fmt.Errorf("pull failed")
		}
		return "", nil
	}, nil, t.TempDir())

	err := svc.OperateCompose(dto.ComposeOperate{ID: item.ID, Operation: "update"})
	if err == nil {
		t.Fatal("expected pull error")
	}
	if len(calls) != 1 || !reflect.DeepEqual(calls[0], []string{"pull"}) {
		t.Fatalf("calls = %#v, want only pull", calls)
	}
}

func TestOperateBusyRejectsSecondCall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	item := &model.ComposeProject{Name: "busyproj", Path: path, Source: "attached"}
	if err := repo.Create(item); err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	var startOnce sync.Once
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		startOnce.Do(func() { close(started) })
		<-release
		return "", nil
	}, nil, t.TempDir())

	done := make(chan error, 1)
	go func() {
		done <- svc.OperateCompose(dto.ComposeOperate{ID: item.ID, Operation: "up"})
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first operate did not start")
	}

	err := svc.OperateCompose(dto.ComposeOperate{ID: item.ID, Operation: "stop"})
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("second operate = %v, want busy", err)
	}

	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("first operate: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first operate did not finish")
	}
}

func TestListMergesRegisteredAndUnmanaged(t *testing.T) {
	repo := newMemoryComposeRepo()
	blog := &model.ComposeProject{
		Name:   "blog",
		Path:   "/data/compose/blog/docker-compose.yml",
		Source: "created",
	}
	if err := repo.Create(blog); err != nil {
		t.Fatal(err)
	}
	svc := testComposeService(repo, func(name, path string, args ...string) (string, error) {
		t.Fatal("list must not run compose")
		return "", nil
	}, func() ([]composeLSItem, error) {
		return []composeLSItem{{
			Name:        "other",
			Status:      "running(1)",
			ConfigFiles: "/opt/other/docker-compose.yml",
		}}, nil
	}, "/data/compose")

	items, err := svc.ListComposeProjects()
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]dto.ComposeInfo{}
	for _, item := range items {
		byName[item.Name] = item
	}
	blogInfo, ok := byName["blog"]
	if !ok {
		t.Fatalf("missing blog in %#v", items)
	}
	if blogInfo.ID != blog.ID || blogInfo.Source != "created" || blogInfo.Status != "stopped" {
		t.Fatalf("blog = %#v", blogInfo)
	}
	other, ok := byName["other"]
	if !ok {
		t.Fatalf("missing other in %#v", items)
	}
	if other.ID != 0 || other.Source != "unmanaged" || other.Status != "running(1)" || other.Path != "/opt/other/docker-compose.yml" {
		t.Fatalf("other = %#v", other)
	}
}

func TestDeleteCreatedRemovesDir(t *testing.T) {
	root := t.TempDir()
	projDir := filepath.Join(root, "blog")
	path := filepath.Join(projDir, "docker-compose.yml")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	item := &model.ComposeProject{Name: "blog", Path: path, Source: "created"}
	if err := repo.Create(item); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		calls = append(calls, append([]string{}, args...))
		return "", nil
	}, nil, root)

	if err := svc.DeleteCompose(item.ID); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || !reflect.DeepEqual(calls[0], []string{"down"}) {
		t.Fatalf("run calls = %#v, want down", calls)
	}
	if _, err := os.Stat(projDir); !os.IsNotExist(err) {
		t.Fatalf("project dir still exists: %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("compose root must remain: %v", err)
	}
	if _, err := repo.Get(item.ID); err == nil {
		t.Fatal("row should be deleted")
	}
}

func TestDeleteAttachedKeepsFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	item := &model.ComposeProject{Name: "attached-app", Path: path, Source: "attached"}
	if err := repo.Create(item); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		calls = append(calls, append([]string{}, args...))
		return "", nil
	}, nil, root)

	if err := svc.DeleteCompose(item.ID); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || !reflect.DeepEqual(calls[0], []string{"down"}) {
		t.Fatalf("run calls = %#v, want down", calls)
	}
	if _, err := os.ReadFile(path); err != nil {
		t.Fatalf("attached file should remain: %v", err)
	}
	if _, err := repo.Get(item.ID); err == nil {
		t.Fatal("row should be deleted")
	}
}

func TestGetAndUpdateContentUsesRegisteredPath(t *testing.T) {
	dir := t.TempDir()
	registered := filepath.Join(dir, "registered.yml")
	other := filepath.Join(dir, "other.yml")
	if err := os.WriteFile(registered, []byte("original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("untouched\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	item := &model.ComposeProject{Name: "blog", Path: registered, Source: "attached"}
	if err := repo.Create(item); err != nil {
		t.Fatal(err)
	}
	svc := testComposeService(repo, func(name, path string, args ...string) (string, error) {
		t.Fatal("content get/update must not run compose")
		return "", nil
	}, nil, dir)

	got, err := svc.GetComposeContent(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != "original\n" {
		t.Fatalf("get = %q", got)
	}
	if err := svc.UpdateComposeContent(item.ID, "updated\n"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(registered)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "updated\n" {
		t.Fatalf("registered file = %q", data)
	}
	data, err = os.ReadFile(other)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "untouched\n" {
		t.Fatalf("other file changed: %q", data)
	}
}

func TestAttachOperateCreatesRow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	called := false
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		called = true
		return "", nil
	}, nil, t.TempDir())

	if err := svc.OperateCompose(dto.ComposeOperate{
		Name:      "external",
		Path:      path,
		Operation: "attach",
	}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("attach must not run compose")
	}
	got, err := repo.GetByName("external")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "attached" || got.Path != path || got.Name != "external" {
		t.Fatalf("row = %#v", got)
	}
}

func TestGetAndUpdateContentBusyWhileOperate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte("original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	item := &model.ComposeProject{Name: "update-busy", Path: path, Source: "attached"}
	if err := repo.Create(item); err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	var startOnce sync.Once
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		startOnce.Do(func() { close(started) })
		<-release
		return "", nil
	}, nil, t.TempDir())

	done := make(chan error, 1)
	go func() {
		done <- svc.OperateCompose(dto.ComposeOperate{ID: item.ID, Operation: "up"})
	}()
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("operate did not start")
	}

	err := svc.UpdateComposeContent(item.ID, "updated\n")
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("update = %v, want busy", err)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "original\n" {
		t.Fatalf("file should be unchanged while busy, got %q", data)
	}

	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("operate: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("operate did not finish")
	}
}

func TestDeleteComposeWhenFileGone(t *testing.T) {
	cases := []struct {
		name   string
		source string
		rmDir  bool
	}{
		{name: "created-file", source: "created", rmDir: false},
		{name: "created-dir", source: "created", rmDir: true},
		{name: "attached-file", source: "attached", rmDir: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			var path string
			if tc.source == "created" {
				projDir := filepath.Join(root, "blog")
				if err := os.MkdirAll(projDir, 0755); err != nil {
					t.Fatal(err)
				}
				path = filepath.Join(projDir, "docker-compose.yml")
			} else {
				path = filepath.Join(t.TempDir(), "docker-compose.yml")
			}
			if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
				t.Fatal(err)
			}
			repo := newMemoryComposeRepo()
			item := &model.ComposeProject{Name: "gone-" + tc.name, Path: path, Source: tc.source}
			if err := repo.Create(item); err != nil {
				t.Fatal(err)
			}
			if tc.rmDir {
				if err := os.RemoveAll(filepath.Dir(path)); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}

			var calls [][]string
			svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
				calls = append(calls, append([]string{}, args...))
				return "", fmt.Errorf("compose file missing")
			}, nil, root)

			if err := svc.DeleteCompose(item.ID); err != nil {
				t.Fatalf("delete missing file: %v", err)
			}
			if len(calls) != 0 {
				t.Fatalf("down should be skipped when file is gone, got %#v", calls)
			}
			if _, err := repo.Get(item.ID); err == nil {
				t.Fatal("row should be deleted")
			}
		})
	}
}

func TestDeleteComposeKeepsRowWhenDownFails(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	item := &model.ComposeProject{Name: "down-fail", Path: path, Source: "attached"}
	if err := repo.Create(item); err != nil {
		t.Fatal(err)
	}
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		return "", fmt.Errorf("stack still running")
	}, nil, root)

	err := svc.DeleteCompose(item.ID)
	if err == nil || !strings.Contains(err.Error(), "stack still running") {
		t.Fatalf("delete = %v, want stack still running", err)
	}
	if _, getErr := repo.Get(item.ID); getErr != nil {
		t.Fatalf("row should remain when down fails: %v", getErr)
	}
}

func TestCreateComposeRejectsLiveUnmanagedName(t *testing.T) {
	root := t.TempDir()
	repo := newMemoryComposeRepo()
	called := false
	svc := testComposeService(repo, func(name, path string, args ...string) (string, error) {
		called = true
		return "", nil
	}, func() ([]composeLSItem, error) {
		return []composeLSItem{{
			Name:        "blog",
			Status:      "running(1)",
			ConfigFiles: "/opt/blog/docker-compose.yml",
		}}, nil
	}, root)

	err := svc.CreateCompose(dto.ComposeCreate{Name: "blog", Content: "services: {}\n"})
	if err == nil {
		t.Fatal("expected error when live unmanaged project has the same name")
	}
	if called {
		t.Fatal("must not up when name is already live unmanaged")
	}
	wantPath := filepath.Join(root, "blog", "docker-compose.yml")
	if _, statErr := os.Stat(wantPath); !os.IsNotExist(statErr) {
		t.Fatalf("must not write panel file, stat=%v", statErr)
	}
	if items, _ := repo.List(); len(items) != 0 {
		t.Fatalf("repo should stay empty, got %#v", items)
	}
}

func TestCreateComposeAttachAllowsLiveUnmanagedName(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	called := false
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		called = true
		return "", nil
	}, func() ([]composeLSItem, error) {
		return []composeLSItem{{
			Name:        "blog",
			Status:      "running(1)",
			ConfigFiles: path,
		}}, nil
	}, root)

	if err := svc.CreateCompose(dto.ComposeCreate{Name: "blog", Path: path}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("attach create must not call runner")
	}
	row, err := repo.GetByName("blog")
	if err != nil {
		t.Fatal(err)
	}
	if row.Source != "attached" || row.Path != path {
		t.Fatalf("row = %#v", row)
	}
}

func TestCreateComposePropagatesGetByNameError(t *testing.T) {
	root := t.TempDir()
	repo := newMemoryComposeRepo()
	repo.getByNameErr = fmt.Errorf("db unavailable")
	called := false
	svc := testComposeService(repo, func(name, path string, args ...string) (string, error) {
		called = true
		return "", nil
	}, nil, root)

	err := svc.CreateCompose(dto.ComposeCreate{Name: "blog", Content: "services: {}\n"})
	if err == nil || !strings.Contains(err.Error(), "db unavailable") {
		t.Fatalf("create = %v, want db unavailable", err)
	}
	if called {
		t.Fatal("must not up when GetByName fails")
	}
	wantPath := filepath.Join(root, "blog", "docker-compose.yml")
	if _, statErr := os.Stat(wantPath); !os.IsNotExist(statErr) {
		t.Fatalf("must not write file when GetByName fails, stat=%v", statErr)
	}
	if items, _ := repo.List(); len(items) != 0 {
		t.Fatalf("repo should stay empty, got %#v", items)
	}
}

func TestCreateComposeAttachPropagatesGetByNameError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	repo.getByNameErr = fmt.Errorf("db unavailable")
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		t.Fatal("runner must not be called")
		return "", nil
	}, nil, t.TempDir())

	err := svc.CreateCompose(dto.ComposeCreate{Name: "blog", Path: path})
	if err == nil || !strings.Contains(err.Error(), "db unavailable") {
		t.Fatalf("attach create = %v, want db unavailable", err)
	}
	if items, _ := repo.List(); len(items) != 0 {
		t.Fatalf("repo should stay empty, got %#v", items)
	}
}

func TestAttachOperatePropagatesGetByNameError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	repo := newMemoryComposeRepo()
	repo.getByNameErr = fmt.Errorf("db unavailable")
	svc := testComposeService(repo, func(name, p string, args ...string) (string, error) {
		t.Fatal("runner must not be called")
		return "", nil
	}, nil, t.TempDir())

	err := svc.OperateCompose(dto.ComposeOperate{
		Name:      "external",
		Path:      path,
		Operation: "attach",
	})
	if err == nil || !strings.Contains(err.Error(), "db unavailable") {
		t.Fatalf("attach = %v, want db unavailable", err)
	}
	if items, _ := repo.List(); len(items) != 0 {
		t.Fatalf("repo should stay empty, got %#v", items)
	}
}

type composeRunCall struct {
	name string
	path string
	args []string
}
