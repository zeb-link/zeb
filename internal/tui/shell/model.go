// Package shell contains the main Bubble Tea model for `zeb tui`.
// It is a live API-backed link browser with active create context controls.
//
// The list is a scrolling window over the loaded links: only the rows that fit
// between the header and footer are rendered, the window follows the
// selection, and moving near the end of the loaded set fetches the next page
// (cursor paging for browsing, offset paging for search). Typing a URL into
// the command line creates a link; typing anything else searches.
package shell

import (
	"context"
	"fmt"
	"image/color"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/zeb-link/zeb/internal/api"
	"github.com/zeb-link/zeb/internal/tui/intro"
	"github.com/zeb-link/zeb/internal/ui/layout"
	"github.com/zeb-link/zeb/internal/ui/theme"
)

const (
	listLimit = 50
	// loadLookahead is how close to the end of the loaded rows the selection
	// may get before the next page is fetched in the background.
	loadLookahead = 10
	pageJump      = 10
)

type Data struct {
	Client           *api.Client
	SpaceID          string
	Links            []api.Link
	NextCursor       *string
	Domains          []api.Domain
	Collections      []api.Collection
	ActiveDomain     string
	ActiveCollection string
}

type Model struct {
	spinner         spinner.Model
	commandInput    textinput.Model
	collectionInput textinput.Model
	intro           intro.Variant
	frame           int
	showingIntro    bool

	width  int
	height int

	client  *api.Client
	spaceID string

	links       []api.Link
	nextCursor  *string
	domains     []api.Domain
	collections []api.Collection

	linkIndex         int
	scrollOffset      int
	domainIndex       int
	collectionIndex   int
	sortIndex         int
	sortAsc           bool
	focus             focusArea
	domainChanged     bool
	collectionChanged bool

	loading     bool
	loadingMore bool
	// loadSeq stamps every async list request; results carrying a stale seq
	// are dropped, so rapid collection cycling or a new search can never be
	// overwritten by a slower earlier response.
	loadSeq int

	searchQuery string
	searchTotal int

	message    string
	messageBad bool
}

type focusArea int

// Tab order mirrors how often each control is reached for: sort first, then
// the collection picker, then the domain changer.
const (
	focusInput focusArea = iota
	focusSort
	focusCollection
	focusDomain
	focusNewCollection
)

// focusCycle is how many areas tab moves through (the new-collection input is
// entered explicitly with n, never by tabbing).
const focusCycle = 4

// sortField is one of the four list orderings, matching the web app. The API
// value is key + "-desc"/"-asc".
type sortField struct {
	key   string
	label string
}

var sortFields = []sortField{
	{key: "creation-date", label: "created"},
	{key: "edit-date", label: "edited"},
	{key: "recent-clicks", label: "clicked"},
	{key: "total-clicks", label: "total clicks"},
}

type introTickMsg time.Time

type createResultMsg struct {
	response api.CreateLinkResponse
	err      error
}

type linksReloadedMsg struct {
	seq      int
	response api.ListLinksResponse
	err      error
}

type moreLinksMsg struct {
	seq      int
	response api.ListLinksResponse
	err      error
}

type searchResultMsg struct {
	seq      int
	query    string
	offset   int
	response api.QueryLinksResponse
	err      error
}

type createCollectionResultMsg struct {
	response api.CreateCollectionResponse
	err      error
}

