package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
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
		log.Panicf("could not respond to interaction: %s", err)
	}
	log.Println("Successfully sent echo message.")
}

func handleInsert(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap, ctx context.Context, db *bun.DB) error {
	author := interactionAuthor(i.Interaction).String()
	input := opts["message"].StringValue()

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
		log.Panicf("could not respond to interaction: %s", err)
	}
	log.Println("Successfully sent input message.")

	_, dbErr := db.NewInsert().
		Model(&dataToInsert).
		Exec(ctx)
	return dbErr
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
}

var (
	Token = flag.String("token", "", "Bot authentication token")
	App   = flag.String("app", "", "Application ID")
	Guild = flag.String("guild", "", "Guild ID")
)

func main() {
	dsn := "postgres://postgres:mathilenjoyer@localhost:5432/app_db?sslmode=disable"

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	ctx := context.Background()

	db := bun.NewDB(sqldb, pgdialect.New())

	defer db.Close()

	// Check the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}

	if err := setupSchema(context.Background(), db); err != nil {
		log.Fatalf("failed to setup schema: %v", err)
	}

	log.Println("Connected to PostgreSQL successfully!")

	flag.Parse()
	if *App == "" {
		log.Fatal("application id is not set")
	}

	session, _ := discordgo.New("Bot " + *Token)

	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		if data.Name != "echo" && data.Name != "insert" {
			return
		}

		if data.Name == "echo" {
			handleEcho(s, i, parseOptions(data.Options))
		}

		if data.Name == "insert" {
			handleInsert(s, i, parseOptions(data.Options), ctx, db)
		}

	})

	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as %s", r.User.String())
	})

	_, err := session.ApplicationCommandBulkOverwrite(*App, *Guild, commands)
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
