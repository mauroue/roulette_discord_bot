package main

import (
	"container/list"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"syscall"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

// types declaration
type Config struct {
	Token         string `yaml:"token"`
	Target        string `yaml:"target"`
	TargetChannel string `yaml:"target-channel"`
	TargetServer  string `yaml:"target-server"`
}
type User struct {
	id      int64
	name    string
	tickets int
	success int
	fail    int
}

// variable declaration and initialization
var DBCon *sql.DB
var cfg *Config
var counter = 0

func loadConfigFromFile(filename string) error {
	// Load config from specified file and parse using yaml decoder
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	decode := yaml.NewDecoder(file)

	if err := decode.Decode(&cfg); err != nil {
		return err
	}

	return err
}

func main() {
	// Load configuration from a file
	if err := loadConfigFromFile("config.yaml"); err != nil {
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
		log.Fatal(err)
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
	TargetMember, _ := s.GuildMember(cfg.TargetServer, cfg.Target)

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
			log.Fatal(err)
		}
		fmt.Println("User registered ID: ", res)
	case err != nil:
		log.Fatal(err)
	}
	// commands
	if m.ChannelID == cfg.TargetChannel {
		switch {
		case strings.Contains(m.Content, "contagem"):
			// list kicked count for today
			_, _ = s.ChannelMessageSend(m.ChannelID, "Hoje foram chutadas "+strconv.Itoa(counter)+" pessoas, "+m.Author.Mention()+"!")
		case strings.Contains(m.Content, "ticket"):
			// show amount of tickets user have
			query := fmt.Sprintf("SELECT tickets from users where id = %v", m.Author.ID)
			var tickets int
			err := DBCon.QueryRow(query).Scan(&tickets)
			if err != nil {
				log.Fatal(err)
			}
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Você tem %v tickets.", tickets))
		case strings.Contains(m.Content, "quero dar"):
			// target command for giving tickets
			if m.Author.ID == cfg.Target {
				for _, user := range m.Mentions {
					query := fmt.Sprintf("UPDATE users SET tickets = tickets + 1 WHERE id = %v", user.ID)
					_, err := DBCon.Exec(query)
					if err != nil {
						log.Fatal(err)
					}
				}
				_, err := s.ChannelMessageSend(m.ChannelID, "Pronto! Você deu!")
				if err != nil {
					log.Fatal(err)
				}
			} else {
				_, err := s.ChannelMessageSend(m.ChannelID, "Huum, você não é católico suficiente para dar tickets")
				if err != nil {
					log.Fatal(err)
				}
			}
		case strings.Contains(m.Content, "hora do perigo"):
			// check if target is trying to kick himself
			if m.Author.ID == cfg.Target {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Na-na-ni-na-não "+m.Author.Mention()+"!")
				return
			}
			// check if user has enough tickets
			if user.tickets <= 0 {
				_, err := s.ChannelMessageSend(m.ChannelID, "Você não tem tickets suficientes! Implore ao "+TargetMember.Mention()+" por mais tickets!")
				if err != nil {
					log.Fatal(err)
				}
				return
			}
			// removing ticket before rolling dice
			err := updateTickets(user, -1)
			if err != nil {
				log.Fatal(err)
			}

			// Check if target is in voice chat
			voiceState, err := s.State.VoiceState(cfg.TargetServer, cfg.Target)
			if voiceState == nil {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Huuum, infelizmente não é possível chutar o "+TargetMember.Mention()+" sem ele estar em alguma sala 😥")
				return
			}
			if err != nil {
				fmt.Println(err)
			}

			// roll dice and print to channel
			dice := rollTheDices()
			_, _ = s.ChannelMessageSend(m.ChannelID, "O valor lançado foi: "+strconv.Itoa(dice))
			if dice == 10 {
				time.Sleep(2 * time.Second)
				_, _ = s.ChannelMessageSend(m.ChannelID, "Boa "+m.Author.Mention()+" azar hein "+TargetMember.Mention()+"até mais!")
				channel := ""
				data := discordgo.GuildMemberParams{
					ChannelID: &channel,
				}
				_, _ = s.GuildMemberEdit(cfg.TargetServer, cfg.Target, &data)
				counter++
			} else if dice == 1 {
				// user is kicked if rolls critical failure
				time.Sleep(2 * time.Second)
				_, _ = s.ChannelMessageSend(m.ChannelID, "Parece que o jogo virou, não é mesmo?!")
				channel := ""
				data := discordgo.GuildMemberParams{
					ChannelID: &channel,
				}
				_, _ = s.GuildMemberEdit(cfg.TargetServer, m.Author.ID, &data)
			} else {
				time.Sleep(2 * time.Second)
				_, _ = s.ChannelMessageSend(m.ChannelID, "Teve sorte desta vez!")
			}
			return
		}
	}
}

func PrepareDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./database/gustabot.db")
	if err != nil {
		log.Fatal("Erro ao abrir conexão com db: ", err)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Erro ao pingar o banco de dados: ", err)
		db.Close()
		return nil, err
	}

	createTableSql := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			name TEXT,
			tickets INTEGER,
			success INTEGER,
			fail INTEGER
		)
	`
	_, err = db.Exec(createTableSql)
	if err != nil {
		err := db.Close()
		if err != nil {
			return nil, err
		}
		log.Fatal("Erro ao criar tabela: ", err)
		return nil, err
	}

	return db, nil

}

func updateTickets(user User, value int) error {
	_, err := DBCon.Exec(fmt.Sprintf("UPDATE users SET tickets = tickets + %v WHERE id = %v", value, user.id))
	if err != nil {
		return err
	}
	return nil
}
