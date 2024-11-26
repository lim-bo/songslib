CREATE DATABASE songs_lib;

CREATE TABLE groups (
    id serial primary key,
    name text not null unique
)

CREATE TABLE songs (
    id SERIAL PRIMARY KEY,
    name text not null,
    group not null references groups(name),
    release_date text not null,
    lyrics text not null default "",
    link text not null
);