// Package table provides a simple table component for Bubble Tea applications.
package table

import (
	"cmp"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Model defines a state for the table widget.
type Model struct {
	KeyMap KeyMap
	Help   help.Model

	cols        []Column
	rows        []Row
	mode        Mode
	cursor      [2]int // [row, col]
	fixedColumn int
	focused     bool
	showHeader  bool
	styles      Styles
	styleFunc   StyleFunc

	// colWidths holds the resolved width of each column, parallel to cols. It
	// is recomputed whenever layoutDirty is set.
	colWidths   []int
	layoutDirty bool

	viewport viewport.Model
	start    int
	end      int
}

var _ tea.Model = (*Model)(nil)

// Mode represents different states the table could be in. This can affect
// rendering and other behavior.
type Mode int

const (
	// ModeNormal is the default row selection mode.
	ModeNormal Mode = iota
	// ModeCell is the cell selection mode.
	ModeCell
	// ModeFixedColumn is the row selection mode with a fixed selected column.
	ModeFixedColumn
)

// Row represents one line in the table.
type Row []string

// Column defines the table structure. It describes the user's intent only; the
// width a column ends up with is resolved from the space available to the
// table and can be read back with [Model.ColumnWidths].
type Column struct {
	Title  string
	Sizing Sizing
	// Lower and upper bounds for the resolved width, in cells. A value of 0
	// means unbounded. If both are set and MinWidth exceeds MaxWidth, MaxWidth
	// wins.
	MinWidth int
	MaxWidth int
}

// sizingKind enumerates the ways a column width can be resolved.
type sizingKind int

const (
	sizingAuto sizingKind = iota
	sizingFixed
	sizingPercent
	sizingFlex
)

// Sizing describes how a column's width is resolved. Its zero value is [Auto].
// Use one of the Auto, Fixed, Percent or Flex constructors to build one.
type Sizing struct {
	kind  sizingKind
	value float64
}

// Auto sizes a column to fit its widest value, including its header. This is
// the zero value of [Sizing].
func Auto() Sizing { return Sizing{kind: sizingAuto} }

// Fixed sizes a column to exactly n cells.
func Fixed(n int) Sizing { return Sizing{kind: sizingFixed, value: float64(max(0, n))} }

// Percent sizes a column to the given fraction (0..1) of the total width
// available to column content, that is, of the table width minus all cell
// padding. Fixed and Auto columns are not subtracted first, so an over-eager
// set of percentages can overflow and will be shrunk to fit.
func Percent(p float64) Sizing { return Sizing{kind: sizingPercent, value: clamp(p, 0, 1)} }

// Flex gives a column a share of whatever space is left after the Fixed, Auto
// and Percent columns have been resolved, proportional to its weight. A weight
// below 1 is treated as 1.
func Flex(weight int) Sizing { return Sizing{kind: sizingFlex, value: float64(max(1, weight))} }

// String implements [fmt.Stringer].
func (s Sizing) String() string {
	switch s.kind {
	case sizingFixed:
		return fmt.Sprintf("Fixed(%d)", int(s.value))
	case sizingPercent:
		return fmt.Sprintf("Percent(%.2f)", s.value)
	case sizingFlex:
		return fmt.Sprintf("Flex(%d)", int(s.value))
	default:
		return "Auto"
	}
}

// KeyMap defines keybindings. It satisfies to the help.KeyMap interface, which
// is used to render the help menu.
type KeyMap struct {
	RowUp         key.Binding
	RowDown       key.Binding
	ColumnLeft    key.Binding
	ColumnRight   key.Binding
	PageUp        key.Binding
	PageDown      key.Binding
	HalfPageUp    key.Binding
	HalfPageDown  key.Binding
	GotoTop       key.Binding
	GotoBottom    key.Binding
	SetNormalMode key.Binding
}

// ShortHelp implements the KeyMap interface.
func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.RowUp, km.RowDown, km.ColumnLeft, km.ColumnRight, km.SetNormalMode}
}

