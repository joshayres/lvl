package main

import (
	"context"
	"database/sql"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	_ "embed"

	_ "modernc.org/sqlite"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/joshayres/lvl/lvl"
	"github.com/joshayres/lvl/templates"
)

//go:embed schema.sql
var ddl string

func NewDatabase(ctx context.Context) (*lvl.Queries, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}

	// Create the tables
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return nil, err
	}

	q := lvl.New(db)

	return q, nil
}

func main() {
	r := chi.NewRouter()
	ctx := context.Background()

	db, err := NewDatabase(ctx)
	if err != nil {
		panic(err)
	}
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		habits, err := db.GetHabits(ctx)
		if err != nil {
			log.Println(err)
		}
		templHabit := make([]templates.Habit, len(habits))
		for i := range habits {
			habitLogs, err := db.GetHabitLogsForHabit(ctx, int64(habits[i].ID))
			if err != nil {
				log.Println(err)
			}
			templLogs := make([]templates.HabitLog, len(habitLogs))
			for j := range habitLogs {
				templLogs[j].LogDate = time.Unix(habitLogs[j].LogDate, 0)
			}
			templHabit[i] = templates.Habit{
				ID:    int(habits[i].ID),
				Name:  habits[i].Name,
				Exp:   int(habits[i].Exp),
				Level: int(habits[i].Level),
				Logs:  templLogs,
			}
		}

		h := templates.Index(templHabit)
		templates.Layout(h).Render(context.Background(), w)
	})

	r.Post("/create-habit", func(w http.ResponseWriter, r *http.Request) {
		name := r.FormValue("name")
		_, err := db.CreateHabit(ctx, lvl.CreateHabitParams{
			Name:  name,
			Level: 1,
			Exp:   0,
		})
		if err != nil {
			log.Println(err)
		}
		habits, err := db.GetHabits(ctx)
		if err != nil {
			log.Println(err)
		}
		templHabit := make([]templates.Habit, len(habits))
		for i := range habits {
			habitLogs, err := db.GetHabitLogsForHabit(ctx, int64(habits[i].ID))
			if err != nil {
				log.Println(err)
			}
			templLogs := make([]templates.HabitLog, len(habitLogs))
			for j := range habitLogs {
				templLogs[j].LogDate = time.Unix(habitLogs[j].LogDate, 0)
			}
			templHabit[i] = templates.Habit{
				ID:    int(habits[i].ID),
				Name:  habits[i].Name,
				Exp:   int(habits[i].Exp),
				Level: int(habits[i].Level),
				Logs:  templLogs,
			}
		}
		templates.HabitList(templHabit).Render(ctx, w)
	})

	r.Post("/log-habit/{habitID}", func(w http.ResponseWriter, r *http.Request) {
		habitIDStr := chi.URLParam(r, "habitID")
		habitID, err := strconv.Atoi(habitIDStr)
		if err != nil {
			log.Println(err)
		}

		t := time.Now().Unix()

		h, err := db.GetHabitLogsForHabitWithHabit(ctx, int64(habitID))
		if err != nil {
			log.Println(err)
		}
		if h == nil {
			h = make([]lvl.GetHabitLogsForHabitWithHabitRow, 0)
		}
		dbHabit, err := db.GetHabit(ctx, int64(habitID))
		if err != nil {
			log.Println(err)
		}
		streak := make([]struct{}, 0)
		for _, hb := range h {
			if time.Unix(hb.LogDate, 0).After(time.Now().Add(-72 * time.Hour)) {
				streak = append(streak, struct{}{})
			}
		}
		multiplier := math.Min(float64((len(streak) + 1)), 4)
		dbHabit.Exp += int64(10 * multiplier)
		if dbHabit.Exp > expNeededForLevel(dbHabit.Level) {
			dbHabit.Exp = dbHabit.Exp - expNeededForLevel(dbHabit.Level)
			dbHabit.Level += 1
		}

		dbHabit, err = db.UpdateHabit(ctx, lvl.UpdateHabitParams{
			ID:    int64(habitID),
			Level: dbHabit.Level,
			Exp:   dbHabit.Exp,
		})
		if err != nil {
			log.Println(err)
		}

		newHabit, err := db.CreateHabitLog(ctx, lvl.CreateHabitLogParams{
			HabitID: int64(habitID),
			LogDate: t,
		})
		if err != nil {
			log.Println(err)
		}
		h = append(h, lvl.GetHabitLogsForHabitWithHabitRow{
			ID:      newHabit.ID,
			LogDate: newHabit.LogDate,
			HabitID: newHabit.HabitID,
			Name:    dbHabit.Name,
			Level:   dbHabit.Level,
			Exp:     dbHabit.Exp,
		})
		habitLog := make([]templates.HabitLog, len(h))
		for i, hl := range h {
			habitLog[i].LogDate = time.Unix(hl.LogDate, 0)
		}
		habit := templates.Habit{
			ID:    int(dbHabit.ID),
			Name:  dbHabit.Name,
			Level: int(dbHabit.Level),
			Exp:   int(dbHabit.Exp),
			Logs:  habitLog,
			// TODO: this is not calculated correctly
			StreakCount: len(streak),
		}
		templates.DrawHabit(habit).Render(ctx, w)
	})
	http.ListenAndServe(":8080", r)
}

func expNeededForLevel(level int64) int64 {
	return int64(math.Pow(float64(level), 3.0/2.0) * 100 * 0.75)
}
