package main

import (
	"container/list"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"time"

	//"database/sql"
	"syscall"

	"github.com/bwmarrin/discordgo"
	//_ "github.com/mattn/go-sqlite3"
)

var DBCon *sql.DB
var Token string

type User struct {
	id      int
	name    string
	tickets int
	success int
	fail    int
}

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

var counter = 0

func main() {
	discord, err := discordgo.New("Bot " + "MTE1NzQ5MTg5Nzk4MDQyNDIwMg.GbJfoK.UM_2yGvXRkIeGdQQW0i_i_Vlz0HK6hd1ipH1dQ")
	if err != nil {
		fmt.Println("Error creating session", err)
		return
	}

	discord.AddHandler(messageCreate)
	discord.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	//DBCon, err := PrepareDb()
	//if err != nil {
	//	log.Fatal(err)
	//} else {
	//	log.Println("Banco inicializado com sucesso.", DBCon.Stats())
	//}

	err = discord.Open()
	if err != nil {
		log.Fatal("Erro ao abrir conexão com banco: ", err)
		return
	}

	fmt.Println("Bot rodando, pressione CTRL + C para sair.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	CloseErr := discord.Close()
	if err != nil {
		log.Println(CloseErr)
	}
}

func messageCreate(s *discordgo.Session, message *discordgo.MessageCreate) {

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
	maxNum := 11
	result := rand.Intn(maxNum) + 1
	return result
}

func mainRoutine(s *discordgo.Session, m *discordgo.MessageCreate) {
	TARGET := "154562406809468928"
	SERVER := "919946669745905714"
	SalaDoBot := "919946669745905716"
	// TARGET := "477619741733683217"
	//SalaDoBot := "1158492378534002800"
	//SERVER := "788516492818776086"
	TargetMember, _ := s.GuildMember(SERVER, TARGET)

	// ignore bot messages
	if m.Author.ID == s.State.User.ID {
		fmt.Println("ignoring bot message")
		return
	}

	// check if user is registered in database
	var authorId string
	query := "SELECT id FROM users WHERE id=?"
	rows, err := DBCon.Query(query, m.Author.ID)
	if err != nil {
		fmt.Println(err)
	}
	for rows.Next() {
		var user User
		err = rows.Scan(&user)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(user)
	}
	fmt.Println(authorId)
	//if err == sql.ErrNoRows {
	//	fmt.Println("this is where we create")
	//}
	if m.ChannelID == SalaDoBot && m.Content == "contagem" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Hoje foram chutadas "+strconv.Itoa(counter)+" pessoas, "+m.Author.Mention()+"!")
		return
	}

	if m.ChannelID == SalaDoBot && m.Content == "hora do perigo" {

		// if m.Author.ID == TARGET {
		// 	_, _ = s.ChannelMessageSend(m.ChannelID, "Na-na-ni-na-não "+m.Author.Mention()+"!")
		// 	return
		// }
		voiceState, err := s.State.VoiceState(SERVER, TARGET)
		if voiceState == nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Huuum, infelizmente não é possível chutar o "+TargetMember.Mention()+" sem ele estar em alguma sala 😥")
			return
		}
		if err != nil {
			fmt.Println(err)
		}
		dice := rollTheDices()
		_, _ = s.ChannelMessageSend(m.ChannelID, "O valor lançado foi: "+strconv.Itoa(dice))
		if dice == 1 {
			time.Sleep(2 * time.Second)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Boa "+m.Author.Mention()+" azar hein "+TargetMember.Mention()+"até mais!")
			channel := ""
			data := discordgo.GuildMemberParams{
				ChannelID: &channel,
			}
			_, _ = s.GuildMemberEdit(SERVER, TARGET, &data)
			counter++
		} else if dice == 12 {
			time.Sleep(2 * time.Second)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Parece que o jogo virou, não é mesmo?!")
			channel := ""
			data := discordgo.GuildMemberParams{
				ChannelID: &channel,
			}
			_, _ = s.GuildMemberEdit(SERVER, m.Author.ID, &data)
		} else {
			time.Sleep(2 * time.Second)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Teve sorte desta vez!")
		}
		return
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
		db.Close()
		log.Fatal("Erro ao criar tabela: ", err)
		return nil, err
	}

	return db, nil

}