func New(variant intro.Variant, data Data) Model {
	commandInput := textinput.New()
	commandInput.Prompt = "zeb > "
	commandInput.Placeholder = "paste a URL to create · type to search"
	commandInput.CharLimit = 600
	commandInput.Focus()

	collectionInput := textinput.New()
	collectionInput.Prompt = "collection > "
	collectionInput.Placeholder = "new collection name"
	collectionInput.CharLimit = 120

	model := Model{
		spinner:         spinner.New(spinner.WithSpinner(spinner.Dot)),
		commandInput:    commandInput,
		collectionInput: collectionInput,
		intro:           variant,
		showingIntro:    true,
		width:           112,
		height:          32,
		client:          data.Client,
		spaceID:         data.SpaceID,
		links:           data.Links,
		nextCursor:      data.NextCursor,
		domains:         data.Domains,
		collections:     data.Collections,
		message:         "Type a URL to create a link, or text to search.",
		domainIndex:     -1,
		collectionIndex: -1,
	}
	model.domainIndex = model.indexForDomain(data.ActiveDomain)
	model.collectionIndex = model.indexForCollection(data.ActiveCollection)
	if data.ActiveCollection != "" && model.collectionIndex == 0 {
		model.message = "Active collection cannot accept new links here; choose a collection or no collection."
	}
	return model
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, introTick(), textinput.Blink)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureVisible()
	case tea.KeyPressMsg:
		if m.showingIntro {
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			}
			return m, nil
		}
		if m.loading {
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			}
			return m, nil
		}
		if m.focus == focusNewCollection {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.cancelNewCollection()
				return m, nil
			case "enter":
				name := strings.TrimSpace(m.collectionInput.Value())
				if name == "" {
					m.setMessage("Collection name cannot be blank.", true)
					return m, nil
				}
				m.loading = true
				m.setMessage("Creating collection...", false)
				return m, m.createCollectionCmd(name)
			}
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.searchQuery != "" {
				return m, m.clearSearch()
			}
			return m, tea.Quit
		case "q":
			if m.focus != focusInput {
				return m, tea.Quit
			}
		case "tab":
			m.focusNext()
			return m, m.focusCmd()
		case "shift+tab", "backtab":
			m.focusPrevious()
			return m, m.focusCmd()
		case "up":
			switch m.focus {
			case focusInput:
				return m, m.moveLink(-1)
			case focusDomain:
				m.cycleDomain(-1)
			case focusCollection:
				return m, m.cycleCollection(-1)
			case focusSort:
				return m, m.toggleSortDirection()
			}
			return m, nil
		case "down":
			switch m.focus {
			case focusInput:
				return m, m.moveLink(1)
			case focusDomain:
				m.cycleDomain(1)
			case focusCollection:
				return m, m.cycleCollection(1)
			case focusSort:
				return m, m.toggleSortDirection()
			}
			return m, nil
		case "pgup":
			if m.focus == focusInput {
				return m, m.moveLink(-pageJump)
			}
		case "pgdown":
			if m.focus == focusInput {
				return m, m.moveLink(pageJump)
			}
		case "home":
			if m.focus == focusInput {
				return m, m.moveLink(-len(m.links))
			}
		case "end":
			if m.focus == focusInput {
				return m, m.moveLink(len(m.links))
			}
		case "left":
			switch m.focus {
			case focusDomain:
				m.cycleDomain(-1)
				return m, nil
			case focusCollection:
				return m, m.cycleCollection(-1)
			case focusSort:
				return m, m.cycleSort(-1)
			}
		case "right":
			switch m.focus {
			case focusDomain:
				m.cycleDomain(1)
				return m, nil
			case focusCollection:
				return m, m.cycleCollection(1)
			case focusSort:
				return m, m.cycleSort(1)
			}
		case "r":
			if m.focus != focusInput {
				return m, m.startReload("Refreshing")
			}
		case "c":
			if m.focus != focusInput {
				return m, m.copySelectedLink()
			}
		case "n":
			if m.focus == focusCollection {
				m.startNewCollection()
				return m, m.focusCmd()
			}
		case "enter":
			if m.focus == focusInput {
				value := strings.TrimSpace(m.commandInput.Value())
				if value == "" {
					// An empty command line makes enter act on the selection:
					// copy its short URL to the clipboard.
					return m, m.copySelectedLink()
				}
				if looksLikeCreateInput(value) {
					if err := validateHTTPURL(value); err != nil {
						m.setMessage(err.Error(), true)
						return m, nil
					}
					m.loading = true
					m.setMessage("Creating link...", false)
					return m, m.createLinkCmd(value)
				}
				return m, m.startSearch(value)
			}
			m.setFocus(focusInput)
			return m, m.focusCmd()
		}
	case introTickMsg:
		if m.showingIntro {
			m.frame++
			if m.frame >= intro.FrameCount {
				m.showingIntro = false
				return m, nil
			}
			return m, introTick()
		}
	case createResultMsg:
		m.loading = false
		if msg.err != nil {
			m.setMessage(msg.err.Error(), true)
			return m, nil
		}
		m.commandInput.SetValue("")
		m.links = append([]api.Link{msg.response.Link}, m.links...)
		m.linkIndex = 0
		m.scrollOffset = 0
		m.setMessage(fmt.Sprintf("Created %s", displayShortLink(msg.response.Link)), false)
		return m, nil
	case linksReloadedMsg:
		if msg.seq != m.loadSeq {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.setMessage(msg.err.Error(), true)
			return m, nil
		}
		m.links = msg.response.Links
		m.nextCursor = msg.response.NextCursor
		m.linkIndex = 0
		m.scrollOffset = 0
		m.setMessage(m.loadedMessage(), false)
		return m, nil
	case moreLinksMsg:
		if msg.seq != m.loadSeq {
			return m, nil
		}
		m.loadingMore = false
		if msg.err != nil {
			m.setMessage(msg.err.Error(), true)
			return m, nil
		}
		m.links = append(m.links, msg.response.Links...)
		m.nextCursor = msg.response.NextCursor
		return m, nil
	case searchResultMsg:
		if msg.seq != m.loadSeq {
			return m, nil
		}
		m.loading = false
		m.loadingMore = false
		if msg.err != nil {
			m.setMessage(msg.err.Error(), true)
			return m, nil
		}
		m.searchTotal = msg.response.Total
		if msg.offset == 0 {
			m.links = msg.response.Links
			m.linkIndex = 0
			m.scrollOffset = 0
		} else {
			m.links = append(m.links, msg.response.Links...)
		}
		m.nextCursor = nil
		if msg.response.Total == 0 {
			m.setMessage(fmt.Sprintf("No links match %q.", msg.query), false)
		} else {
			m.setMessage(fmt.Sprintf("%d links match %q. Esc clears the search.", msg.response.Total, msg.query), false)
		}
		return m, nil
	case createCollectionResultMsg:
		m.loading = false
		if msg.err != nil {
			m.setMessage(msg.err.Error(), true)
			return m, nil
		}
		collection := msg.response.Collection
		m.collections = append(m.collections, collection)
		m.collectionIndex = m.indexForCollection(collection.ID)
		m.collectionChanged = true
		m.collectionInput.SetValue("")
		m.setFocus(focusCollection)
		m.setMessage(fmt.Sprintf("Created collection %s; new links go there.", collection.Name), false)
		if m.searchQuery == "" {
			return m, m.startReload("Loading")
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	switch {
	case m.focus == focusInput && !m.showingIntro:
		var inputCmd tea.Cmd
		m.commandInput, inputCmd = m.commandInput.Update(msg)
		cmd = tea.Batch(cmd, inputCmd)
	case m.focus == focusNewCollection && !m.showingIntro:
		var inputCmd tea.Cmd
		m.collectionInput, inputCmd = m.collectionInput.Update(msg)
		cmd = tea.Batch(cmd, inputCmd)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	if m.showingIntro {
		return altView("\n" + m.intro.RenderFrame(m.frame) + "\n")
	}

	width := m.effectiveWidth()
	header := m.renderHeader(width)
	footer := m.renderFooter(width)
	viewHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer) - 1
	list := m.renderLinkList(width, viewHeight)

	content := lipgloss.JoinVertical(lipgloss.Left, header, list)
	if m.height > 0 {
		content = layout.FillHeight(width, m.height-lipgloss.Height(footer)-1, content)
	}
	return altView(content + "\n" + footer)
}

