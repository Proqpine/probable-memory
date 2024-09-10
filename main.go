package main

// https://www.alexedwards.net/blog/introduction-to-using-sql-databases-in-go
// https://stackoverflow.com/questions/32746858/how-to-represent-postgresql-interval-in-go
import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Proqpine/probable-memory/sqlite"
	"github.com/Proqpine/probable-memory/src"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
)

const (
	hotPink  = lipgloss.Color("#FF06B7")
	darkGray = lipgloss.Color("#767676")
)

var (
	appStyle   = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)
	statusMessage = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
			Render()
	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)
)

type model struct {
	list                  list.Model
	Queries               *sqlite.Queries
	Activities            []sqlite.Activity
	SelectedActivity      *sqlite.Activity
	WeeklyProgressSummary *string
	IsGeneratingSummary   bool
	Error                 error
	Loading               bool
	keys                  keyMap
	addingActivity        bool
	inputs                []textinput.Model
	inputIndex            int
	viewport              viewport.Model
	viewingActivity       bool
	editingActivity       bool
	editInputs            []textinput.Model
	editInputIndex        int
}

type keyMap struct {
	toggleSpinner    key.Binding
	toggleTitleBar   key.Binding
	toggleStatusBar  key.Binding
	togglePagination key.Binding
	toggleHelpMenu   key.Binding
	insertItem       key.Binding
	viewItem         key.Binding
	editItem         key.Binding
}

func main() {
	src.SummariseActivities()
	dbConnection := setupDBConnection()
	defer dbConnection.Close()
	db := sqlite.New(dbConnection)
	p := tea.NewProgram(initialModel(db))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

func newKeyMap() keyMap {
	return keyMap{
		editItem: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit item"),
		),
		viewItem: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "view item"),
		),
		insertItem: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add item"),
		),
		toggleSpinner: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle spinner"),
		),
		toggleTitleBar: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "toggle title"),
		),
		toggleStatusBar: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "toggle status"),
		),
		togglePagination: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle pagination"),
		),
		toggleHelpMenu: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle help"),
		),
	}
}

type item struct {
	activity sqlite.Activity
}

func (i item) Title() string       { return i.activity.ActivityName }
func (i item) Description() string { return i.activity.Description }
func (i item) FilterValue() string { return i.activity.ActivityName }

type errorMsg struct {
	error error
}

type fetchActivitiesMsg struct {
	activities []sqlite.Activity
}

func (m model) Init() tea.Cmd {
	return m.fetchActivities
}

