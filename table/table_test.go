package table

import (
	"reflect"
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"
)

var testCols = []Column{
	{Title: "col1", Sizing: Fixed(10)},
	{Title: "col2", Sizing: Fixed(10)},
	{Title: "col3", Sizing: Fixed(10)},
}

// unpaddedStyles removes the cell padding so that the layout budget equals the
// table width, which keeps the expected column widths easy to reason about.
func unpaddedStyles() Styles {
	s := DefaultStyles()
	s.Header = s.Header.Padding(0)
	s.Cell = s.Cell.Padding(0)
	return s
}

func TestNew(t *testing.T) {
	tests := map[string]struct {
		opts []Option
		want Model
	}{
		"Default": {
			want: Model{
				// Default fields
				viewport: viewport.New(
					viewport.WithWidth(0),
					viewport.WithHeight(20),
				),
				KeyMap:     DefaultKeyMap(),
				Help:       help.New(),
				styles:     DefaultStyles(),
				showHeader: true,
			},
		},
		"WithColumns": {
			opts: []Option{
				WithColumns([]Column{
					{Title: "Foo", Sizing: Fixed(1)},
					{Title: "Bar", Sizing: Fixed(2)},
				}),
			},
			want: Model{
				// Default fields
				viewport: viewport.New(
					viewport.WithWidth(0),
					viewport.WithHeight(20),
				),
				KeyMap:     DefaultKeyMap(),
				Help:       help.New(),
				styles:     DefaultStyles(),
				showHeader: true,

				// Modified fields
				cols: []Column{
					{Title: "Foo", Sizing: Fixed(1)},
					{Title: "Bar", Sizing: Fixed(2)},
				},
			},
		},
		"WithColumns; WithRows": {
			opts: []Option{
				WithColumns([]Column{
					{Title: "Foo", Sizing: Fixed(1)},
					{Title: "Bar", Sizing: Fixed(2)},
				}),
				WithRows([]Row{
					{"1", "Foo"},
					{"2", "Bar"},
				}),
			},
			want: Model{
				// Default fields
				viewport: viewport.New(
					viewport.WithWidth(0),
					viewport.WithHeight(20),
				),
				KeyMap:     DefaultKeyMap(),
				Help:       help.New(),
				styles:     DefaultStyles(),
				showHeader: true,

				// Modified fields
				cols: []Column{
					{Title: "Foo", Sizing: Fixed(1)},
					{Title: "Bar", Sizing: Fixed(2)},
				},
				rows: []Row{
					{"1", "Foo"},
					{"2", "Bar"},
				},
			},
		},
		"WithHeight": {
			opts: []Option{
				WithHeight(10),
			},
			want: Model{
				// Default fields
				KeyMap:     DefaultKeyMap(),
				Help:       help.New(),
				styles:     DefaultStyles(),
				showHeader: true,

				// Modified fields
				viewport: viewport.New(
					viewport.WithWidth(0),
					viewport.WithHeight(9),
				),
			},
		},
		"WithWidth": {
			opts: []Option{
				WithWidth(10),
			},
			want: Model{
				// Default fields
				KeyMap:     DefaultKeyMap(),
				Help:       help.New(),
				styles:     DefaultStyles(),
				showHeader: true,

				// Modified fields
				viewport: viewport.New(
					viewport.WithWidth(10),
					viewport.WithHeight(20),
				),
			},
		},
		"WithFocused": {
			opts: []Option{
				WithFocused(true),
			},
			want: Model{
				// Default fields
				viewport: viewport.New(
					viewport.WithWidth(0),
					viewport.WithHeight(20),
				),
				KeyMap:     DefaultKeyMap(),
				Help:       help.New(),
				styles:     DefaultStyles(),
				showHeader: true,

				// Modified fields
				focused: true,
			},
		},
		"WithStyles": {
			opts: []Option{
				WithStyles(Styles{}),
			},
			want: Model{
				// Default fields
				viewport: viewport.New(
					viewport.WithWidth(0),
					viewport.WithHeight(20),
				),
				KeyMap:     DefaultKeyMap(),
				Help:       help.New(),
				showHeader: true,

				// Modified fields
				styles: Styles{},
			},
		},
		"WithKeyMap": {
			opts: []Option{
				WithKeyMap(KeyMap{}),
			},
			want: Model{
				// Default fields
				viewport: viewport.New(
					viewport.WithWidth(0),
					viewport.WithHeight(20),
				),
				Help:       help.New(),
				styles:     DefaultStyles(),
				showHeader: true,

				// Modified fields
				KeyMap: KeyMap{},
			},
		},
		"WithHeader": {
			opts: []Option{
				WithHeader(false),
			},
			want: Model{
				// Default fields
				viewport: viewport.New(
					viewport.WithWidth(0),
					viewport.WithHeight(20),
				),
				KeyMap:     DefaultKeyMap(),
				Help:       help.New(),
				styles:     DefaultStyles(),
				showHeader: false,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// New resolves the layout once all options have been applied, so
			// the expected model has to do the same.
			tc.want.layoutDirty = true
			tc.want.UpdateViewport()

			got := New(tc.opts...)

			// NOTE(@andreynering): Funcs have different references, so we need
			// to clear them out to compare the structs.
			tc.want.viewport.LeftGutterFunc = nil
			got.viewport.LeftGutterFunc = nil

			if !reflect.DeepEqual(tc.want, got) {
				t.Errorf("\n\nwant %v\n\ngot %v", tc.want, got)
			}
		})
	}
}