// FullHelp implements the KeyMap interface.
func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.RowUp, km.RowDown, km.ColumnLeft, km.ColumnRight},
		{km.PageUp, km.PageDown, km.HalfPageUp, km.HalfPageDown},
		{km.GotoTop, km.GotoBottom, km.SetNormalMode},
	}
}

// StyleFunc is a function that can be used to customize the style of a table
// cell based on the current render context.
type StyleFunc func(ctx RenderContext) lipgloss.Style

// RenderContext contains the model's current state, useful for cell styling.
type RenderContext struct {
	// Cursor is the current [row, column] position of the table selection.
	Cursor [2]int
	// Cell is the [row, column] position of the cell being rendered.
	Cell [2]int
	// Value is the string content of the cell.
	Value string
	// Mode indicates the current table mode.
	Mode Mode
	// IsFocused indicates whether the table is currently focused.
	IsFocused bool
}

// DefaultKeyMap returns a default set of keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		RowUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		RowDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		ColumnLeft: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/←", "left"),
		),
		ColumnRight: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("l/→", "right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("b", "pgup"),
			key.WithHelp("b/pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("f", "pgdown", "space"),
			key.WithHelp("f/pgdn", "page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("u", "ctrl+u"),
			key.WithHelp("u", "½ page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("d", "ctrl+d"),
			key.WithHelp("d", "½ page down"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g/home", "go to start"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G/end", "go to end"),
		),
		SetNormalMode: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "return to normal mode"),
		),
	}
}

// Styles contains style definitions for this list component. By default, these
// values are generated by DefaultStyles.
//
// Header and Cell should use the same horizontal padding, since the layout
// budget is computed from Cell and columns would otherwise not line up.
type Styles struct {
	Header   lipgloss.Style
	Cell     lipgloss.Style
	Selected lipgloss.Style
}

// DefaultStyles returns a set of default style definitions for this table.
func DefaultStyles() Styles {
	return Styles{
		Selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		Header:   lipgloss.NewStyle().Bold(true).Padding(0, 1),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
	}
}

// SetStyles sets the table styles.
func (m *Model) SetStyles(s Styles) {
	m.styles = s
	m.layoutDirty = true // cell padding is part of the layout budget
	m.UpdateViewport()
}

// Option is used to set options in New. For example:
//
//	table := New(WithColumns([]Column{{Title: "ID", Sizing: Fixed(10)}}))
type Option func(*Model)

// New creates a new model for the table widget.
func New(opts ...Option) Model {
	m := Model{
		viewport: viewport.New(viewport.WithHeight(20)), //nolint:mnd

		KeyMap:     DefaultKeyMap(),
		Help:       help.New(),
		styles:     DefaultStyles(),
		showHeader: true,
	}

	for _, opt := range opts {
		opt(&m)
	}

	if m.mode == ModeFixedColumn && len(m.cols) > 0 {
		m.cursor[1] = clamp(m.fixedColumn, 0, len(m.cols)-1)
	}

	// Options may be applied in any order, so resolve the layout only once
	// they've all run.
	m.layoutDirty = true
	m.UpdateViewport()

	return m
}

// WithColumns sets the table columns (headers).
func WithColumns(cols []Column) Option {
	return func(m *Model) {
		m.cols = cols
	}
}

// WithRows sets the table rows (data).
func WithRows(rows []Row) Option {
	return func(m *Model) {
		m.rows = rows
	}
}

// WithHeight sets the height of the table.
func WithHeight(h int) Option {
	return func(m *Model) {
		m.viewport.SetHeight(h - m.headerHeight())
	}
}

// WithWidth sets the width of the table.
func WithWidth(w int) Option {
	return func(m *Model) {
		m.viewport.SetWidth(w)
	}
}

// WithFocused sets the focus state of the table.
func WithFocused(f bool) Option {
	return func(m *Model) {
		m.focused = f
	}
}

// WithStyles sets the table styles.
func WithStyles(s Styles) Option {
	return func(m *Model) {
		m.styles = s
	}
}

