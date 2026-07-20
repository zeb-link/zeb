package shell

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/tui/intro"
)

func TestFocusCyclesForwardWithTab(t *testing.T) {
	model := testModel()

	for i, want := range []focusArea{focusSort, focusCollection, focusDomain, focusInput} {
		model = updateKey(t, model, tea.KeyTab)
		if model.focus != want {
			t.Fatalf("focus after tab %d = %v, want %v", i+1, model.focus, want)
		}
	}
}

func TestFocusCyclesBackwardWithShiftTab(t *testing.T) {
	model := testModel()

	for i, want := range []focusArea{focusDomain, focusCollection, focusSort, focusInput} {
		model = updateShiftTab(t, model)
		if model.focus != want {
			t.Fatalf("focus after shift-tab %d = %v, want %v", i+1, model.focus, want)
		}
	}
}

func TestTypingOnlyChangesInputWhenInputFocused(t *testing.T) {
	model := testModel()
	model = updateRunes(t, model, "h")
	if got := model.commandInput.Value(); got != "h" {
		t.Fatalf("input value while input focused = %q, want h", got)
	}

	model = updateKey(t, model, tea.KeyTab)
	model = updateRunes(t, model, "x")
	if got := model.commandInput.Value(); got != "h" {
		t.Fatalf("input value while domain focused = %q, want unchanged h", got)
	}
}

func TestArrowsAffectFocusedPickerOnly(t *testing.T) {
	model := testModel()
	model = updateKey(t, model, tea.KeyDown)
	if model.linkIndex != 1 {
		t.Fatalf("link index after down on input = %d, want 1", model.linkIndex)
	}
	if model.domainIndex != 0 {
		t.Fatalf("domain index after down on input = %d, want unchanged 0", model.domainIndex)
	}

	model = updateKey(t, model, tea.KeyTab)
	model = updateKey(t, model, tea.KeyTab)
	model = updateKey(t, model, tea.KeyRight)
	if model.collectionIndex != 1 {
		t.Fatalf("collection index after right on collection = %d, want 1", model.collectionIndex)
	}
	if model.domainIndex != 0 {
		t.Fatalf("domain index after right on collection = %d, want unchanged 0", model.domainIndex)
	}
	// Selecting a collection starts a scoped reload; land it so the next
	// keys are not swallowed by the loading guard.
	model = updateMsg(t, model, linksReloadedMsg{seq: model.loadSeq, response: api.ListLinksResponse{}})

	model = updateKey(t, model, tea.KeyTab)
	model = updateKey(t, model, tea.KeyRight)
	if model.domainIndex != 1 {
		t.Fatalf("domain index after right on domain = %d, want 1", model.domainIndex)
	}
}

func TestFooterRendersInputBeforePickers(t *testing.T) {
	model := testModel()
	view := model.View().Content

	inputIndex := strings.Index(view, "zeb >")
	domainIndex := strings.Index(view, "Domain")
	collectionIndex := strings.Index(view, "Collection")
	if inputIndex < 0 {
		t.Fatalf("rendered view does not contain input prompt:\n%s", view)
	}
	if domainIndex < 0 || collectionIndex < 0 {
		t.Fatalf("rendered view does not contain context pickers:\n%s", view)
	}
	if inputIndex > domainIndex || inputIndex > collectionIndex {
		t.Fatalf("input rendered after pickers: input=%d domain=%d collection=%d", inputIndex, domainIndex, collectionIndex)
	}
}

func TestFooterHelpSeparatesKeysActionsAndState(t *testing.T) {
	help := footerHelp(focusDomain, true, false, false)
	for _, text := range []string{"←/→", "domain", "tab", "copy link", "saves on quit"} {
		if !strings.Contains(help, text) {
			t.Fatalf("footerHelp() = %q, want %q", help, text)
		}
	}
}