func TestSizing_String(t *testing.T) {
	tests := []struct {
		sizing Sizing
		want   string
	}{
		{Sizing{}, "Auto"},
		{Auto(), "Auto"},
		{Fixed(10), "Fixed(10)"},
		{Fixed(-5), "Fixed(0)"},
		{Percent(0.25), "Percent(0.25)"},
		{Percent(1.5), "Percent(1.00)"},
		{Percent(-0.5), "Percent(0.00)"},
		{Flex(3), "Flex(3)"},
		{Flex(0), "Flex(1)"},
	}

	for _, tc := range tests {
		if got := tc.sizing.String(); got != tc.want {
			t.Errorf("want %q, got %q", tc.want, got)
		}
	}
}

func TestModel_ColumnWidths(t *testing.T) {
	tests := map[string]struct {
		opts []Option
		want []int
	}{
		"no columns": {
			opts: []Option{WithWidth(100)},
			want: []int{},
		},
		"fixed": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Fixed(10)},
					{Title: "b", Sizing: Fixed(20)},
				}),
			},
			want: []int{10, 20},
		},
		"fixed wider than the table": {
			opts: []Option{
				WithWidth(20),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Fixed(30)},
					{Title: "b", Sizing: Fixed(10)},
				}),
			},
			want: []int{15, 5},
		},
		"percent": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Percent(0.25)},
					{Title: "b", Sizing: Percent(0.5)},
				}),
			},
			want: []int{25, 50},
		},
		"percent is taken from the content width": {
			opts: []Option{
				WithWidth(22), // 22 - 2 cells of padding = 20
				WithColumns([]Column{
					{Title: "a", Sizing: Percent(0.5)},
				}),
			},
			want: []int{10},
		},
		"percent oversubscribed": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Percent(0.8)},
					{Title: "b", Sizing: Percent(0.8)},
				}),
			},
			want: []int{50, 50},
		},
		"auto includes the header": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{{Title: "Identifier"}}),
				WithRows([]Row{{"1"}, {"22"}}),
			},
			want: []int{10},
		},
		"auto without a header": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithHeader(false),
				WithColumns([]Column{{Title: "Identifier"}}),
				WithRows([]Row{{"1"}, {"22"}}),
			},
			want: []int{2},
		},
		"auto measures the widest cell": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "ID"},
					{Title: "Name"},
				}),
				WithRows([]Row{
					{"1", "Chocolate"},
					{"22", "Tim Tams"},
				}),
			},
			want: []int{2, 9},
		},
		"flex equal weights": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Flex(1)},
					{Title: "b", Sizing: Flex(1)},
				}),
			},
			want: []int{50, 50},
		},
		"flex weighted": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Flex(1)},
					{Title: "b", Sizing: Flex(3)},
				}),
			},
			want: []int{25, 75},
		},
		"flex remainder goes to the last column": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Flex(1)},
					{Title: "b", Sizing: Flex(1)},
					{Title: "c", Sizing: Flex(1)},
				}),
			},
			want: []int{33, 33, 34},
		},
		"flex takes what fixed and auto leave": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Fixed(10)},
					{Title: "ab"},
					{Title: "c", Sizing: Flex(1)},
				}),
			},
			want: []int{10, 2, 88},
		},
		"flex max width is redistributed": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Flex(1), MaxWidth: 20},
					{Title: "b", Sizing: Flex(1)},
				}),
			},
			want: []int{20, 80},
		},
		"flex min width is taken from the other flex columns": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Flex(1), MinWidth: 60},
					{Title: "b", Sizing: Flex(1)},
				}),
			},
			want: []int{60, 40},
		},
		"min width on auto": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{{Title: "ab", MinWidth: 15}}),
			},
			want: []int{15},
		},
		"max width beats min width": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Fixed(50), MinWidth: 30, MaxWidth: 10},
				}),
			},
			want: []int{10},
		},
		"shrinking respects min width": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Fixed(80), MinWidth: 80},
					{Title: "b", Sizing: Fixed(40)},
				}),
			},
			want: []int{80, 20},
		},
		"zero width table": {
			opts: []Option{
				WithWidth(0),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Fixed(10)},
					{Title: "b", Sizing: Fixed(10)},
				}),
			},
			want: []int{0, 0},
		},
		"mixed sizing": {
			opts: []Option{
				WithWidth(100),
				WithStyles(unpaddedStyles()),
				WithColumns([]Column{
					{Title: "a", Sizing: Fixed(10)},
					{Title: "b", Sizing: Percent(0.2)},
					{Title: "Name"},
					{Title: "d", Sizing: Flex(1)},
				}),
				WithRows([]Row{
					{"x", "y", "Chocolate Digestives", "z"},
				}),
			},
			want: []int{10, 20, 20, 50},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			table := New(tc.opts...)

			got := table.ColumnWidths()
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("\n\nwant %v\n\ngot %v", tc.want, got)
			}
		})
	}
}