// altView wraps rendered content as a full-screen (alt-screen) frame. In
// lipgloss/bubbletea v2, alt-screen is requested per-frame via View.AltScreen
// rather than a program option.
func altView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) effectiveWidth() int {
	if m.width < 78 {
		return 78
	}
	return m.width
}

func (m Model) ActiveDomain() string {
	return m.domainOption().Hostname
}

func (m Model) ActiveCollection() string {
	return m.collectionOption().ID
}

func (m Model) ContextChanged() bool {
	return m.domainChanged || m.collectionChanged
}

func (m Model) DomainChanged() bool {
	return m.domainChanged
}

func (m Model) CollectionChanged() bool {
	return m.collectionChanged
}

func (m *Model) setMessage(text string, bad bool) {
	m.message = text
	m.messageBad = bad
}

func (m *Model) focusNext() {
	m.setFocus((m.focus + 1) % focusCycle)
}

func (m *Model) focusPrevious() {
	if m.focus == focusInput {
		m.setFocus(focusDomain)
		return
	}
	m.setFocus(m.focus - 1)
}

func (m *Model) setFocus(focus focusArea) {
	m.focus = focus
	if focus == focusInput {
		m.commandInput.Focus()
		m.collectionInput.Blur()
		m.setMessage("Type a URL to create a link, or text to search.", false)
		return
	}
	m.commandInput.Blur()
	if focus == focusNewCollection {
		m.collectionInput.Focus()
		m.setMessage("Name the new collection, then press enter.", false)
		return
	}
	m.collectionInput.Blur()
	switch focus {
	case focusDomain:
		m.setMessage("Domain picker focused. Use arrows to choose the domain for new links.", false)
	case focusCollection:
		m.setMessage("Collection picker focused. Arrows browse a collection, n makes a new one.", false)
	case focusSort:
		m.setMessage("Sort focused. ←/→ pick the order, ↑/↓ flip direction.", false)
	}
}

