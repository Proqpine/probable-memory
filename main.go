package main

// https://www.alexedwards.net/blog/introduction-to-using-sql-databases-in-go
// https://stackoverflow.com/questions/32746858/how-to-represent-postgresql-interval-in-go
import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/sanyokbig/pqinterval"
)

type PgDuration time.Duration

type WorkEntry struct {
	ID          int
	Date        time.Time
	StartTime   time.Time
	EndTime     time.Time
	Duration    pqinterval.Interval
	Description string
	Project     string
}

var entries []WorkEntry

const (
	host     = "localhost"
	port     = 54322
	user     = "postgres"
	password = "postgres"
	dbname   = "postgres"
)

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected!")

	for {
		fmt.Println("\n1. Start work")
		fmt.Println("2. End work")
		fmt.Println("3. View entries")
		fmt.Println("4. Export to Markdown")
		fmt.Println("5. Quit")
		fmt.Print("Choose an option: ")

		var choice int
		fmt.Scanln(&choice)
		switch choice {
		case 1:
			taskID, err := startWork(db)
			if err != nil {
				log.Fatalf("Failed at 1: %v", err)
			}
			fmt.Printf("Added new task with ID: %d\n", taskID)
		case 2:
			err := endWork(db)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Work ended.")
		case 3:
			err := viewEntries(db)
			if err != nil {
				log.Printf("Failed to view entries: %v", err)
			}
			// case 4:
			// 	exportToMarkdown()
		case 5:
			return
		default:
			fmt.Println("Invalid option")
		}
	}
}

func startWork(db *sql.DB) (int, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter task description: ")
	desc, _ := reader.ReadString('\n')
	description := strings.TrimSpace(desc)

	fmt.Print("Enter project name: ")
	proj, _ := reader.ReadString('\n')
	project := strings.TrimSpace(proj)

	// fmt.Printf("%s, %s", description, project)
	var taskID int
	err := db.QueryRow(`
	    INSERT INTO work_entry (date, start_time, description, project)
	    VALUES ($1, $2, $3, $4)
	    RETURNING id
	`, time.Now(), time.Now(), description, project).Scan(&taskID)

	if err != nil {
		return 0, err
	}

	return taskID, nil
}

func endWork(db *sql.DB) error {
	if len(entries) == 0 || !entries[len(entries)-1].EndTime.IsZero() {
		fmt.Println("No active work entry.")
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter task id: ")
	id, _ := reader.ReadString('\n')
	taskID := strings.TrimSpace(id)

	endTime := time.Now()
	_, err := db.Exec(`
	UPDATE work_entry
	SET end_time = $1, duration = $1 - start_time
	WHERE ID = $2
	`, endTime, taskID)

	return err
}

func viewEntries(db *sql.DB) error {
	rows, err := db.Query(`
        SELECT id, date, start_time, end_time, duration, description, project
        FROM work_entry
		WHERE end_time IS NOT NULL
        ORDER BY date DESC, start_time DESC
    `)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var entry WorkEntry

		err := rows.Scan(&entry.ID, &entry.Date, &entry.StartTime, &entry.EndTime, &entry.Duration, &entry.Description, &entry.Project)
		if err != nil {
			return err
		}

		parsedDuration, err := entry.Duration.Duration()
		if err != nil {
			return err
		}
		duration := parsedDuration.String()

		fmt.Print("\n")
		fmt.Printf("%s | %s - %s | %v | %s | %s\n",
			entry.Date.Format("2006-01-02"),
			entry.StartTime.Format("15:04"),
			entry.EndTime.Format("15:04"),
			duration,
			entry.Description,
			entry.Project)
	}

	return rows.Err()
}