func TestModel_ColumnWidths_ReturnsCopy(t *testing.T) {
	table := New(
		WithWidth(50),
		WithColumns([]Column{{Title: "a", Sizing: Fixed(10)}}),
	)

	widths := table.ColumnWidths()
	widths[0] = 99

	if got := table.ColumnWidths()[0]; got != 10 {
		t.Fatalf("want 10, got %d", got)
	}
}

func TestModel_ContentWidth(t *testing.T) {
	tests := map[string]struct {
		table Model
		want  int
	}{
		"default padding": {
			table: New(WithWidth(59), WithColumns(testCols)),
			want:  53, // 59 - 3 columns of 2 cells of padding
		},
		"no padding": {
			table: New(WithWidth(59), WithColumns(testCols), WithStyles(unpaddedStyles())),
			want:  59,
		},
		"padding exceeds the width": {
			table: New(WithWidth(2), WithColumns(testCols)),
			want:  0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tc.table.ContentWidth(); got != tc.want {
				t.Errorf("want %d, got %d", tc.want, got)
			}
		})
	}
}

// TestModel_LayoutRecompute checks that the resolved widths are invalidated by
// everything they depend on.
func TestModel_LayoutRecompute(t *testing.T) {
	t.Run("SetWidth", func(t *testing.T) {
		table := New(
			WithWidth(50),
			WithStyles(unpaddedStyles()),
			WithColumns([]Column{
				{Title: "a", Sizing: Flex(1)},
				{Title: "b", Sizing: Flex(1)},
			}),
		)
		assertWidths(t, &table, []int{25, 25})

		table.SetWidth(30)
		assertWidths(t, &table, []int{15, 15})
	})

	t.Run("SetRows", func(t *testing.T) {
		table := New(
			WithWidth(100),
			WithStyles(unpaddedStyles()),
			WithHeader(false),
			WithColumns([]Column{{Title: "a"}}),
			WithRows([]Row{{"a"}}),
		)
		assertWidths(t, &table, []int{1})

		table.SetRows([]Row{{"abcdef"}})
		assertWidths(t, &table, []int{6})
	})

	t.Run("SetRow", func(t *testing.T) {
		table := New(
			WithWidth(100),
			WithStyles(unpaddedStyles()),
			WithHeader(false),
			WithColumns([]Column{{Title: "a"}}),
			WithRows([]Row{{"a"}, {"b"}}),
		)
		assertWidths(t, &table, []int{1})

		if err := table.SetRow(1, Row{"abcdef"}); err != nil {
			t.Fatalf("got unexpected error %q", err)
		}
		assertWidths(t, &table, []int{6})
	})

	t.Run("SetColumns", func(t *testing.T) {
		table := New(
			WithWidth(100),
			WithStyles(unpaddedStyles()),
			WithColumns([]Column{{Title: "a", Sizing: Fixed(10)}}),
		)
		assertWidths(t, &table, []int{10})

		table.SetColumns([]Column{
			{Title: "a", Sizing: Fixed(20)},
			{Title: "b", Sizing: Fixed(5)},
		})
		assertWidths(t, &table, []int{20, 5})
	})

	t.Run("SetStyles", func(t *testing.T) {
		table := New(
			WithWidth(20),
			WithStyles(unpaddedStyles()),
			WithColumns([]Column{{Title: "a", Sizing: Percent(1)}}),
		)
		assertWidths(t, &table, []int{20})

		table.SetStyles(DefaultStyles())
		assertWidths(t, &table, []int{18})
	})

	t.Run("SetHeader", func(t *testing.T) {
		table := New(
			WithWidth(100),
			WithStyles(unpaddedStyles()),
			WithColumns([]Column{{Title: "Identifier"}}),
			WithRows([]Row{{"a"}}),
		)
		assertWidths(t, &table, []int{10})

		table.SetHeader(false)
		assertWidths(t, &table, []int{1})
	})
}

func assertWidths(t *testing.T, m *Model, want []int) {
	t.Helper()
	if got := m.ColumnWidths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("\n\nwant %v\n\ngot %v", want, got)
	}
}

// TestModel_FlexFillsWidth checks that flex columns consume the space left
// over by the other columns exactly, padding included.
func TestModel_FlexFillsWidth(t *testing.T) {
	const width = 60

	table := New(
		WithWidth(width),
		WithColumns([]Column{
			{Title: "A", Sizing: Flex(1)},
			{Title: "B", Sizing: Flex(2)},
			{Title: "C", Sizing: Fixed(10)},
		}),
		WithRows([]Row{{"a", "b", "c"}}),
	)

	if want, got := []int{14, 30, 10}, table.ColumnWidths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("\n\nwant %v\n\ngot %v", want, got)
	}

	if got := ansi.StringWidth(ansiStrip(table.headersView())); got != width {
		t.Errorf("header width: want %d, got %d", width, got)
	}
	if got := ansi.StringWidth(ansiStrip(table.renderRow(0))); got != width {
		t.Errorf("row width: want %d, got %d", width, got)
	}
}