// WithStyleFunc sets the table style func which can be determined cell style
// per column, row, and selected state.
func WithStyleFunc(f StyleFunc) Option {
	return func(m *Model) {
		m.styleFunc = f
	}
}

// WithMode sets the initial table mode.
func WithMode(md Mode) Option {
	return func(m *Model) {
		m.mode = md
	}
}

// WithFixedColumn sets the column used by ModeFixedColumn.
func WithFixedColumn(col int) Option {
	return func(m *Model) {
		m.fixedColumn = col
	}
}

// WithHeader controls whether the header row is rendered.
func WithHeader(show bool) Option {
	return func(m *Model) {
		m.showHeader = show
	}
}

// WithKeyMap sets the key map.
func WithKeyMap(km KeyMap) Option {
	return func(m *Model) {
		m.KeyMap = km
	}
}

// Init implements the [tea.Model] interface.
func (_ *Model) Init() tea.Cmd { return nil }

// Update is the Bubble Tea update loop. The table lays itself out from the
// width it was assigned via [Model.SetWidth], not from the terminal width, so
// window size messages need no handling here.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.KeyMap.RowUp):
			m.MoveUp(1)
		case key.Matches(msg, m.KeyMap.RowDown):
			m.MoveDown(1)
		case key.Matches(msg, m.KeyMap.ColumnLeft):
			m.MoveLeft(1)
		case key.Matches(msg, m.KeyMap.ColumnRight):
			m.MoveRight(1)
		case key.Matches(msg, m.KeyMap.PageUp):
			m.MoveUp(m.viewport.Height())
		case key.Matches(msg, m.KeyMap.PageDown):
			m.MoveDown(m.viewport.Height())
		case key.Matches(msg, m.KeyMap.HalfPageUp):
			m.MoveUp(m.viewport.Height() / 2) //nolint:mnd
		case key.Matches(msg, m.KeyMap.HalfPageDown):
			m.MoveDown(m.viewport.Height() / 2) //nolint:mnd
		case key.Matches(msg, m.KeyMap.GotoTop):
			m.GotoTop()
		case key.Matches(msg, m.KeyMap.GotoBottom):
			m.GotoBottom()
		case key.Matches(msg, m.KeyMap.SetNormalMode):
			m.SetMode(ModeNormal)
		}
	}

	return m, nil
}

// Focused returns the focus state of the table.
func (m Model) Focused() bool {
	return m.focused
}

// Focus focuses the table, allowing the user to move around the rows and
// interact.
func (m *Model) Focus() {
	m.focused = true
	m.UpdateViewport()
}

// Blur blurs the table, preventing selection or movement.
func (m *Model) Blur() {
	m.focused = false
	m.UpdateViewport()
}

// View renders the component.
func (m *Model) View() tea.View {
	content := m.viewport.View().Content
	if m.showHeader {
		content = m.headersView() + "\n" + content
	}
	return tea.NewView(content)
}

// ShortHelp implements the KeyMap interface.
func (m *Model) ShortHelp() []key.Binding {
	return m.KeyMap.ShortHelp()
}

// FullHelp implements the KeyMap interface.
func (m *Model) FullHelp() [][]key.Binding {
	return m.KeyMap.FullHelp()
}

// HelpView is a helper method for rendering the help menu from the keymap.
// Note that this view is not rendered by default and you must call it
// manually in your application, where applicable.
func (m Model) HelpView() string {
	return m.Help.View(m.KeyMap)
}

