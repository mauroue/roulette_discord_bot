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

func main() {
	// Load configuration from a file
	if err := LoadConfigFromFile("config.yaml"); err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
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
		log.Fatal("Erro ao inicializar banco de dados: ", err)
	} else {
		log.Println("Banco inicializado com sucesso: ", DBCon.Stats())
	}

	// start connection with discord servers
	err = discord.Open()
	if err != nil {
		log.Fatal("Erro ao abrir conexão com discord: ", err)
		return
	}

	// console message and exit keybind
	fmt.Println("Bot rodando, pressione CTRL + C para sair.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	CloseErr := discord.Close()
	if CloseErr != nil {
		log.Println(CloseErr)
	}
}

func messageCreate(s *discordgo.Session, message *discordgo.MessageCreate) {
	/* 	get the command received and put it in a fifo queue to be processed in order */
	queue := list.New()
	queue.PushBack(message)
	for queue.Len() > 0 {
		listElement := queue.Front()
		m := listElement.Value.(*discordgo.MessageCreate)
		mainRoutine(s, m)
		queue.Remove(listElement)
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
		createUserQuery := fmt.Sprintf("INSERT INTO users(id,name,tickets,success,fail) VALUES ('%s','%s','0','0','0');", m.Author.ID, m.Author.Username)
		fmt.Println(createUserQuery)
		res, err := DBCon.Exec(createUserQuery)
		if err != nil {
			log.Fatal("Error creating user: ", err)
		}
		fmt.Println("User registered ID: ", res)
	case err != nil:
		log.Fatal(err)
	}
	// commands
	if m.ChannelID == cfg.TargetChannel {
		runCommand(s, m, user)
	}
}

func updateTickets(user User, value int) error {
	_, err := DBCon.Exec(fmt.Sprintf("UPDATE users SET tickets = tickets + %v WHERE id = %v", value, user.id))
	if err != nil {
		return err
	}
	return nil
}

func logHistory(authorid string, targetid string, command string, success bool, roll int) {
	log.Println(authorid, targetid, command, success, roll)
	query := fmt.Sprintf(`
		INSERT INTO history(user,target,command,success,roll) VALUES('%v','%v','%v','%v','%v')
	`, authorid, targetid, command, success, roll)
	_, err := DBCon.Exec(query)
	if err != nil {
		log.Fatalln("Log history failed: ", err)
	}
}