func TestModel_FromValues(t *testing.T) {
	input := "foo1,bar1\nfoo2,bar2\nfoo3,bar3"
	table := New(WithColumns([]Column{{Title: "Foo"}, {Title: "Bar"}}))
	table.FromValues(input, ",")

	if len(table.rows) != 3 {
		t.Fatalf("expect table to have 3 rows but it has %d", len(table.rows))
	}

	expect := []Row{
		{"foo1", "bar1"},
		{"foo2", "bar2"},
		{"foo3", "bar3"},
	}
	if !reflect.DeepEqual(table.rows, expect) {
		t.Fatalf("\n\nwant %v\n\ngot %v", expect, table.rows)
	}
}

func TestModel_FromValues_WithTabSeparator(t *testing.T) {
	input := "foo1.\tbar1\nfoo,bar,baz\tbar,2"
	table := New(WithColumns([]Column{{Title: "Foo"}, {Title: "Bar"}}))
	table.FromValues(input, "\t")

	if len(table.rows) != 2 {
		t.Fatalf("expect table to have 2 rows but it has %d", len(table.rows))
	}

	expect := []Row{
		{"foo1.", "bar1"},
		{"foo,bar,baz", "bar,2"},
	}
	if !reflect.DeepEqual(table.rows, expect) {
		t.Fatalf("\n\nwant %v\n\ngot %v", expect, table.rows)
	}
}

func TestModel_RenderRow(t *testing.T) {
	tests := []struct {
		name     string
		table    *Model
		expected string
	}{
		{
			name: "simple row",
			table: &Model{
				rows:     []Row{{"Foooooo", "Baaaaar", "Baaaaaz"}},
				cols:     testCols,
				styles:   Styles{Cell: lipgloss.NewStyle()},
				viewport: viewport.New(viewport.WithWidth(30)),
			},
			expected: "Foooooo   Baaaaar   Baaaaaz   ",
		},
		{
			name: "simple row with truncations",
			table: &Model{
				rows:     []Row{{"Foooooooooo", "Baaaaaaaaar", "Quuuuuuuuux"}},
				cols:     testCols,
				styles:   Styles{Cell: lipgloss.NewStyle()},
				viewport: viewport.New(viewport.WithWidth(30)),
			},
			expected: "Foooooooo…Baaaaaaaa…Quuuuuuuu…",
		},
		{
			name: "simple row avoiding truncations",
			table: &Model{
				rows:     []Row{{"Fooooooooo", "Baaaaaaaar", "Quuuuuuuux"}},
				cols:     testCols,
				styles:   Styles{Cell: lipgloss.NewStyle()},
				viewport: viewport.New(viewport.WithWidth(30)),
			},
			expected: "FoooooooooBaaaaaaaarQuuuuuuuux",
		},
		{
			name: "simple row with style func",
			table: &Model{
				rows:     []Row{{"Foooooo", "Baaaaar", "Baaaaaz"}},
				cols:     testCols,
				viewport: viewport.New(viewport.WithWidth(30)),
				styleFunc: func(ctx RenderContext) lipgloss.Style {
					if strings.HasSuffix(ctx.Value, "z") {
						return lipgloss.NewStyle().Transform(strings.ToLower)
					}
					return lipgloss.NewStyle().Transform(strings.ToUpper)
				},
			},
			expected: "FOOOOOO   BAAAAAR   baaaaaz   ",
		},
		{
			name: "auto sized columns",
			table: &Model{
				rows: []Row{{"Foo", "Baaaaar"}},
				cols: []Column{
					{Title: "col1"},
					{Title: "col2"},
				},
				styles:     Styles{Cell: lipgloss.NewStyle()},
				showHeader: true,
				viewport:   viewport.New(viewport.WithWidth(30)),
			},
			expected: "Foo Baaaaar",
		},
		{
			name: "flex sized columns",
			table: &Model{
				rows: []Row{{"Foo", "Bar"}},
				cols: []Column{
					{Title: "col1", Sizing: Flex(1)},
					{Title: "col2", Sizing: Flex(1)},
				},
				styles:   Styles{Cell: lipgloss.NewStyle()},
				viewport: viewport.New(viewport.WithWidth(20)),
			},
			expected: "Foo       Bar       ",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.table.ensureLayout()

			row := tc.table.renderRow(0)
			if row != tc.expected {
				t.Fatalf("\n\nWant: \n%s\n\nGot:  \n%s\n", tc.expected, row)
			}
		})
	}
}

func TestModel_RenderRow_ZeroWidthColumns(t *testing.T) {
	// There's no room for any column, so nothing is rendered.
	table := New(WithWidth(0), WithColumns(testCols), WithRows([]Row{{"a", "b", "c"}}))

	if got := ansiStrip(table.renderRow(0)); got != "" {
		t.Fatalf("want an empty row, got %q", got)
	}
}