// UpdateViewport updates the list content based on the previously defined
// columns and rows.
func (m *Model) UpdateViewport() {
	m.ensureLayout()

	renderedRows := make([]string, 0, len(m.rows))

	// Render only rows from: m.cursor-m.viewport.Height to: m.cursor+m.viewport.Height
	// Constant runtime, independent of number of rows in a table.
	// Limits the number of renderedRows to a maximum of 2*m.viewport.Height
	if m.cursor[0] >= 0 {
		m.start = clamp(m.cursor[0]-m.viewport.Height(), 0, m.cursor[0])
	} else {
		m.start = 0
	}
	m.end = clamp(m.cursor[0]+m.viewport.Height(), m.cursor[0], len(m.rows))
	for i := m.start; i < m.end; i++ {
		renderedRows = append(renderedRows, m.renderRow(i))
	}

	m.viewport.SetContent(
		lipgloss.JoinVertical(lipgloss.Left, renderedRows...),
	)
}

// SelectedRow returns the selected row.
// You can cast it to your own implementation.
func (m Model) SelectedRow() Row {
	if m.cursor[0] < 0 || m.cursor[0] >= len(m.rows) {
		return nil
	}

	return m.rows[m.cursor[0]]
}

// SelectedCell returns the selected cell value.
func (m Model) SelectedCell() string {
	row := m.SelectedRow()
	if row == nil || m.cursor[1] < 0 || m.cursor[1] >= len(row) {
		return ""
	}

	return row[m.cursor[1]]
}

// Rows returns the current rows.
func (m Model) Rows() []Row {
	return m.rows
}

// Columns returns the current columns.
func (m Model) Columns() []Column {
	return m.cols
}

// ColumnWidths returns the resolved width of each column, in cells, excluding
// cell padding.
func (m *Model) ColumnWidths() []int {
	m.ensureLayout()
	widths := make([]int, len(m.colWidths))
	copy(widths, m.colWidths)
	return widths
}

// ContentWidth returns the total width available to column content, that is,
// the table width minus the viewport frame and all cell padding. It is the
// budget that [Percent] fractions are taken from, and the sum of
// [Model.ColumnWidths] whenever the columns can be made to fill the table.
func (m *Model) ContentWidth() int {
	frame := m.viewport.Style.GetHorizontalFrameSize() +
		len(m.cols)*m.styles.Cell.GetHorizontalFrameSize()
	return max(0, m.viewport.Width()-frame)
}

// SetRow sets a single row at index i.
func (m *Model) SetRow(i int, r Row) error {
	if i < 0 || i > len(m.rows)-1 {
		return fmt.Errorf("index %d is out of bounds", i)
	}
	m.rows[i] = r
	m.layoutDirty = true // Auto columns are measured from the content
	m.UpdateViewport()
	return nil
}

// SetRows sets a new rows state.
func (m *Model) SetRows(r []Row) {
	m.rows = r

	if m.cursor[0] > len(m.rows)-1 {
		m.cursor[0] = len(m.rows) - 1
	}

	m.layoutDirty = true
	m.UpdateViewport()
}

// SetColumns sets a new columns state.
func (m *Model) SetColumns(c []Column) {
	m.cols = c
	if m.mode == ModeFixedColumn && len(m.cols) > 0 {
		m.cursor[1] = clamp(m.fixedColumn, 0, len(m.cols)-1)
	}
	m.layoutDirty = true
	m.UpdateViewport()
}

// SetWidth sets the width of the viewport of the table.
func (m *Model) SetWidth(w int) {
	m.viewport.SetWidth(w)
	m.layoutDirty = true
	m.UpdateViewport()
}

// SetHeight sets the height of the viewport of the table.
func (m *Model) SetHeight(h int) {
	m.viewport.SetHeight(h - m.headerHeight())
	m.UpdateViewport()
}

// Height returns the viewport height of the table.
func (m Model) Height() int {
	return m.viewport.Height()
}

// Width returns the viewport width of the table.
func (m Model) Width() int {
	return m.viewport.Width()
}

// Cursor returns the indices of the selected row and column [row, col].
func (m Model) Cursor() [2]int {
	return m.cursor
}

// SetCursor sets the cursor position in the table. If a value is negative, that
// position won't be updated.
func (m *Model) SetCursor(row, col int) {
	prev := m.cursor
	if row >= 0 {
		m.cursor[0] = clamp(row, 0, len(m.rows)-1)
	}
	if col >= 0 {
		m.cursor[1] = clamp(col, 0, len(m.cols)-1)
	}
	if prev != m.cursor {
		m.UpdateViewport()
	}
}

