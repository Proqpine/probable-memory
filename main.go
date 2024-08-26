package main

// https://www.alexedwards.net/blog/introduction-to-using-sql-databases-in-go
// https://stackoverflow.com/questions/32746858/how-to-represent-postgresql-interval-in-go
import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Ayomided/probable-memory.git/sqlite"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
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
}

type keyMap struct {
	toggleSpinner    key.Binding
	toggleTitleBar   key.Binding
	toggleStatusBar  key.Binding
	togglePagination key.Binding
	toggleHelpMenu   key.Binding
	insertItem       key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
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
			keys.toggleTitleBar,
			keys.toggleStatusBar,
			keys.togglePagination,
			keys.toggleHelpMenu,
		}
	}
	m := model{
		list:           l,
		Queries:        queries,
		Activities:     []sqlite.Activity{},
		Loading:        true,
		keys:           keys,
		addingActivity: false,
		inputs:         make([]textinput.Model, 5),
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		if m.addingActivity {
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
		}
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
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

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
	return appStyle.Render(m.list.View())
}

func main() {
	dbConnection := setupDBConnection()
	defer dbConnection.Close()
	db := sqlite.New(dbConnection)
	p := tea.NewProgram(initialModel(db))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

func setupDBConnection() *sql.DB {
	db, err := sql.Open("sqlite3", "activity.db")
	if err != nil {
		panic(err)
	}
	return db
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
