# Description
Pooply is a Discord Bot aimed at sharing your regularity with friends.

# Bot Installation
To install Pooply use the following discord link: `https://discord.com/oauth2/authorize?client_id=1327013598551347283`

# Development
Pooply is a golang application running in a docker image. When testing locally you'll require the app id and an application token.

## Running
When running locally you should be able to do something similar to:

```shell
goose postgres "postgres://username:password@localhost:5432/pooply?sslmode=disable" -dir migrations up
go run . -app <app_id> -token <token>
```

## Tools
* Database Schema Migrations - (`goose`)[https://github.com/pressly/goose]

# Contributions
Contributions are welcome, but must come in the form of a pull request against the main branch