// MoveUp moves the selection up by n number of rows.
// It can not go above the first row.
func (m *Model) MoveUp(n int) {
	m.cursor[0] = clamp(m.cursor[0]-n, 0, len(m.rows)-1)

	offset := m.viewport.YOffset()
	switch {
	case m.start == 0:
		offset = clamp(offset, 0, m.cursor[0])
	case m.start < m.viewport.Height():
		offset = clamp(clamp(offset+n, 0, m.cursor[0]), 0, m.viewport.Height())
	case offset >= 1:
		offset = clamp(offset+n, 1, m.viewport.Height())
	}
	m.viewport.SetYOffset(offset)
	m.UpdateViewport()
}

// MoveDown moves the selection down by n number of rows.
// It can not go below the last row.
func (m *Model) MoveDown(n int) {
	m.cursor[0] = clamp(m.cursor[0]+n, 0, len(m.rows)-1)

	offset := m.viewport.YOffset()
	switch {
	case m.end == len(m.rows) && offset > 0:
		offset = clamp(offset-n, 1, m.viewport.Height())
	case m.cursor[0] > (m.end-m.start)/2 && offset > 0:
		offset = clamp(offset-n, 1, m.cursor[0])
	case offset > 1:
	case m.cursor[0] > offset+m.viewport.Height()-1:
		offset = clamp(offset+1, 0, 1)
	}
	m.viewport.SetYOffset(offset)
	m.UpdateViewport()
}

// MoveLeft sets the table to cell selection mode and moves the selection left
// by n number of columns. It can not go past the first column.
func (m *Model) MoveLeft(n int) {
	if m.mode == ModeFixedColumn {
		return
	}
	m.mode = ModeCell
	m.cursor[1] = clamp(m.cursor[1]-n, 0, len(m.cols)-1)
	m.UpdateViewport()
}

// MoveRight sets the table to cell selection mode and moves the selection right
// by n number of columns. It can not go past the last column.
func (m *Model) MoveRight(n int) {
	if m.mode == ModeFixedColumn {
		return
	}
	m.mode = ModeCell
	m.cursor[1] = clamp(m.cursor[1]+n, 0, len(m.cols)-1)
	m.UpdateViewport()
}

// GotoTop moves the selection to the first row.
func (m *Model) GotoTop() {
	m.MoveUp(m.cursor[0])
}

// GotoBottom moves the selection to the last row.
func (m *Model) GotoBottom() {
	m.MoveDown(len(m.rows))
}

// SetMode sets the table mode.
func (m *Model) SetMode(md Mode) {
	m.mode = md
	switch md {
	case ModeNormal:
		m.cursor[1] = 0 // reset col
	case ModeFixedColumn:
		if len(m.cols) > 0 {
			m.cursor[1] = clamp(m.fixedColumn, 0, len(m.cols)-1)
		}
	}
	m.UpdateViewport()
}

// Mode returns the current table mode.
func (m Model) Mode() Mode {
	return m.mode
}

// SetFixedColumn sets the column used by ModeFixedColumn.
func (m *Model) SetFixedColumn(col int) {
	if len(m.cols) == 0 {
		m.fixedColumn = 0
		return
	}
	m.fixedColumn = clamp(col, 0, len(m.cols)-1)
	if m.mode == ModeFixedColumn {
		m.cursor[1] = m.fixedColumn
	}
	m.UpdateViewport()
}

// FixedColumn returns the currently fixed column.
func (m Model) FixedColumn() int {
	return m.fixedColumn
}

// SetHeader controls whether the header row is rendered.
func (m *Model) SetHeader(show bool) {
	if m.showHeader == show {
		return
	}
	oldHeaderHeight := m.headerHeight()
	m.showHeader = show
	m.layoutDirty = true // titles count towards the width of Auto columns
	// Re-apply height so the viewport fills the intended space.
	m.viewport.SetHeight(m.viewport.Height() + oldHeaderHeight - m.headerHeight())
	m.UpdateViewport()
}

