package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"strconv"
	"strings"
	"time"
)

func runCommand(s *discordgo.Session, m *discordgo.MessageCreate, user User) {
	TargetMember, _ := s.GuildMember(cfg.TargetServer, cfg.Target)
	switch {
	case strings.Contains(m.Content, "contagem"):
		_, _ = s.ChannelMessageSend(m.ChannelID, "Função desabilitada temporariamente.")
	case strings.Contains(m.Content, ShowTicketsCommand):
		// show amount of tickets user have
		query := fmt.Sprintf("SELECT tickets from users where id = %v", m.Author.ID)
		var tickets int
		err := DBCon.QueryRow(query).Scan(&tickets)
		if err != nil {
			logHistory(m.Author.ID, m.Author.ID, ShowTicketsCommand, false, 0)
			log.Fatal("Tickets command failed: ", err)
		}
		logHistory(m.Author.ID, m.Author.ID, ShowTicketsCommand, true, 0)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Você tem %v tickets.", tickets))
	case strings.Contains(m.Content, GiveTicketsCommand):
		// target command for giving tickets
		if m.Author.ID == cfg.Target {
			for _, user := range m.Mentions {
				query := fmt.Sprintf("UPDATE users SET tickets = tickets + 1 WHERE id = %v", user.ID)
				_, err := DBCon.Exec(query)
				if err != nil {
					log.Fatal(err)
				}
				logHistory(m.Author.ID, user.ID, GiveTicketsCommand, true, 0)
			}
			_, err := s.ChannelMessageSend(m.ChannelID, "Pronto! Você deu!")
			if err != nil {
				log.Fatal("Erro trying to send message to channel: ", err)
			}
		} else {
			for _, user := range m.Mentions {
				logHistory(m.Author.ID, user.ID, GiveTicketsCommand, false, 0)
			}
			_, err := s.ChannelMessageSend(m.ChannelID, "Huum, você não é católico suficiente para dar tickets")
			if err != nil {
				log.Fatal("Error trying to send message to channel: ", err)
			}
		}
	case strings.Contains(m.Content, RollDiceCommand):
		// check if target is trying to kick himself
		if m.Author.ID == cfg.Target {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Na-na-ni-na-não "+m.Author.Mention()+"!")
			logHistory(m.Author.ID, m.Author.ID, RollDiceCommand, false, 0)
			return
		}
		// check if user has enough tickets
		if user.tickets <= 0 {
			_, err := s.ChannelMessageSend(m.ChannelID, "Você não tem tickets suficientes! Implore ao "+TargetMember.Mention()+" por mais tickets!")
			logHistory(m.Author.ID, TargetMember.User.ID, RollDiceCommand, false, 0)
			if err != nil {
				log.Fatal("Erro ao postar no canal: ", err)
			}
			return
		}

		// Check if target is in voice chat
		userState, err := s.State.VoiceState(cfg.TargetServer, m.Author.ID)
		if err != nil {
			log.Fatal("Erro ao acessar o canal de voz: ", err)
		}
		fmt.Printf(userState.ChannelID)
		voiceState, err := s.State.VoiceState(cfg.TargetServer, cfg.Target)
		if err != nil {
			log.Fatal("Erro ao acessar o canal de voz: ", err)
		}
		if voiceState == nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Huuum, infelizmente não é possível chutar o "+TargetMember.Mention()+" sem ele estar em alguma sala 😥")
			logHistory(m.Author.ID, TargetMember.User.ID, RollDiceCommand, false, 0)
			return
		}

		// removing ticket before rolling dice
		err = updateTickets(user, -1)
		if err != nil {
			log.Fatal("Erro ao atualizar tickets: ", err)
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
	case strings.Contains(m.Content, RollDiceCommand):
		tickets := QueryTickets(m.Author.ID)
		message := fmt.Sprintf("%v tem %v tickets restantes.", m.Author.Mention(), tickets)
		_, err := s.ChannelMessageSend(m.ChannelID, message)
		if err != nil {
			log.Fatal(err)
		}
	}

}

func QueryTickets(userID string) int {
	query := fmt.Sprintf("SELECT tickets from users where id = %v", userID)
	var tickets int
	err := DBCon.QueryRow(query).Scan(&tickets)
	if err != nil {
		log.Fatal("Query Tickets", err)
	}
	return tickets
}
