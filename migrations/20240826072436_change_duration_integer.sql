-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
create table if not exists activities(
    id integer primary key,
    start_time timestamp not null,
    end_time timestamp,
    duration integer,
    activity_name varchar(255) not null,
    description varchar(255) not null,
    project varchar(255) not null,
    notes varchar(255) not null
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
drop table if exists activities;
-- +goose StatementEnd