func TestModel_RenderRow_AnsiWidth(t *testing.T) {
	value := "\x1b[31mABCDEFGH\x1b[0m"
	table := &Model{
		rows:     []Row{{value}},
		cols:     []Column{{Title: "col1", Sizing: Fixed(8)}},
		styles:   Styles{Cell: lipgloss.NewStyle()},
		viewport: viewport.New(viewport.WithWidth(8)),
	}
	table.ensureLayout()

	got := ansi.Strip(table.renderRow(0))
	want := "ABCDEFGH"
	if got != want {
		t.Fatalf("\n\nWant: \n%s\n\nGot:  \n%s\n", want, got)
	}
}

func TestTableAlignment(t *testing.T) {
	t.Run("No border", func(t *testing.T) {
		biscuits := New(
			WithWidth(59),
			WithHeight(5),
			WithColumns([]Column{
				{Title: "Name", Sizing: Fixed(25)},
				{Title: "Country of Origin", Sizing: Fixed(16)},
				{Title: "Dunk-able", Sizing: Fixed(12)},
			}),
			WithRows([]Row{
				{"Chocolate Digestives", "UK", "Yes"},
				{"Tim Tams", "Australia", "No"},
				{"Hobnobs", "UK", "Yes"},
			}),
		)
		got := ansiStrip(biscuits.View().Content)
		golden.RequireEqual(t, []byte(got))
	})
	t.Run("With border", func(t *testing.T) {
		baseStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

		s := DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)

		biscuits := New(
			WithWidth(59),
			WithHeight(5),
			WithColumns([]Column{
				{Title: "Name", Sizing: Fixed(25)},
				{Title: "Country of Origin", Sizing: Fixed(16)},
				{Title: "Dunk-able", Sizing: Fixed(12)},
			}),
			WithRows([]Row{
				{"Chocolate Digestives", "UK", "Yes"},
				{"Tim Tams", "Australia", "No"},
				{"Hobnobs", "UK", "Yes"},
			}),
			WithStyles(s),
		)
		got := ansiStrip(baseStyle.Render(biscuits.View().Content))
		golden.RequireEqual(t, []byte(got))
	})
}

func ansiStrip(s string) string {
	// Replace all \r\n with \n
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return ansi.Strip(s)
}

func TestCursorNavigation(t *testing.T) {
	type wantS struct {
		cursor [2]int
		mode   Mode
	}
	tests := map[string]struct {
		rows   []Row
		action func(*Model)
		want   wantS
	}{
		"New": {
			rows: []Row{
				{"r1"},
				{"r2"},
				{"r3"},
			},
			action: func(_ *Model) {},
			want:   wantS{cursor: [2]int{0, 0}, mode: ModeNormal},
		},
		"MoveDown": {
			rows: []Row{
				{"r1"},
				{"r2"},
				{"r3"},
				{"r4"},
			},
			action: func(t *Model) {
				t.MoveDown(2)
			},
			want: wantS{cursor: [2]int{2, 0}, mode: ModeNormal},
		},
		"MoveUp": {
			rows: []Row{
				{"r1"},
				{"r2"},
				{"r3"},
				{"r4"},
			},
			action: func(t *Model) {
				t.cursor[0] = 3
				t.MoveUp(2)
			},
			want: wantS{cursor: [2]int{1, 0}, mode: ModeNormal},
		},
		"MoveRight": {
			rows: []Row{
				{"r1a", "r1b", "r1c"},
			},
			action: func(t *Model) {
				t.MoveRight(2)
			},
			want: wantS{cursor: [2]int{0, 2}, mode: ModeCell},
		},
		"MoveLeft": {
			rows: []Row{
				{"r1a", "r1b", "r1c"},
			},
			action: func(t *Model) {
				t.cursor[1] = 2
				t.MoveLeft(1)
			},
			want: wantS{cursor: [2]int{0, 1}, mode: ModeCell},
		},
		"GotoBottom": {
			rows: []Row{
				{"r1"},
				{"r2"},
				{"r3"},
				{"r4"},
			},
			action: func(t *Model) {
				t.GotoBottom()
			},
			want: wantS{cursor: [2]int{3, 0}, mode: ModeNormal},
		},
		"GotoTop": {
			rows: []Row{
				{"r1"},
				{"r2"},
				{"r3"},
				{"r4"},
			},
			action: func(t *Model) {
				t.cursor[0] = 3
				t.GotoTop()
			},
			want: wantS{cursor: [2]int{0, 0}, mode: ModeNormal},
		},
		"SetCursor Row": {
			rows: []Row{
				{"r1"},
				{"r2"},
				{"r3"},
				{"r4"},
			},
			action: func(t *Model) {
				t.SetCursor(2, -1)
			},
			want: wantS{cursor: [2]int{2, 0}, mode: ModeNormal},
		},
		"SetCursor Row Col": {
			rows: []Row{
				{"r1a", "r1b"},
				{"r2a", "r2b"},
				{"r3a", "r3b"},
				{"r4a", "r4b"},
			},
			action: func(t *Model) {
				t.SetCursor(2, 1)
			},
			want: wantS{cursor: [2]int{2, 1}, mode: ModeNormal},
		},
		"MoveDown with overflow": {
			rows: []Row{
				{"r1"},
				{"r2"},
				{"r3"},
				{"r4"},
			},
			action: func(t *Model) {
				t.MoveDown(5)
			},
			want: wantS{cursor: [2]int{3, 0}, mode: ModeNormal},
		},
		"MoveUp with overflow": {
			rows: []Row{
				{"r1"},
				{"r2"},
				{"r3"},
				{"r4"},
			},
			action: func(t *Model) {
				t.cursor[0] = 3
				t.MoveUp(5)
			},
			want: wantS{cursor: [2]int{0, 0}, mode: ModeNormal},
		},
		"Blur does not stop movement": {
			rows: []Row{
				{"r1"},
				{"r2"},
				{"r3"},
				{"r4"},
			},
			action: func(t *Model) {
				t.Blur()
				t.MoveDown(2)
			},
			want: wantS{cursor: [2]int{2, 0}, mode: ModeNormal},
		},
		"SetMode Normal": {
			rows: []Row{
				{"r1a", "r1b", "r1c"},
			},
			action: func(t *Model) {
				t.SetCursor(0, 1)
				t.SetMode(ModeNormal)
			},
			want: wantS{cursor: [2]int{0, 0}, mode: ModeNormal},
		},
		"SetMode Cell": {
			rows: []Row{
				{"r1a", "r1b", "r1c"},
			},
			action: func(t *Model) {
				t.SetCursor(0, 1)
				t.SetMode(ModeCell)
			},
			want: wantS{cursor: [2]int{0, 1}, mode: ModeCell},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			table := New(WithColumns(testCols), WithRows(tc.rows))
			tc.action(&table)

			if table.Cursor() != tc.want.cursor {
				t.Errorf("want %d, got %d", tc.want.cursor, table.Cursor())
			}
			if table.Mode() != tc.want.mode {
				t.Errorf("want %d, got %d", tc.want.mode, table.Mode())
			}
		})
	}
}