// copyReady reports whether enter would copy the selected link: command line
// focused and empty, with links on screen.
func (m Model) copyReady() bool {
	return m.focus == focusInput && strings.TrimSpace(m.commandInput.Value()) == "" && len(m.links) > 0
}

func (m Model) focusCmd() tea.Cmd {
	if m.focus != focusInput && m.focus != focusNewCollection {
		return nil
	}
	return textinput.Blink
}

func (m *Model) startNewCollection() {
	m.collectionInput.SetValue("")
	m.setFocus(focusNewCollection)
}

func (m *Model) cancelNewCollection() {
	m.collectionInput.SetValue("")
	m.setFocus(focusCollection)
	m.setMessage("New collection cancelled.", false)
}

// moveLink moves the selection, keeps it inside the visible window, and kicks
// off a background fetch of the next page when the selection nears the end of
// what is loaded.
func (m *Model) moveLink(delta int) tea.Cmd {
	m.linkIndex = clamp(m.linkIndex+delta, 0, len(m.links)-1)
	m.ensureVisible()
	return m.maybeLoadMore()
}

// ensureVisible adjusts scrollOffset so the selected row is fully inside the
// list viewport: never above the window, and scrolled down just far enough
// that the selection's full row fits.
func (m *Model) ensureVisible() {
	if len(m.links) == 0 {
		m.scrollOffset = 0
		return
	}
	m.linkIndex = clamp(m.linkIndex, 0, len(m.links)-1)
	m.scrollOffset = clamp(m.scrollOffset, 0, len(m.links)-1)
	if m.scrollOffset > m.linkIndex {
		m.scrollOffset = m.linkIndex
	}
	viewHeight := m.listViewHeight()
	if viewHeight <= 0 {
		return
	}
	width := m.effectiveWidth()
	heights := make([]int, len(m.links))
	for i, link := range m.links {
		heights[i] = lipgloss.Height(m.renderLink(link, i == m.linkIndex, width))
	}
	for m.scrollOffset < m.linkIndex {
		total := 0
		for i := m.scrollOffset; i <= m.linkIndex; i++ {
			total += heights[i]
		}
		if total <= viewHeight {
			break
		}
		m.scrollOffset++
	}
}

// listViewHeight is the line budget between the header and the footer.
func (m Model) listViewHeight() int {
	if m.height <= 0 {
		return 0
	}
	width := m.effectiveWidth()
	return m.height - lipgloss.Height(m.renderHeader(width)) - lipgloss.Height(m.renderFooter(width)) - 1
}

func (m *Model) maybeLoadMore() tea.Cmd {
	if m.loading || m.loadingMore || len(m.links) == 0 {
		return nil
	}
	if m.linkIndex < len(m.links)-loadLookahead {
		return nil
	}
	if m.searchQuery != "" {
		if len(m.links) >= m.searchTotal {
			return nil
		}
		m.loadingMore = true
		return m.searchCmd(m.loadSeq, m.searchQuery, len(m.links))
	}
	if m.nextCursor == nil {
		return nil
	}
	m.loadingMore = true
	return m.loadMoreCmd(m.loadSeq, *m.nextCursor)
}

func (m *Model) startSearch(query string) tea.Cmd {
	m.loadSeq++
	m.loading = true
	m.searchQuery = query
	m.searchTotal = 0
	m.commandInput.SetValue("")
	m.setMessage(fmt.Sprintf("Searching for %q...", query), false)
	return m.searchCmd(m.loadSeq, query, 0)
}

func (m *Model) clearSearch() tea.Cmd {
	m.searchQuery = ""
	m.searchTotal = 0
	return m.startReload("Loading")
}

// startReload refetches the current list source (search results, the selected
// collection, or the whole space) from the top.
func (m *Model) startReload(verb string) tea.Cmd {
	m.loadSeq++
	m.loading = true
	if m.searchQuery != "" {
		m.setMessage(fmt.Sprintf("%s search results...", verb), false)
		return m.searchCmd(m.loadSeq, m.searchQuery, 0)
	}
	m.setMessage(fmt.Sprintf("%s %s...", verb, collectionScopeLabel(m.collectionOption())), false)
	return m.reloadCmd(m.loadSeq)
}