func TestNewCollectionModeStartsFromCollectionPicker(t *testing.T) {
	model := testModel()
	model = updateKey(t, model, tea.KeyTab)
	model = updateKey(t, model, tea.KeyTab)
	model = updateRunes(t, model, "n")
	if model.focus != focusNewCollection {
		t.Fatalf("focus after n on collection picker = %v, want new collection", model.focus)
	}

	model = updateRunes(t, model, "Projects")
	if got := model.collectionInput.Value(); got != "Projects" {
		t.Fatalf("collection input = %q, want Projects", got)
	}
	if got := model.commandInput.Value(); got != "" {
		t.Fatalf("command input changed in new collection mode = %q, want empty", got)
	}

	model = updateKey(t, model, tea.KeyEsc)
	if model.focus != focusCollection {
		t.Fatalf("focus after cancelling new collection = %v, want collection", model.focus)
	}
	if got := model.collectionInput.Value(); got != "" {
		t.Fatalf("collection input after cancel = %q, want empty", got)
	}
}

func TestCreateCollectionResultSelectsNewCollection(t *testing.T) {
	model := testModel()
	model = updateMsg(t, model, createCollectionResultMsg{
		response: api.CreateCollectionResponse{
			Collection: api.Collection{ID: "col_new", Name: "Projects", Type: "manual"},
		},
	})

	if got := model.ActiveCollection(); got != "col_new" {
		t.Fatalf("ActiveCollection() = %q, want col_new", got)
	}
	if !model.CollectionChanged() {
		t.Fatalf("CollectionChanged() = false, want true")
	}
	if model.focus != focusCollection {
		t.Fatalf("focus after create collection result = %v, want collection", model.focus)
	}
}

func TestViewNeverExceedsTerminalHeight(t *testing.T) {
	model := manyLinksModel(60)
	model = updateMsg(t, model, tea.WindowSizeMsg{Width: 100, Height: 24})

	view := model.View().Content
	if got := strings.Count(view, "\n") + 1; got > 24 {
		t.Fatalf("view height = %d lines, want <= terminal height 24", got)
	}
	if !strings.Contains(view, "zeb >") {
		t.Fatalf("footer input not visible in view:\n%s", view)
	}
}

func TestScrollWindowFollowsSelection(t *testing.T) {
	model := manyLinksModel(60)
	model = updateMsg(t, model, tea.WindowSizeMsg{Width: 100, Height: 24})

	for i := 0; i < 30; i++ {
		model = updateKey(t, model, tea.KeyDown)
	}
	if model.linkIndex != 30 {
		t.Fatalf("link index after 30 downs = %d, want 30", model.linkIndex)
	}
	if model.scrollOffset == 0 {
		t.Fatalf("scrollOffset stayed 0 after moving far past the viewport")
	}
	view := model.View().Content
	if !strings.Contains(view, "/path-30") {
		t.Fatalf("selected row not rendered in scrolled view:\n%s", view)
	}

	for i := 0; i < 30; i++ {
		model = updateKey(t, model, tea.KeyUp)
	}
	if model.scrollOffset != 0 {
		t.Fatalf("scrollOffset after returning to top = %d, want 0", model.scrollOffset)
	}
}

func TestMovingNearEndRequestsNextPage(t *testing.T) {
	model := manyLinksModel(50)
	cursor := "cur_next"
	model.nextCursor = &cursor
	model = updateMsg(t, model, tea.WindowSizeMsg{Width: 100, Height: 24})

	fired := false
	for i := 0; i < 45; i++ {
		var cmd tea.Cmd
		model, cmd = updateKeyCmd(t, model, tea.KeyDown)
		if cmd != nil {
			fired = true
		}
	}
	if !fired {
		t.Fatalf("no load-more command fired near the end of the loaded rows")
	}
	if !model.loadingMore {
		t.Fatalf("loadingMore = false after load-more fired")
	}

	model = updateMsg(t, model, moreLinksMsg{
		seq:      model.loadSeq,
		response: api.ListLinksResponse{Links: []api.Link{{ID: "lnk_appended", Hostname: "zbra.local", Path: "/appended"}}},
	})
	if len(model.links) != 51 {
		t.Fatalf("links after append = %d, want 51", len(model.links))
	}
	if model.loadingMore {
		t.Fatalf("loadingMore still true after page arrived")
	}
	if model.nextCursor != nil {
		t.Fatalf("nextCursor = %v, want nil after final page", *model.nextCursor)
	}
}

