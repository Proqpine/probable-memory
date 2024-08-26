-- name: QueryActivities :many
select * from activities;

-- name: QueryActivityByProject :one
select * from activities where project=?;

-- name: InsertActivity :one
insert into activities (start_time, end_time, duration, activity_name, description, project, notes) values (?, ?, ?, ?, ?, ?, ?) returning *;