func TestFixedColumn(t *testing.T) {
	type wantS struct {
		cursor      [2]int
		mode        Mode
		fixedColumn int
	}
	tests := map[string]struct {
		opts   []Option
		action func(*Model)
		want   wantS
	}{
		"Default": {
			opts: []Option{
				WithColumns(testCols),
				WithMode(ModeFixedColumn),
				WithFixedColumn(1),
			},
			action: func(_ *Model) {},
			want:   wantS{cursor: [2]int{0, 1}, mode: ModeFixedColumn, fixedColumn: 1},
		},
		"MoveRight": {
			opts: []Option{
				WithColumns(testCols),
				WithMode(ModeFixedColumn),
				WithFixedColumn(1),
			},
			action: func(t *Model) {
				t.MoveRight(1)
			},
			want: wantS{cursor: [2]int{0, 1}, mode: ModeFixedColumn, fixedColumn: 1},
		},
		"MoveLeft": {
			opts: []Option{
				WithColumns(testCols),
				WithMode(ModeFixedColumn),
				WithFixedColumn(1),
			},
			action: func(t *Model) {
				t.MoveLeft(1)
			},
			want: wantS{cursor: [2]int{0, 1}, mode: ModeFixedColumn, fixedColumn: 1},
		},
		"SetMode Normal": {
			opts: []Option{
				WithColumns(testCols),
				WithMode(ModeFixedColumn),
				WithFixedColumn(2),
			},
			action: func(t *Model) {
				t.SetMode(ModeNormal)
			},
			want: wantS{cursor: [2]int{0, 0}, mode: ModeNormal, fixedColumn: 2},
		},
		"SetMode FixedColumn from cell": {
			opts: []Option{
				WithColumns(testCols),
				WithRows([]Row{{"a", "b", "c"}}),
			},
			action: func(t *Model) {
				t.SetCursor(0, 2)
				t.SetMode(ModeFixedColumn)
				t.SetFixedColumn(1)
			},
			want: wantS{cursor: [2]int{0, 1}, mode: ModeFixedColumn, fixedColumn: 1},
		},
		"SetFixedColumn clamps high": {
			opts: []Option{
				WithColumns(testCols),
			},
			action: func(t *Model) {
				t.SetFixedColumn(10)
			},
			want: wantS{cursor: [2]int{0, 0}, mode: ModeNormal, fixedColumn: 2},
		},
		"SetFixedColumn clamps low": {
			opts: []Option{
				WithColumns(testCols),
			},
			action: func(t *Model) {
				t.SetFixedColumn(-1)
			},
			want: wantS{cursor: [2]int{0, 0}, mode: ModeNormal, fixedColumn: 0},
		},
		"SetFixedColumn updates cursor": {
			opts: []Option{
				WithColumns(testCols),
				WithMode(ModeFixedColumn),
				WithFixedColumn(0),
			},
			action: func(t *Model) {
				t.SetFixedColumn(2)
			},
			want: wantS{cursor: [2]int{0, 2}, mode: ModeFixedColumn, fixedColumn: 2},
		},
		"SetFixedColumn no columns": {
			action: func(t *Model) {
				t.SetFixedColumn(3)
			},
			want: wantS{cursor: [2]int{0, 0}, mode: ModeNormal, fixedColumn: 0},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			table := New(tc.opts...)
			tc.action(&table)

			if table.Cursor() != tc.want.cursor {
				t.Errorf("cursor: want %d, got %d", tc.want.cursor, table.Cursor())
			}
			if table.Mode() != tc.want.mode {
				t.Errorf("mode: want %d, got %d", tc.want.mode, table.Mode())
			}
			if table.FixedColumn() != tc.want.fixedColumn {
				t.Errorf("fixed column: want %d, got %d", tc.want.fixedColumn, table.FixedColumn())
			}
		})
	}
}