func (m *Model) cycleDomain(delta int) {
	options := m.domainOptions()
	if len(options) == 0 {
		return
	}
	m.domainIndex = wrapIndex(m.domainIndex+delta, len(options))
	m.domainChanged = true
	m.setMessage("New links use domain "+domainLabel(m.domainOption().Hostname), false)
}

// cycleSort moves through the four list orderings; toggleSortDirection flips
// ascending/descending. Both reload the browsed list — the search endpoint has
// no sort, so during a search the change is noted and applies after esc.
func (m *Model) cycleSort(delta int) tea.Cmd {
	m.sortIndex = wrapIndex(m.sortIndex+delta, len(sortFields))
	return m.sortChanged()
}

func (m *Model) toggleSortDirection() tea.Cmd {
	m.sortAsc = !m.sortAsc
	return m.sortChanged()
}

func (m *Model) sortChanged() tea.Cmd {
	if m.searchQuery != "" {
		m.setMessage("Sorted by "+m.sortLabel()+" while browsing. Esc clears the search to see it.", false)
		return nil
	}
	return m.startReload("Sorting")
}

// apiSort is the wire value for the current sort selection.
func (m Model) apiSort() string {
	direction := "-desc"
	if m.sortAsc {
		direction = "-asc"
	}
	return sortFields[wrapIndex(m.sortIndex, len(sortFields))].key + direction
}

func (m Model) sortLabel() string {
	arrow := "↓"
	if m.sortAsc {
		arrow = "↑"
	}
	return sortFields[wrapIndex(m.sortIndex, len(sortFields))].label + " " + arrow
}

// copySelectedLink puts the selected short URL on the clipboard two ways at
// once: OSC 52 through the terminal (works locally and over ssh in most
// modern emulators) and pbcopy on macOS as a belt-and-braces fallback.
func (m *Model) copySelectedLink() tea.Cmd {
	if len(m.links) == 0 {
		return nil
	}
	short := displayShortLink(m.links[clamp(m.linkIndex, 0, len(m.links)-1)])
	m.setMessage("Copied "+short+" to the clipboard.", false)
	return tea.Batch(tea.SetClipboard(short), systemClipboardCmd(short))
}

func systemClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		if runtime.GOOS == "darwin" {
			cmd := exec.Command("pbcopy")
			cmd.Stdin = strings.NewReader(text)
			_ = cmd.Run()
		}
		return nil
	}
}

// cycleCollection changes the active collection, which is both the create
// context and the browse filter: the list reloads scoped to the selection.
func (m *Model) cycleCollection(delta int) tea.Cmd {
	options := m.collectionOptions()
	if len(options) == 0 {
		return nil
	}
	m.collectionIndex = wrapIndex(m.collectionIndex+delta, len(options))
	m.collectionChanged = true
	if m.searchQuery != "" {
		m.setMessage("New links go to "+collectionLabel(m.collectionOption())+". Esc clears the search to browse it.", false)
		return nil
	}
	return m.startReload("Loading")
}

func (m Model) createLinkCmd(targetURL string) tea.Cmd {
	return func() tea.Msg {
		input := api.CreateLinkInput{
			TargetURL:  targetURL,
			Domain:     m.ActiveDomain(),
			Collection: m.ActiveCollection(),
		}
		response, err := m.client.CreateLink(context.Background(), m.spaceID, input, true)
		return createResultMsg{response: response, err: err}
	}
}

func (m Model) reloadCmd(seq int) tea.Cmd {
	client, spaceID := m.client, m.spaceID
	collectionID := m.collectionOption().ID
	sort := m.apiSort()
	return func() tea.Msg {
		options := api.ListLinksOptions{Limit: listLimit, Sort: sort, IncludeClicks: true}
		var response api.ListLinksResponse
		var err error
		if collectionID != "" {
			response, err = client.ListCollectionLinks(context.Background(), spaceID, collectionID, options)
		} else {
			response, err = client.ListLinks(context.Background(), spaceID, options)
		}
		return linksReloadedMsg{seq: seq, response: response, err: err}
	}
}

