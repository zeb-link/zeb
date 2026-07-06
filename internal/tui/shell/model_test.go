package shell

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kerns/zlink-zeb/internal/api"
	"github.com/kerns/zlink-zeb/internal/tui/intro"
)

func TestFocusCyclesForwardWithTab(t *testing.T) {
	model := testModel()

	model = updateKey(t, model, tea.KeyTab)
	if model.focus != focusDomain {
		t.Fatalf("focus after first tab = %v, want domain", model.focus)
	}

	model = updateKey(t, model, tea.KeyTab)
	if model.focus != focusCollection {
		t.Fatalf("focus after second tab = %v, want collection", model.focus)
	}

	model = updateKey(t, model, tea.KeyTab)
	if model.focus != focusInput {
		t.Fatalf("focus after third tab = %v, want input", model.focus)
	}
}

func TestFocusCyclesBackwardWithShiftTab(t *testing.T) {
	model := testModel()

	model = updateKey(t, model, tea.KeyShiftTab)
	if model.focus != focusCollection {
		t.Fatalf("focus after shift-tab from input = %v, want collection", model.focus)
	}

	model = updateKey(t, model, tea.KeyShiftTab)
	if model.focus != focusDomain {
		t.Fatalf("focus after second shift-tab = %v, want domain", model.focus)
	}

	model = updateKey(t, model, tea.KeyShiftTab)
	if model.focus != focusInput {
		t.Fatalf("focus after third shift-tab = %v, want input", model.focus)
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
	model = updateKey(t, model, tea.KeyRight)
	if model.domainIndex != 1 {
		t.Fatalf("domain index after right on domain = %d, want 1", model.domainIndex)
	}
	if model.collectionIndex != 0 {
		t.Fatalf("collection index after right on domain = %d, want unchanged 0", model.collectionIndex)
	}

	model = updateKey(t, model, tea.KeyTab)
	model = updateKey(t, model, tea.KeyRight)
	if model.collectionIndex != 1 {
		t.Fatalf("collection index after right on collection = %d, want 1", model.collectionIndex)
	}
}

func TestFooterRendersInputBeforePickers(t *testing.T) {
	model := testModel()
	view := model.View()

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
	help := footerHelp(focusDomain, true)
	for _, text := range []string{"←/→", "domain", "tab", "collection", "saves on quit"} {
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

func updateKey(t *testing.T, model Model, key tea.KeyType) Model {
	t.Helper()
	return updateMsg(t, model, tea.KeyMsg{Type: key})
}

func updateRunes(t *testing.T, model Model, value string) Model {
	t.Helper()
	return updateMsg(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)})
}

func updateMsg(t *testing.T, model Model, msg tea.Msg) Model {
	t.Helper()
	updated, _ := model.Update(msg)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want shell.Model", updated)
	}
	return next
}
