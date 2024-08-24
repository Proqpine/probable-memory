-- name: QueryActivities :many
select * from activities;

-- name: QueryActivityByProject :one
select * from activities where project=?;

-- name: InsertActivity :one
insert into activities (start_time, activity_name, description, project, notes) values ()
-- TODO here