func (m Model) loadMoreCmd(seq int, cursor string) tea.Cmd {
	client, spaceID := m.client, m.spaceID
	collectionID := m.collectionOption().ID
	sort := m.apiSort()
	return func() tea.Msg {
		options := api.ListLinksOptions{Limit: listLimit, Cursor: cursor, Sort: sort, IncludeClicks: true}
		var response api.ListLinksResponse
		var err error
		if collectionID != "" {
			response, err = client.ListCollectionLinks(context.Background(), spaceID, collectionID, options)
		} else {
			response, err = client.ListLinks(context.Background(), spaceID, options)
		}
		return moreLinksMsg{seq: seq, response: response, err: err}
	}
}

func (m Model) searchCmd(seq int, query string, offset int) tea.Cmd {
	client, spaceID := m.client, m.spaceID
	return func() tea.Msg {
		response, err := client.QueryLinks(context.Background(), spaceID, api.QueryLinksInput{
			LinkFilter: api.LinkFilter{Query: query},
			Limit:      listLimit,
			Offset:     offset,
		})
		return searchResultMsg{seq: seq, query: query, offset: offset, response: response, err: err}
	}
}

func (m Model) createCollectionCmd(name string) tea.Cmd {
	return func() tea.Msg {
		response, err := m.client.CreateCollection(context.Background(), m.spaceID, api.CreateCollectionInput{Name: name})
		return createCollectionResultMsg{response: response, err: err}
	}
}

func (m Model) loadedMessage() string {
	scope := collectionScopeLabel(m.collectionOption())
	if m.nextCursor != nil {
		return fmt.Sprintf("Loaded the first %d links in %s.", len(m.links), scope)
	}
	return fmt.Sprintf("Loaded %d links in %s.", len(m.links), scope)
}

func (m Model) renderHeader(width int) string {
	title := theme.Heading.Render("Links")
	switch {
	case m.searchQuery != "":
		title += theme.MutedText.Render(" matching ") + theme.Title.Render(fmt.Sprintf("%q", m.searchQuery))
	case m.collectionOption().ID != "":
		title += theme.MutedText.Render(" in ") + theme.CollectionText.Render(m.collectionOption().Name)
	}
	return lipgloss.NewStyle().
		Width(width - 2).
		MarginBottom(1).
		Render(title + "\n" + theme.MutedText.Render(m.scopeLine()))
}

// scopeLine is the header's second line: where the selection sits in the full
// result set, and whether more rows exist beyond the loaded window.
func (m Model) scopeLine() string {
	if len(m.links) == 0 {
		if m.loading {
			return "loading · " + m.spinner.View()
		}
		return "empty"
	}
	position := fmt.Sprintf("%d of %d", m.linkIndex+1, len(m.links))
	extra := ""
	if m.searchQuery != "" {
		if m.searchTotal > len(m.links) {
			position = fmt.Sprintf("%d of %d matches", m.linkIndex+1, m.searchTotal)
			extra = fmt.Sprintf(" · %d loaded", len(m.links))
		} else {
			position = fmt.Sprintf("%d of %d matches", m.linkIndex+1, len(m.links))
		}
	} else if m.nextCursor != nil {
		extra = " · more available"
	}
	scope := position + extra
	if m.loadingMore {
		scope += " · loading more " + m.spinner.View()
	} else if m.loading {
		scope += " · " + m.spinner.View()
	}
	return scope
}

// renderLinkList renders the window of rows starting at scrollOffset that fits
// in viewHeight lines. ensureVisible keeps the selection inside that window.
func (m Model) renderLinkList(width int, viewHeight int) string {
	if len(m.links) == 0 {
		if m.searchQuery != "" {
			return theme.MutedText.Render(fmt.Sprintf("No links match %q.", m.searchQuery))
		}
		return theme.MutedText.Render("No links found.")
	}
	start := clamp(m.scrollOffset, 0, len(m.links)-1)
	rows := make([]string, 0, len(m.links)-start)
	used := 0
	for i := start; i < len(m.links); i++ {
		row := m.renderLink(m.links[i], i == m.linkIndex, width)
		rowHeight := lipgloss.Height(row)
		if viewHeight > 0 && used+rowHeight > viewHeight && len(rows) > 0 {
			break
		}
		rows = append(rows, row)
		used += rowHeight
		if viewHeight > 0 && used >= viewHeight {
			break
		}
	}
	return strings.TrimRight(strings.Join(rows, "\n"), "\n")
}

