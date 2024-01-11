package main

import (
	"container/list"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	id      int64
	name    string
	tickets int
	success int
	fail    int
}

const (
	defaultTickets = 0
	defaultSuccess = 0
	defaultFail    = 0
)

func main() {
	// Load configuration from a file
	if err := LoadConfigFromFile("config.yaml"); err != nil {
		log.Fatalf("Error loading config: %v", err)
		return
	}
	// get the config values and put it in a variable cfg
	token := "Bot " + cfg.Token
	log.Println("Token is: ", cfg.Token)
	log.Println("Channel is: ", cfg.TargetChannel)
	log.Println("Server is: ", cfg.TargetServer)
	log.Println("Target is: ", cfg.Target)

	// initialize discord session
	discord, err := discordgo.New(token)
	if err != nil {
		fmt.Println("Erro ao criar sessão: ", err)
		return
	}
	// create a message handler instance and its privileges
	discord.AddHandler(messageCreate)
	discord.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	// initialize database and prepare it if its the first time being ran
	DBCon, err = PrepareDb()
	if err != nil {
		log.Fatal("Error initializing database: ", err)
	} else {
		log.Println("Database initialized succesfully: ", DBCon.Stats())
	}

	// start connection with discord servers
	err = discord.Open()
	if err != nil {
		log.Fatal("Error connecting with discord API: ", err)
		return
	}

	// console message and exit keybind
	fmt.Println("Bot is running. Press CTRL + C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	CloseErr := discord.Close()
	if CloseErr != nil {
		log.Println(CloseErr)
	}
}

var commandQueue = list.New()

func messageCreate(s *discordgo.Session, message *discordgo.MessageCreate) {
	/* 	get the command received and put it in a fifo queue to be processed in order */
	commandQueue.PushBack(message)
	for commandQueue.Len() > 0 {
		listElement := commandQueue.Front()
		m := listElement.Value.(*discordgo.MessageCreate)
		mainRoutine(s, m)
		commandQueue.Remove(listElement)
	}
}

func rollTheDices() int {
	/* 	roll a 10 sided dice */
	maxNum := 9
	result := rand.Intn(maxNum) + 1
	return result
}

func mainRoutine(s *discordgo.Session, m *discordgo.MessageCreate) {

	// ignore bot messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	// check if user is registered in database
	query := fmt.Sprintf("SELECT * FROM users WHERE id = %v", m.Author.ID)
	var user User
	err := DBCon.QueryRow(query).Scan(&user.id, &user.name, &user.tickets, &user.success, &user.fail)
	switch {
	case err == sql.ErrNoRows:
		fmt.Println("User not found, registering...")
		createUserQuery := "INSERT INTO users(id, name, tickets, success, fail) VALUES (?, ?, ?, ?, ?);"
		res, err := DBCon.Exec(createUserQuery, m.Author.ID, m.Author.Username, defaultTickets, defaultSuccess, defaultFail)
		if err != nil {
			log.Fatal("Error creating user: ", err)
		}
		fmt.Println("User registered ID: ", res)
	case err != nil:
		log.Fatal(err)
	}
	// parse message and run command from message
	if m.ChannelID == cfg.TargetChannel {
		runCommand(s, m, user)
	}
}

func updateTickets(user User, value int) error {
	ticketQuery := "UPDATE users SET tickets = tickets + ? WHERE id = ?"
	_, err := DBCon.Exec(ticketQuery, value, user.id)
	if err != nil {
		log.Println("Error updating tickets: ", err)
	}
	return nil
}

func logHistory(authorid string, targetid string, command string, success bool, roll int) {
	historyQuery := "INSERT INTO history(user,target,command,success,roll) VALUES (?, ?, ?, ?, ?);"
	_, err := DBCon.Exec(historyQuery, authorid, targetid, command, success, roll)
	if err != nil {
		log.Println("Log history failed: ", err)
	}
}
