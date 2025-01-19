package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type Message struct {
	ID        int64     `bun:",pk,autoincrement"` // primary key, auto-increment
	Author    string    `bun:"author,notnull"`
	Input     string    `bun:"input,notnull"`
	CreatedAt time.Time `bun:"created_at,nullzero,default:CURRENT_TIMESTAMP"`
}

type Log struct {
	ID      uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	Author  string    `bun:"author,notnull"`
	Started time.Time `bun:"started,nullzero,default:CURRENT_TIMESTAMP"`
	Ended   time.Time `bun:"ended,nullzero"`
}

type optionMap = map[string]*discordgo.ApplicationCommandInteractionDataOption

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) (om optionMap) {
	om = make(optionMap)
	for _, opt := range options {
		om[opt.Name] = opt
	}
	return
}

func interactionAuthor(i *discordgo.Interaction) *discordgo.User {
	if i.Member != nil {
		return i.Member.User
	}
	return i.User
}

func handleEcho(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
	builder := new(strings.Builder)
	if v, ok := opts["author"]; ok && v.BoolValue() {
		author := interactionAuthor(i.Interaction)
		builder.WriteString("**" + author.String() + "** says: ")
	}
	builder.WriteString(opts["message"].StringValue())

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: builder.String(),
		},
	})
	if err != nil {
		log.Fatalf("could not respond to interaction: %s\n", err)
	}
	log.Println("Successfully sent echo message.")
}

func handleInsert(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap, ctx context.Context, db *bun.DB) error {
	author := interactionAuthor(i.Interaction).String()
	input := opts["input"].StringValue()

	dataToInsert := []Message{
		{Author: author, Input: input},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Added to the database!",
		},
	})
	if err != nil {
		log.Fatalf("could not respond to interaction: %s\n", err)
	}
	log.Println("Successfully sent input message.")

	_, dbErr := db.NewInsert().
		Model(&dataToInsert).
		Exec(ctx)
	return dbErr
}

func handleLog(s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context, db *bun.DB) {
	author := interactionAuthor(i.Interaction).String()

	dataToInsert := []Log{{Author: author}}

	_, dbErr := db.NewInsert().
		Model(&dataToInsert).
		Exec(ctx)

	userMsg := "Successfully logged"
	if dbErr != nil {
		userMsg = fmt.Sprintf("handleLog - Failed to insert log into database. author=%s err=%v\n", author, dbErr)
		log.Fatalln(userMsg)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: userMsg,
		},
	})
	if err != nil {
		log.Fatalf("handleLog - Failed to respond to interaction. author=%s err=%v\n", author, err)
	}
}

func setupSchema(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().
		Model((*Message)(nil)).
		IfNotExists().
		Exec(ctx)
	return err
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "echo",
		Description: "Say something through a bot",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "message",
				Description: "Contents of the message",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
			{
				Name:        "author",
				Description: "Whether to prepend message's author",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
		},
	},
	{
		Name:        "insert",
		Description: "Add information to the database",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "input",
				Description: "Contents of the information to add",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	},
	{
		Name:        "log",
		Description: "Log your log",
	},
}

func main() {
	ctx := context.Background()

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", AppConfig.Postgres.User, AppConfig.Postgres.Password, AppConfig.Postgres.Host, AppConfig.Postgres.Port, AppConfig.Postgres.Database)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	// Check the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to PostgreSQL. url=%s err=%v", dsn, err)
	}
	log.Println("Connected to PostgreSQL successfully!")

	session, _ := discordgo.New("Bot " + AppConfig.AuthToken)

	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		switch data.Name {
		case "echo":
			handleEcho(s, i, parseOptions(data.Options))
		case "insert":
			handleInsert(s, i, parseOptions(data.Options), ctx, db)
		case "log":
			handleLog(s, i, ctx, db)
		default:
			log.Fatalf("Command not implemented. command=%s\n", data.Name)
		}
	})

	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as %s", r.User.String())
	})

	_, err := session.ApplicationCommandBulkOverwrite(AppConfig.AppID, "", commands)
	if err != nil {
		log.Fatalf("could not register commands: %s", err)
	}

	err = session.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = session.Close()
	if err != nil {
		log.Printf("could not close session gracefully: %s", err)
	}
}