// renderLink renders one row. The selected row lights up: every segment gets
// a muted background wash (theme.Panel2, which adapts to light/dark), the
// short link goes bold, the target lifts from muted to body ink, and each
// line pads to full width so the wash reads as one quiet block.
func (m Model) renderLink(link api.Link, focused bool, width int) string {
	wash := func(style lipgloss.Style) lipgloss.Style {
		if focused {
			return style.Background(theme.Panel2)
		}
		return style
	}
	// A switched-off link doesn't get the live emerald: its URL drops to body
	// ink and its status reads in the web app's tomato.
	shortStyle := wash(theme.LinkText)
	if !link.IsActive {
		shortStyle = wash(theme.BodyText)
	}
	targetStyle := wash(theme.MutedText)
	titleStyle := wash(theme.MutedText)
	metaStyle := wash(theme.MutedText)
	if focused {
		shortStyle = shortStyle.Bold(true)
		targetStyle = wash(theme.BodyText)
		titleStyle = wash(theme.SubtleText)
	}

	short := displayShortLink(link)
	targetWidth := width - len(short) - 10
	if targetWidth < 28 {
		targetWidth = 28
	}
	statusTone, statusLabel := theme.Good, "active"
	if !link.IsActive {
		statusTone, statusLabel = theme.Bad, "inactive"
	}
	dot := wash(lipgloss.NewStyle().Foreground(statusTone)).Render("●")
	status := wash(lipgloss.NewStyle().Foreground(statusTone)).Render(statusLabel)
	pad := func(line string) string {
		if !focused {
			return line
		}
		gap := width - 2 - lipgloss.Width(line)
		if gap <= 0 {
			return line
		}
		return line + wash(lipgloss.NewStyle()).Render(strings.Repeat(" ", gap))
	}

	line := pad(fmt.Sprintf("%s%s %s %s", layout.Gutter(focused), dot, shortStyle.Render(short), targetStyle.Render("→ "+layout.Truncate(link.TargetURL, targetWidth))))
	title := ""
	if link.Title != nil && strings.TrimSpace(*link.Title) != "" {
		title = "\n" + pad("  "+titleStyle.Render(layout.Truncate(strings.TrimSpace(*link.Title), width-8)))
	}
	metaLine := metaStyle.Render(link.ID+" · ") + status
	if link.TotalClicks != nil {
		metaLine += metaStyle.Render(fmt.Sprintf(" · %d clicks", *link.TotalClicks))
	}
	meta := "\n" + pad("  "+metaLine)
	return line + title + meta + "\n"
}

func (m Model) renderFooter(width int) string {
	domain := contextPill("Domain", domainLabel(m.domainOption().Hostname), theme.Sand, true, m.focus == focusDomain)
	collection := contextPill("Collection", collectionLabel(m.collectionOption()), theme.Collection, m.collectionOption().ID != "", m.focus == focusCollection)
	sort := contextPill("Sort", m.sortLabel(), theme.Link, m.sortIndex != 0 || m.sortAsc, m.focus == focusSort)
	toolbar := lipgloss.JoinHorizontal(lipgloss.Center, sort, theme.MutedText.Render("  "), collection, theme.MutedText.Render("  "), domain)

	input := m.renderFooterInput()
	help := footerHelp(m.focus, m.ContextChanged(), m.searchQuery != "", m.copyReady())
	messageStyle := theme.MutedText
	if m.messageBad {
		messageStyle = theme.BadText
	}
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		input,
		toolbar,
		messageStyle.Render(m.message),
		help,
	)
	return lipgloss.NewStyle().
		Width(width-2).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(theme.Dim).
		PaddingTop(1).
		Render(body)
}

func (m Model) renderFooterInput() string {
	if m.focus == focusNewCollection {
		return m.collectionInput.View()
	}
	if m.focus == focusInput {
		return m.commandInput.View()
	}
	return theme.MutedText.Render(m.commandInput.Prompt + m.commandInput.Value())
}

type helpItem struct {
	key    string
	action string
}