func TestModel_SelectedCell(t *testing.T) {
	table := New(WithColumns(testCols), WithRows([]Row{{"a1", "a2", "a3"}, {"b1", "b2", "b3"}}))
	table.SetCursor(1, 1)
	got := table.SelectedCell()
	want := "b2"
	if got != want {
		t.Errorf("want %s, got %s", want, got)
	}
}

func TestModel_SetRow(t *testing.T) {
	tests := []struct {
		name     string
		row      Row
		idx      int
		wantRows []Row
		wantErr  string
	}{
		{
			name:     "ok",
			row:      Row{"r1-new"},
			idx:      0,
			wantRows: []Row{{"r1-new"}, {"r2"}},
		},
		{
			name:     "err/index_negative",
			row:      Row{"r1-new"},
			idx:      -1,
			wantRows: []Row{{"r1"}, {"r2"}},
			wantErr:  "index -1 is out of bounds",
		},
		{
			name:     "err/index_out_of_bounds",
			row:      Row{"r1-new"},
			idx:      2,
			wantRows: []Row{{"r1"}, {"r2"}},
			wantErr:  "index 2 is out of bounds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := New(WithColumns(testCols), WithRows([]Row{{"r1"}, {"r2"}}))

			if len(table.rows) != 2 {
				t.Fatalf("want 2, got %d", len(table.rows))
			}

			gotErr := table.SetRow(tt.idx, tt.row)
			if tt.wantErr != "" {
				if gotErr == nil || gotErr.Error() != tt.wantErr {
					t.Fatalf("want error %q, got %q", tt.wantErr, gotErr)
				}
			} else {
				if gotErr != nil {
					t.Fatalf("got unexpected error %q", gotErr)
				}
			}

			if len(table.rows) != 2 {
				t.Fatalf("want 2, got %d", len(table.rows))
			}

			if !reflect.DeepEqual(table.rows, tt.wantRows) {
				t.Fatalf("\n\nwant %v\n\ngot %v", tt.wantRows, table.rows)
			}
		})
	}
}

func TestModel_SetRows(t *testing.T) {
	table := New(WithColumns(testCols))

	if len(table.rows) != 0 {
		t.Fatalf("want 0, got %d", len(table.rows))
	}

	table.SetRows([]Row{{"r1"}, {"r2"}})

	if len(table.rows) != 2 {
		t.Fatalf("want 2, got %d", len(table.rows))
	}

	want := []Row{{"r1"}, {"r2"}}
	if !reflect.DeepEqual(table.rows, want) {
		t.Fatalf("\n\nwant %v\n\ngot %v", want, table.rows)
	}
}

func TestModel_SetColumns(t *testing.T) {
	table := New()

	if len(table.cols) != 0 {
		t.Fatalf("want 0, got %d", len(table.cols))
	}

	table.SetColumns([]Column{{Title: "Foo"}, {Title: "Bar"}})

	if len(table.cols) != 2 {
		t.Fatalf("want 2, got %d", len(table.cols))
	}

	want := []Column{{Title: "Foo"}, {Title: "Bar"}}
	if !reflect.DeepEqual(table.cols, want) {
		t.Fatalf("\n\nwant %v\n\ngot %v", want, table.cols)
	}
}

func TestModel_SetHeader(t *testing.T) {
	table := New(
		WithWidth(36),
		WithColumns(testCols),
		WithRows([]Row{{"r1", "r2", "r3"}}),
		WithHeight(10),
	)

	if !table.showHeader {
		t.Fatal("want showHeader to be true by default")
	}
	initialHeight := table.viewport.Height()

	table.SetHeader(false)
	if table.showHeader {
		t.Fatal("want showHeader to be false after SetHeader(false)")
	}
	if table.viewport.Height() != initialHeight+1 {
		t.Fatalf("want viewport height %d after hiding header, got %d", initialHeight+1, table.viewport.Height())
	}

	table.SetHeader(true)
	if !table.showHeader {
		t.Fatal("want showHeader to be true after SetHeader(true)")
	}
	if table.viewport.Height() != initialHeight {
		t.Fatalf("want viewport height %d after showing header, got %d", initialHeight, table.viewport.Height())
	}
}