func TestStaleListResponsesAreDropped(t *testing.T) {
	model := manyLinksModel(5)
	model.loadSeq = 7
	model = updateMsg(t, model, linksReloadedMsg{
		seq:      3,
		response: api.ListLinksResponse{Links: []api.Link{{ID: "lnk_stale"}}},
	})
	if len(model.links) != 5 {
		t.Fatalf("stale reload replaced links: len = %d, want 5", len(model.links))
	}
}

func TestEnterWithTextStartsSearchAndEscClearsIt(t *testing.T) {
	model := testModel()
	model = updateRunes(t, model, "cnn")
	model, cmd := updateKeyCmd(t, model, tea.KeyEnter)
	if cmd == nil {
		t.Fatalf("no search command fired for free text")
	}
	if model.searchQuery != "cnn" {
		t.Fatalf("searchQuery = %q, want cnn", model.searchQuery)
	}
	if !model.loading {
		t.Fatalf("loading = false while search in flight")
	}

	model = updateMsg(t, model, searchResultMsg{
		seq:   model.loadSeq,
		query: "cnn",
		response: api.QueryLinksResponse{
			Links: []api.Link{{ID: "lnk_hit", Hostname: "zbra.local", Path: "/hit", IsActive: true}},
			Total: 41,
		},
	})
	if len(model.links) != 1 || model.links[0].ID != "lnk_hit" {
		t.Fatalf("links after search = %+v, want the one hit", model.links)
	}
	if model.searchTotal != 41 {
		t.Fatalf("searchTotal = %d, want 41", model.searchTotal)
	}
	if !strings.Contains(model.message, "41") {
		t.Fatalf("message %q does not report the match total", model.message)
	}

	model, cmd = updateKeyCmd(t, model, tea.KeyEsc)
	if model.searchQuery != "" {
		t.Fatalf("searchQuery after esc = %q, want cleared", model.searchQuery)
	}
	if cmd == nil {
		t.Fatalf("no reload command fired after clearing search")
	}
	if !model.loading {
		t.Fatalf("loading = false while post-search reload in flight")
	}
}

func TestEnterWithURLStillCreates(t *testing.T) {
	model := testModel()
	model = updateRunes(t, model, "https://example.com/page")
	model, cmd := updateKeyCmd(t, model, tea.KeyEnter)
	if cmd == nil {
		t.Fatalf("no create command fired for URL input")
	}
	if model.searchQuery != "" {
		t.Fatalf("URL input started a search: %q", model.searchQuery)
	}
	if !strings.Contains(model.message, "Creating link") {
		t.Fatalf("message = %q, want creating-link progress", model.message)
	}
}

func TestCyclingCollectionReloadsScopedList(t *testing.T) {
	model := testModel()
	model = updateKey(t, model, tea.KeyTab)
	model = updateKey(t, model, tea.KeyTab)
	if model.focus != focusCollection {
		t.Fatalf("focus = %v, want collection", model.focus)
	}
	model, cmd := updateKeyCmd(t, model, tea.KeyRight)
	if model.collectionIndex != 1 {
		t.Fatalf("collectionIndex = %d, want 1", model.collectionIndex)
	}
	if cmd == nil {
		t.Fatalf("no reload command fired after selecting a collection")
	}
	if !model.loading {
		t.Fatalf("loading = false while collection reload in flight")
	}

	model = updateMsg(t, model, linksReloadedMsg{
		seq:      model.loadSeq,
		response: api.ListLinksResponse{Links: []api.Link{{ID: "lnk_scoped", Hostname: "zbra.local", Path: "/scoped"}}},
	})
	if len(model.links) != 1 || model.links[0].ID != "lnk_scoped" {
		t.Fatalf("links after collection reload = %+v, want the scoped row", model.links)
	}
}

func TestEnterWithEmptyInputCopiesSelectedLink(t *testing.T) {
	model := testModel()
	model = updateKey(t, model, tea.KeyDown)
	model, cmd := updateKeyCmd(t, model, tea.KeyEnter)
	if cmd == nil {
		t.Fatalf("no copy command fired for enter on empty input")
	}
	if !strings.Contains(model.message, "Copied") || !strings.Contains(model.message, "/two") {
		t.Fatalf("message = %q, want copied confirmation for the selected link", model.message)
	}
}