// FromValues create the table rows from a simple string. It uses `\n` by
// default for getting all the rows and the given separator for the fields on
// each row.
func (m *Model) FromValues(value, separator string) {
	rows := []Row{} //nolint:prealloc
	for _, line := range strings.Split(value, "\n") {
		r := Row{}
		for _, field := range strings.Split(line, separator) {
			r = append(r, field)
		}
		rows = append(rows, r)
	}

	m.SetRows(rows)
}

func (m *Model) headersView() string {
	m.ensureLayout()
	s := make([]string, 0, len(m.cols))
	for c, col := range m.cols {
		w := m.colWidths[c]
		if w <= 0 {
			continue
		}
		style := lipgloss.NewStyle().Width(w).MaxWidth(w).Inline(true)
		renderedCell := style.Render(ansi.Truncate(col.Title, w, "…"))
		s = append(s, m.styles.Header.Render(renderedCell))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, s...)
}

// headerHeight returns the rendered height of the header, or 0 when hidden.
func (m *Model) headerHeight() int {
	if !m.showHeader {
		return 0
	}
	return lipgloss.Height(m.headersView())
}

func (m *Model) renderRow(r int) string {
	row := m.rows[r]
	ctx := m.newRenderContext()
	s := make([]string, 0, len(m.cols))
	for c := range m.cols {
		w := m.colWidths[c]
		if w <= 0 {
			continue
		}

		// Rows are not required to have a value for every column.
		var value string
		if c < len(row) {
			value = row[c]
		}

		ctx.Cell = [2]int{r, c}
		ctx.Value = value

		cellStyle := m.styles.Cell
		if m.styleFunc != nil {
			cellStyle = m.styleFunc(ctx)
		}

		if r == m.cursor[0] && c == m.cursor[1] && (m.mode == ModeCell || m.mode == ModeFixedColumn) {
			cellStyle = cellStyle.Inherit(m.styles.Selected)
		}

		style := lipgloss.NewStyle().Width(w).MaxWidth(w).Inline(true)
		renderedCell := cellStyle.Render(style.Render(ansi.Truncate(value, w, "…")))
		s = append(s, renderedCell)
	}

	rendered := lipgloss.JoinHorizontal(lipgloss.Top, s...)

	if r == m.cursor[0] && m.mode == ModeNormal {
		return m.styles.Selected.Render(rendered)
	}

	return rendered
}

// ensureLayout resolves the column widths if anything they depend on has
// changed since the last time.
func (m *Model) ensureLayout() {
	if !m.layoutDirty && len(m.colWidths) == len(m.cols) {
		return
	}
	m.resolveColumnWidths()
	m.layoutDirty = false
}

// sizing returns the sizing spec for column i. It exists as a single funnel so
// that an override mechanism can be added later without touching the solver.
func (m *Model) sizing(i int) Sizing {
	return m.cols[i].Sizing
}

// resolveColumnWidths computes the width of every column from the space
// available to the table. It must not call UpdateViewport, to avoid recursion.
func (m *Model) resolveColumnWidths() {
	budget := m.ContentWidth()
	widths := make([]int, len(m.cols))

	// Pass 1: resolve everything that doesn't depend on leftover space.
	var (
		flex []int
		used int
	)
	for i := range m.cols {
		s := m.sizing(i)
		switch s.kind {
		case sizingFixed:
			widths[i] = int(s.value)
		case sizingPercent:
			widths[i] = int(s.value * float64(budget))
		case sizingFlex:
			flex = append(flex, i)
			continue // resolved in pass 2
		default: // Auto
			widths[i] = m.measureColumn(i)
		}
		widths[i] = clampWidth(widths[i], m.cols[i])
		used += widths[i]
	}

	// Pass 2: hand what's left to the flex columns.
	if len(flex) > 0 {
		m.distributeFlex(widths, flex, max(0, budget-used))
		for _, i := range flex {
			used += widths[i]
		}
	}

	// Pass 3: the columns may still ask for more than the table has, either
	// because the percentages oversubscribe it or because minimums do.
	if used > budget {
		shrinkToFit(widths, m.cols, budget)
	}

	m.colWidths = widths
}