func initialModel(queries *sqlite.Queries) model {
	keys := newKeyMap()
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Activities"
	l.Styles.Title = titleStyle
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.toggleSpinner,
			keys.insertItem,
			keys.viewItem,
			keys.editItem,
			keys.toggleTitleBar,
			keys.toggleStatusBar,
			keys.togglePagination,
			keys.toggleHelpMenu,
		}
	}
	m := model{
		list:             l,
		Queries:          queries,
		Activities:       []sqlite.Activity{},
		Loading:          true,
		keys:             keys,
		addingActivity:   false,
		inputs:           make([]textinput.Model, 5),
		SelectedActivity: nil,
		viewport:         viewport.New(80, 20),
		viewingActivity:  false,
		editingActivity:  false,
		editInputs:       make([]textinput.Model, 5),
		editInputIndex:   0,
	}
	for i := range m.editInputs {
		t := textinput.New()
		switch i {
		case 0:
			t.Placeholder = "Activity Name"
		case 1:
			t.Placeholder = "Description"
		case 2:
			t.Placeholder = "Project"
		case 3:
			t.Placeholder = "Notes"
		case 4:
			t.Placeholder = "Duration (in seconds)"
		}
		m.editInputs[i] = t
	}
	for i := range m.inputs {
		t := textinput.New()
		switch i {
		case 0:
			t.Placeholder = "Activity Name"
		case 1:
			t.Placeholder = "Description"
		case 2:
			t.Placeholder = "Project"
		case 3:
			t.Placeholder = "Notes"
		case 4:
			t.Placeholder = "Duration (in seconds)"
		}
		m.inputs[i] = t
	}
	m.inputs[0].Focus()

	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		if m.viewingActivity {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		} else {
			m.list.SetSize(msg.Width-h, msg.Height-v)
		}

	case tea.KeyMsg:
		if m.editingActivity {
			switch msg.String() {
			case "up":
				m.editInputIndex = max(0, m.editInputIndex-1)
				m.editInputs[m.editInputIndex].Focus()
				return m, nil
			case "down":
				m.editInputIndex = min(len(m.editInputs)-1, m.editInputIndex+1)
				m.editInputs[m.editInputIndex].Focus()
				return m, nil
			case "esc":
				m.editingActivity = false
				m.editInputIndex = 0
				return m, nil
			}
			if m.editInputIndex == len(m.editInputs)-1 && msg.String() == "enter" {
				return m, m.updateActivity
			}
			var cmd tea.Cmd
			m.editInputs[m.editInputIndex], cmd = m.editInputs[m.editInputIndex].Update(msg)
			return m, cmd
		} else if m.viewingActivity {
			switch msg.String() {
			case "q", "esc":
				m.viewingActivity = false
				m.SelectedActivity = nil
				return m, nil
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		} else if m.addingActivity {
			switch msg.String() {
			case "enter":
				if m.inputIndex == len(m.inputs)-1 {
					return m, m.addActivity
				}
				m.inputIndex++
				m.inputs[m.inputIndex].Focus()
				return m, nil
			case "esc":
				m.addingActivity = false
				return m, nil
			}
			var cmd tea.Cmd
			m.inputs[m.inputIndex], cmd = m.inputs[m.inputIndex].Update(msg)
			return m, cmd
		} else {
			switch {
			case key.Matches(msg, m.keys.toggleSpinner):
				cmd := m.list.ToggleSpinner()
				return m, cmd

			case key.Matches(msg, m.keys.toggleTitleBar):
				v := !m.list.ShowTitle()
				m.list.SetShowTitle(v)
				m.list.SetShowFilter(v)
				m.list.SetFilteringEnabled(v)
				return m, nil

			case key.Matches(msg, m.keys.toggleStatusBar):
				m.list.SetShowStatusBar(!m.list.ShowStatusBar())
				return m, nil

			case key.Matches(msg, m.keys.togglePagination):
				m.list.SetShowPagination(!m.list.ShowPagination())
				return m, nil

			case key.Matches(msg, m.keys.toggleHelpMenu):
				m.list.SetShowHelp(!m.list.ShowHelp())
				return m, nil

			case key.Matches(msg, m.keys.insertItem):
				m.addingActivity = true
				return m, nil

			case key.Matches(msg, m.keys.viewItem):
				if i, ok := m.list.SelectedItem().(item); ok {
					m.viewingActivity = true
					m.SelectedActivity = &i.activity
					m.viewport.SetContent(m.activityView())
					return m, nil
				}
			case key.Matches(msg, m.keys.editItem):
				if i, ok := m.list.SelectedItem().(item); ok {
					m.viewingActivity = true
					m.SelectedActivity = &i.activity
					m.editingActivity = true
					m.populateEditInputs()
					m.editInputs[0].Focus()
					return m, nil
				}
			}
		}

	case activityAddedMsg:
		m.addingActivity = false
		for i := range m.inputs {
			m.inputs[i].Reset()
		}
		m.inputIndex = 0
		return m, m.fetchActivities

	case fetchActivitiesMsg:
		m.Activities = msg.activities
		m.Loading = false
		items := make([]list.Item, len(m.Activities))
		for i, a := range m.Activities {
			items[i] = item{activity: a}
		}
		m.list.SetItems(items)
		return m, nil

	case errorMsg:
		m.Error = msg.error
		m.Loading = false
		return m, nil

	case activityUpdatedMsg:
		m.editingActivity = false
		m.viewingActivity = false
		m.SelectedActivity = nil
		for i := range m.editInputs {
			m.editInputs[i].Reset()
		}
		m.editInputIndex = 0
		return m, m.fetchActivities

	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) updateActivity() tea.Msg {
	if m.SelectedActivity == nil {
		return errorMsg{error: fmt.Errorf("no activity selected")}
	}

	duration, err := strconv.ParseInt(m.editInputs[4].Value(), 10, 64)
	if err != nil {
		return errorMsg{error: fmt.Errorf("invalid duration: %v", err)}
	}

	updatedActivity := sqlite.UpdateActivityParams{
		ID:           m.SelectedActivity.ID,
		ActivityName: m.editInputs[0].Value(),
		Description:  m.editInputs[1].Value(),
		Project:      m.editInputs[2].Value(),
		Notes:        m.editInputs[3].Value(),
		Duration:     sql.NullInt64{Int64: duration, Valid: true},
	}

	_, err = m.Queries.UpdateActivity(context.Background(), updatedActivity)
	if err != nil {
		return errorMsg{error: fmt.Errorf("failed to update activity: %v", err)}
	}

	return activityUpdatedMsg{}
}