func TestCKeyCopiesWhenPickerFocused(t *testing.T) {
	model := testModel()
	model = updateKey(t, model, tea.KeyTab)
	model, cmd := updateKeyCmd(t, model, updateRunesKey("c"))
	if cmd == nil {
		t.Fatalf("no copy command fired for c on a picker")
	}
	if !strings.Contains(model.message, "Copied") {
		t.Fatalf("message = %q, want copied confirmation", model.message)
	}
}

func TestSortChipCyclesAndReloads(t *testing.T) {
	model := testModel()
	model = updateKey(t, model, tea.KeyTab)
	if model.focus != focusSort {
		t.Fatalf("focus = %v, want sort", model.focus)
	}
	model, cmd := updateKeyCmd(t, model, tea.KeyRight)
	if model.apiSort() != "edit-date-desc" {
		t.Fatalf("apiSort after right = %q, want edit-date-desc", model.apiSort())
	}
	if cmd == nil || !model.loading {
		t.Fatalf("sort change did not reload (cmd=%v loading=%v)", cmd, model.loading)
	}
	model = updateMsg(t, model, linksReloadedMsg{seq: model.loadSeq, response: api.ListLinksResponse{}})

	model, cmd = updateKeyCmd(t, model, tea.KeyUp)
	if model.apiSort() != "edit-date-asc" {
		t.Fatalf("apiSort after direction toggle = %q, want edit-date-asc", model.apiSort())
	}
	if cmd == nil {
		t.Fatalf("direction toggle did not reload")
	}
}

func manyLinksModel(count int) Model {
	links := make([]api.Link, 0, count)
	for i := 0; i < count; i++ {
		links = append(links, api.Link{
			ID:        fmt.Sprintf("lnk_%03d", i),
			TargetURL: fmt.Sprintf("https://example.com/%d", i),
			Hostname:  "zbra.local",
			Path:      fmt.Sprintf("/path-%d", i),
			IsActive:  true,
		})
	}
	model := New(intro.Variant{}, Data{Links: links})
	model.showingIntro = false
	return model
}

func testModel() Model {
	model := New(intro.Variant{}, Data{
		Links: []api.Link{
			{ID: "lnk_1", TargetURL: "https://example.com/1", Hostname: "zbra.local", Path: "/one", IsActive: true},
			{ID: "lnk_2", TargetURL: "https://example.com/2", Hostname: "zbra.local", Path: "/two", IsActive: true},
		},
		Domains: []api.Domain{
			{Hostname: "custom.example", Type: "custom"},
		},
		Collections: []api.Collection{
			{ID: "col_1", Name: "Inbox", Type: "manual"},
		},
	})
	model.showingIntro = false
	return model
}

// updateKey sends a named key press (v2 key codes are runes, e.g. tea.KeyTab).
func updateKey(t *testing.T, model Model, code rune) Model {
	t.Helper()
	return updateMsg(t, model, tea.KeyPressMsg{Code: code})
}

// updateShiftTab sends shift+tab, which in v2 is tab with the shift modifier
// (there is no dedicated KeyShiftTab code). Its String() renders "shift+tab".
func updateShiftTab(t *testing.T, model Model) Model {
	t.Helper()
	return updateMsg(t, model, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
}

// updateRunes types printable text. A single press carries the whole string in
// Text; the textinput bubble inserts every rune of msg.Text.
func updateRunes(t *testing.T, model Model, value string) Model {
	t.Helper()
	return updateMsg(t, model, tea.KeyPressMsg{Code: []rune(value)[0], Text: value})
}

func updateMsg(t *testing.T, model Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := updateMsgCmd(t, model, msg)
	return next
}

// updateKeyCmd is updateKey but also returns the command, for tests that
// assert a background fetch was (or was not) kicked off.
func updateKeyCmd(t *testing.T, model Model, code rune) (Model, tea.Cmd) {
	t.Helper()
	return updateMsgCmd(t, model, tea.KeyPressMsg{Code: code})
}

// updateRunesKey builds the key code for a printable single-character press,
// for use with updateKeyCmd. (Printable presses carry their text in Code.)
func updateRunesKey(value string) rune {
	return []rune(value)[0]
}

func updateMsgCmd(t *testing.T, model Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := model.Update(msg)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want shell.Model", updated)
	}
	return next, cmd
}