func TestModel_View(t *testing.T) {
	tests := map[string]struct {
		modelFunc func() Model
		skip      bool
	}{
		"Empty": {
			modelFunc: func() Model {
				return New(
					WithWidth(60),
					WithHeight(21),
				)
			},
		},
		"Single row and column": {
			modelFunc: func() Model {
				return New(
					WithWidth(27),
					WithHeight(21),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives"},
					}),
				)
			},
		},
		"Multiple rows and columns": {
			modelFunc: func() Model {
				return New(
					WithWidth(59),
					WithHeight(21),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
				)
			},
		},
		"Auto columns": {
			modelFunc: func() Model {
				return New(
					WithWidth(60),
					WithHeight(10),
					WithColumns([]Column{
						{Title: "Name"},
						{Title: "Country of Origin"},
						{Title: "Dunk-able"},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
				)
			},
		},
		"Flex and percent columns": {
			modelFunc: func() Model {
				return New(
					WithWidth(60),
					WithHeight(10),
					WithColumns([]Column{
						{Title: "Name", Sizing: Flex(2)},
						{Title: "Country of Origin", Sizing: Flex(1), MinWidth: 12},
						{Title: "Dunk-able", Sizing: Percent(0.2)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
				)
			},
		},
		"No header": {
			modelFunc: func() Model {
				return New(
					WithWidth(59),
					WithHeight(21),
					WithHeader(false),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
				)
			},
		},
		// TODO(fix): since the table height is tied to the viewport height, adding vertical padding to the headers' height directly increases the table height.
		"Extra padding": {
			modelFunc: func() Model {
				s := DefaultStyles()
				s.Header = lipgloss.NewStyle().Padding(2, 2)
				s.Cell = lipgloss.NewStyle().Padding(2, 2)

				return New(
					WithWidth(60),
					WithHeight(10),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
					WithStyles(s),
				)
			},
		},
		"No padding": {
			modelFunc: func() Model {
				s := DefaultStyles()
				s.Header = lipgloss.NewStyle()
				s.Cell = lipgloss.NewStyle()

				return New(
					WithWidth(53),
					WithHeight(10),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
					WithStyles(s),
				)
			},
		},
		// TODO(?): the total height is modified with bordered headers, however not with bordered cells. Is this expected/desired?
		"Bordered headers": {
			modelFunc: func() Model {
				return New(
					WithWidth(59),
					WithHeight(23),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
					WithStyles(Styles{
						Header: lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()),
					}),
				)
			},
		},
		// TODO(fix): Headers are not horizontally aligned with cells due to the border adding width to the cells.
		"Bordered cells": {
			modelFunc: func() Model {
				return New(
					WithWidth(59),
					WithHeight(21),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
					WithStyles(Styles{
						Cell: lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()),
					}),
				)
			},
		},
		"Height greater than rows": {
			modelFunc: func() Model {
				return New(
					WithWidth(59),
					WithHeight(6),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
				)
			},
		},
		"Height less than rows": {
			modelFunc: func() Model {
				return New(
					WithWidth(59),
					WithHeight(2),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
				)
			},
		},
		// TODO(fix): spaces are added to the right of the viewport to fill the width, but the headers end as though they are not aware of the width.
		"Width greater than columns": {
			modelFunc: func() Model {
				return New(
					WithWidth(80),
					WithHeight(21),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
				)
			},
		},
		// The fixed widths oversubscribe the table, so every column is shrunk
		// in proportion to the space it asked for.
		"Width less than columns": {
			modelFunc: func() Model {
				return New(
					WithWidth(30),
					WithHeight(15),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
				)
			},
		},
		"Width less than columns with min widths": {
			modelFunc: func() Model {
				return New(
					WithWidth(30),
					WithHeight(15),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25), MinWidth: 15},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
				)
			},
		},
		"Modified viewport height": {
			modelFunc: func() Model {
				m := New(
					WithWidth(59),
					WithHeight(15),
					WithColumns([]Column{
						{Title: "Name", Sizing: Fixed(25)},
						{Title: "Country of Origin", Sizing: Fixed(16)},
						{Title: "Dunk-able", Sizing: Fixed(12)},
					}),
					WithRows([]Row{
						{"Chocolate Digestives", "UK", "Yes"},
						{"Tim Tams", "Australia", "No"},
						{"Hobnobs", "UK", "Yes"},
					}),
				)

				m.viewport.SetHeight(2)

				return m
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.skip {
				t.Skip()
			}

			table := tc.modelFunc()

			got := ansi.Strip(table.View().Content)

			golden.RequireEqual(t, []byte(got))
		})
	}
}

// TODO: Fix table to make this test will pass.
func TestModel_View_CenteredInABox(t *testing.T) {
	t.Skip()

	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		Align(lipgloss.Center)

	table := New(
		WithHeight(6),
		WithWidth(80),
		WithColumns([]Column{
			{Title: "Name", Sizing: Fixed(25)},
			{Title: "Country of Origin", Sizing: Fixed(16)},
			{Title: "Dunk-able", Sizing: Fixed(12)},
		}),
		WithRows([]Row{
			{"Chocolate Digestives", "UK", "Yes"},
			{"Tim Tams", "Australia", "No"},
			{"Hobnobs", "UK", "Yes"},
		}),
	)

	tableView := ansi.Strip(table.View().Content)
	got := boxStyle.Render(tableView)

	golden.RequireEqual(t, []byte(got))
}
