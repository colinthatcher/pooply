APP_ID?=
TOKEN?=
POSTGRES_USER?=
POSTGRES_PASSWORD?=
POSTGRES_DB?=pooply
POSTGRES_HOST?=localhost

include .env

run:
	go run . -app ${APP_ID} -token ${TOKEN}

migrate:
	goose postgres "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:5432/${POSTGRES_DB}?sslmode=disable" -dir migrations up