type activityUpdatedMsg struct{}

func (m model) View() string {
	if m.Loading {
		return "Loading activities..."
	}
	if m.Error != nil {
		return fmt.Sprintf("Error: %v", m.Error)
	}
	if m.addingActivity {
		return m.addActivityView()
	}
	if m.editingActivity {
		return m.editActivityView()
	}
	if m.viewingActivity {
		return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	}
	return appStyle.Render(m.list.View())
}

func setupDBConnection() *sql.DB {
	db, err := sql.Open("sqlite3", "activity.db")
	if err != nil {
		panic(err)
	}
	return db
}

func (m *model) populateEditInputs() {
	if m.SelectedActivity == nil {
		return
	}

	m.editInputs[0].SetValue(m.SelectedActivity.ActivityName)
	m.editInputs[1].SetValue(m.SelectedActivity.Description)
	m.editInputs[2].SetValue(m.SelectedActivity.Project)
	m.editInputs[3].SetValue(m.SelectedActivity.Notes)
	m.editInputs[4].SetValue(fmt.Sprintf("%d", m.SelectedActivity.Duration.Int64))
}

func (m model) addActivityView() string {
	var s string
	for i := range m.inputs {
		s += m.inputs[i].View() + "\n"
	}
	return fmt.Sprintf(
		"Adding new activity\n\n%s\n\n(esc to cancel)",
		s,
	)
}

func (m model) editActivityView() string {
	// Define styles
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Bold(true)

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0).
		Width(50)

	focusedStyle := inputStyle.
		BorderForeground(lipgloss.Color("#FF00FF"))

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true)

	// Create the view
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Editing Activity") + "\n\n")

	// Inputs
	labels := []string{"Activity Name", "Description", "Project", "Notes", "Duration (seconds)"}
	for i, input := range m.editInputs {
		// Label
		b.WriteString(lipgloss.NewStyle().Bold(true).Render(labels[i]) + "\n")

		// Input field
		style := inputStyle
		if i == m.editInputIndex {
			style = focusedStyle
		}
		b.WriteString(style.Render(input.View()) + "\n\n")
	}

	// Instructions
	instructions := lipgloss.JoinHorizontal(lipgloss.Center,
		infoStyle.Render("↑/↓: Navigate • "),
		infoStyle.Render("Enter: Save • "),
		infoStyle.Render("Esc: Cancel"),
	)
	b.WriteString(instructions)

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m model) activityView() string {
	if m.SelectedActivity == nil {
		return "No activity selected"
	}

	a := m.SelectedActivity
	return lipgloss.NewStyle().Margin(1, 2).Render(fmt.Sprintf(`
Description: %s

Project: %s

Notes: %s

Duration: %d seconds

Start Time: %s

End Time: %s

(press 'e' to edit, esc to go back)`,
		a.Description,
		a.Project,
		a.Notes,
		a.Duration.Int64,
		a.StartTime.Format(time.RFC3339),
		a.EndTime.Time.Format(time.RFC3339),
	))
}

type activityAddedMsg struct{}

func (m model) addActivity() tea.Msg {
	activity := sqlite.InsertActivityParams{
		StartTime:    time.Now(),
		EndTime:      sql.NullTime{Time: time.Now(), Valid: true},
		ActivityName: m.inputs[0].Value(),
		Description:  m.inputs[1].Value(),
		Project:      m.inputs[2].Value(),
		Notes:        m.inputs[3].Value(),
	}
	duration, _ := time.ParseDuration(m.inputs[4].Value() + "s")
	activity.Duration = sql.NullInt64{Int64: int64(duration.Seconds()), Valid: true}

	_, err := m.Queries.InsertActivity(context.Background(), activity)
	if err != nil {
		return errorMsg{err}
	}
	return activityAddedMsg{}
}

func (m model) headerView() string {
	title := titleStyle.Render(m.SelectedActivity.ActivityName)
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m model) fetchActivities() tea.Msg {
	activities, err := m.Queries.QueryActivities(context.Background())
	if err != nil {
		return errorMsg{err}
	}
	items := make([]list.Item, len(activities))
	for i, a := range activities {
		items[i] = item{activity: a}
	}
	m.list.SetItems(items)
	return fetchActivitiesMsg{activities: activities}
}