func footerHelp(focus focusArea, changed bool, searching bool, copyReady bool) string {
	quit := helpItem{key: "esc", action: "quit"}
	if searching {
		quit = helpItem{key: "esc", action: "clear search"}
	}
	enter := helpItem{key: "enter", action: "create/search"}
	if copyReady {
		enter = helpItem{key: "enter", action: "copy link"}
	}
	items := []helpItem{
		{key: "tab", action: "next"},
		{key: "↑/↓", action: "move"},
		{key: "pgup/pgdn", action: "page"},
		enter,
		quit,
	}
	switch focus {
	case focusSort:
		items = []helpItem{
			{key: "←/→", action: "sort by"},
			{key: "↑/↓", action: "direction"},
			{key: "c", action: "copy link"},
			{key: "tab", action: "collection"},
			{key: "r", action: "refresh"},
			{key: "q", action: "quit"},
		}
	case focusCollection:
		items = []helpItem{
			{key: "←/→", action: "collection"},
			{key: "n", action: "new"},
			{key: "c", action: "copy link"},
			{key: "tab", action: "domain"},
			{key: "r", action: "refresh"},
			{key: "q", action: "quit"},
		}
	case focusDomain:
		items = []helpItem{
			{key: "←/→", action: "domain"},
			{key: "c", action: "copy link"},
			{key: "tab", action: "type"},
			{key: "enter", action: "type"},
			{key: "r", action: "refresh"},
			{key: "q", action: "quit"},
		}
	case focusNewCollection:
		items = []helpItem{
			{key: "enter", action: "create collection"},
			{key: "esc", action: "cancel"},
			{key: "ctrl+c", action: "quit"},
		}
	}
	parts := make([]string, 0, len(items)+1)
	for _, item := range items {
		parts = append(parts, theme.KeyText.Render(item.key)+" "+theme.MutedText.Render(item.action))
	}
	if changed {
		parts = append(parts, theme.GoodText.Render("saves on quit"))
	}
	return strings.Join(parts, lipgloss.NewStyle().Foreground(theme.Dim).Render("  ·  "))
}

func contextPill(label string, value string, tone color.Color, active bool, focused bool) string {
	return layout.Chip(label, value, layout.ChipOpts{Tone: tone, Active: active, Focused: focused})
}

func (m Model) domainOptions() []domainOption {
	options := []domainOption{{Hostname: "", Type: "default"}}
	for _, domain := range m.domains {
		options = append(options, domainOption{Hostname: domain.Hostname, Type: domain.Type})
	}
	return options
}

func (m Model) collectionOptions() []collectionOption {
	options := []collectionOption{{Name: "(no collection)"}}
	for _, collection := range m.collections {
		if collection.Type == "smart" {
			continue
		}
		options = append(options, collectionOption{ID: collection.ID, Name: collection.Name})
	}
	return options
}

func (m Model) domainOption() domainOption {
	options := m.domainOptions()
	if len(options) == 0 {
		return domainOption{}
	}
	return options[clamp(m.domainIndex, 0, len(options)-1)]
}

func (m Model) collectionOption() collectionOption {
	options := m.collectionOptions()
	if len(options) == 0 {
		return collectionOption{Name: "(no collection)"}
	}
	return options[clamp(m.collectionIndex, 0, len(options)-1)]
}

func (m Model) indexForDomain(hostname string) int {
	for i, option := range m.domainOptions() {
		if option.Hostname == hostname {
			return i
		}
	}
	return 0
}

func (m Model) indexForCollection(collectionID string) int {
	for i, option := range m.collectionOptions() {
		if option.ID == collectionID {
			return i
		}
	}
	return 0
}

type domainOption struct {
	Hostname string
	Type     string
}

type collectionOption struct {
	ID   string
	Name string
}

// looksLikeCreateInput distinguishes the command line's two intents: an
// http(s) URL means "create this link", anything else is a free-text search.
func looksLikeCreateInput(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func validateHTTPURL(value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("%q is not a valid URL", value)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%q must start with http:// or https://", value)
	}
	return nil
}

func displayShortLink(link api.Link) string {
	if link.ShortURL != "" {
		return link.ShortURL
	}
	return shortLink(link.Hostname, link.Path)
}

func shortLink(hostname string, path string) string {
	cleanPath := strings.Trim(path, "/")
	if cleanPath == "" {
		return hostname
	}
	return hostname + "/" + cleanPath
}

func domainLabel(hostname string) string {
	if hostname == "" {
		return "server default"
	}
	return hostname
}

func collectionLabel(collection collectionOption) string {
	if collection.ID == "" {
		return "(no collection)"
	}
	return collection.Name
}

// collectionScopeLabel names the current browse scope in messages.
func collectionScopeLabel(collection collectionOption) string {
	if collection.ID == "" {
		return "all links"
	}
	return collection.Name
}

func clamp(value int, min int, max int) int {
	if max < min {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func wrapIndex(value int, length int) int {
	if length <= 0 {
		return 0
	}
	value %= length
	if value < 0 {
		value += length
	}
	return value
}

func introTick() tea.Cmd {
	return tea.Tick(intro.FrameDelay, func(t time.Time) tea.Msg {
		return introTickMsg(t)
	})
}
