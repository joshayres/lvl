-- name: GetHabit :one
SELECT * FROM habits
WHERE id = ? LIMIT 1;

-- name: GetHabits :many
SELECT * FROM habits
ORDER BY name;

-- name: CreateHabit :one
INSERT INTO habits (
	name, level, exp
) VALUES (
	?, ?, ?
)
RETURNING *;

-- name: UpdateHabit :one
UPDATE habits
set level = ?,
exp = ?
WHERE id = ?
RETURNING *;

-- name: DeleteHabit :exec
DELETE FROM habits
WHERE id = ?;

-- name: CreateHabitLog :one
INSERT INTO habitlogs (
	habit_id, log_date
) VALUES (
	?, ?
)
RETURNING *;

-- name: GetHabitLog :one
SELECT * FROM habitlogs
WHERE id = ? LIMIT 1;

-- name: GetHabitLogsForHabit :many
SELECT
	id,
	log_date
FROM
	habitlogs
WHERE
	habit_id = ?;

-- name: GetHabitLogsForHabitWithHabit :many
SELECT
	hl.id,
	hl.log_date,
	hl.habit_id,
	h.name,
	h.level,
	h.exp
FROM
	habitlogs AS hl
JOIN
	habits AS h ON hl.habit_id = h.id
WHERE
	hl.habit_id = ?;


-- name: GetHabitLogsWithHabit :many
SELECT
	hl.id,
	hl.log_date,
	hl.habit_id,
	h.name,
	h.level,
	h.exp
FROM
	habitlogs AS hl
JOIN
	habits AS h ON hl.habit_id = h.id;

-- name: GetHabitLogsForHabitWithinLastThreeDays :many
SELECT
	id,
	log_date
FROM
	habitlogs
WHERE
	habit_id = ? AND log_date >= DATE('now', '-3 days');