// measureColumn returns the display width of the widest value in a column,
// including its title when the header is shown.
func (m *Model) measureColumn(i int) int {
	w := 0
	if m.showHeader {
		w = ansi.StringWidth(m.cols[i].Title)
	}
	for _, row := range m.rows {
		if i < len(row) {
			w = max(w, ansi.StringWidth(row[i]))
		}
	}
	return w
}

// distributeFlex splits space between the given columns in proportion to their
// weights. Columns pinned by their Min/MaxWidth are settled and dropped from
// the pool, and the remaining space is shared out again, so their surplus or
// deficit is absorbed by the columns that can still move. Each round shrinks
// the pool, so this terminates.
func (m *Model) distributeFlex(widths []int, flex []int, space int) {
	pool := flex
	for len(pool) > 0 {
		var totalWeight float64
		for _, i := range pool {
			totalWeight += m.sizing(i).value
		}

		var (
			unclamped []int
			taken     int
		)
		for _, i := range pool {
			want := 0
			if space > 0 && totalWeight > 0 {
				want = int(float64(space) * m.sizing(i).value / totalWeight)
			}
			widths[i] = clampWidth(want, m.cols[i])
			if widths[i] == want {
				unclamped = append(unclamped, i)
			} else {
				taken += widths[i]
			}
		}

		if len(unclamped) == len(pool) {
			// Nothing hit a limit. Give the rounding remainder to the last
			// column so the table fills its budget exactly.
			total := 0
			for _, i := range pool {
				total += widths[i]
			}
			if last := pool[len(pool)-1]; space > total {
				widths[last] = clampWidth(widths[last]+space-total, m.cols[last])
			}
			return
		}

		space -= taken
		pool = unclamped
	}
}

// shrinkToFit reduces columns that have slack above their MinWidth, in
// proportion to that slack, until the total fits the budget. If the minimums
// alone overflow the table, the widths are left as they are and the trailing
// cells are clipped by the viewport.
func shrinkToFit(widths []int, cols []Column, budget int) {
	total := 0
	for _, w := range widths {
		total += w
	}
	excess := total - budget
	if excess <= 0 {
		return
	}

	slack := make([]int, len(widths))
	totalSlack := 0
	for i := range widths {
		slack[i] = max(0, widths[i]-cols[i].MinWidth)
		totalSlack += slack[i]
	}
	if totalSlack == 0 {
		return
	}

	take := min(excess, totalSlack)
	removed := 0
	for i := range widths {
		if slack[i] == 0 {
			continue
		}
		d := slack[i] * take / totalSlack
		widths[i] -= d
		slack[i] -= d
		removed += d
	}
	// Truncation above leaves a few cells unaccounted for; reclaim them one at
	// a time from whatever slack remains.
	for i := 0; removed < take; i = (i + 1) % len(widths) {
		if slack[i] > 0 {
			widths[i]--
			slack[i]--
			removed++
		}
	}
}

// clampWidth applies a column's bounds to a width. MaxWidth is applied last, so
// it wins over a MinWidth that exceeds it.
func clampWidth(w int, col Column) int {
	w = max(0, w)
	if col.MinWidth > 0 {
		w = max(w, col.MinWidth)
	}
	if col.MaxWidth > 0 {
		w = min(w, col.MaxWidth)
	}
	return w
}

func (m *Model) newRenderContext() RenderContext {
	return RenderContext{
		Cursor:    m.cursor,
		Mode:      m.mode,
		IsFocused: m.focused,
	}
}

func clamp[T cmp.Ordered](v, low, high T) T {
	if low > high {
		low, high = high, low
	}
	return min(high, max(low, v))
}
