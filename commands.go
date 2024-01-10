package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func runCommand(s *discordgo.Session, m *discordgo.MessageCreate, user User) {
	targetMember, _ := s.GuildMember(cfg.TargetServer, cfg.Target)
	switch {
	case strings.Contains(m.Content, ShowTicketsCommand):
		// show amount of tickets user have
		query := fmt.Sprintf("SELECT tickets from users where id = %v", m.Author.ID)
		var tickets int
		err := DBCon.QueryRow(query).Scan(&tickets)
		if err != nil {
			logHistory(m.Author.ID, m.Author.ID, ShowTicketsCommand, false, 0)
			log.Println("Tickets command failed: ", err)
		}
		logHistory(m.Author.ID, m.Author.ID, ShowTicketsCommand, true, 0)
		message := fmt.Sprintf(ShowTicketsMessage, tickets)
		SendMessageChannel(s, m, message)
	case strings.Contains(m.Content, GiveTicketsCommand):
		// target command for giving tickets
		if m.Author.ID == cfg.Target {
			for _, user := range m.Mentions {
				re := regexp.MustCompile("[0-9]+")
				numberOfTickets := re.FindAllString(m.Message.Content, 2)
				query := fmt.Sprintf("UPDATE users SET tickets = tickets + %v WHERE id = %v", numberOfTickets, user.ID)
				_, err := DBCon.Exec(query)
				if err != nil {
					log.Println(err)
				}
				logHistory(m.Author.ID, user.ID, GiveTicketsCommand, true, 0)
			}

			SendMessageChannel(s, m, GiveTicketsMessage)
		} else {
			for _, user := range m.Mentions {
				logHistory(m.Author.ID, user.ID, GiveTicketsCommand, false, 0)
			}
			_, err := s.ChannelMessageSend(m.ChannelID, "Huum, você não é católico suficiente para dar tickets")
			if err != nil {
				log.Println("Error trying to send message to channel: ", err)
			}
		}
	case strings.Contains(m.Content, RollDiceCommand):
		// check if target is trying to kick himself
		if m.Author.ID == cfg.Target {
			message := fmt.Sprintf(TargetKickHimselfMessage, m.Author.Mention())
			SendMessageChannel(s, m, message)
			logHistory(m.Author.ID, m.Author.ID, RollDiceCommand, false, 0)
			return
		}
		// check if user has enough tickets
		if user.tickets <= 0 {
			message := fmt.Sprintf(UserBegsForTicketMessage, targetMember.Mention())
			SendMessageChannel(s, m, message)
			logHistory(m.Author.ID, targetMember.User.ID, RollDiceCommand, false, 0)
			return
		}

		// Check if target is in voice chat
		userState, err := s.State.VoiceState(cfg.TargetServer, m.Author.ID)
		if err != nil {
			log.Println("Error trying to access voice channel: ", err)
		}
		fmt.Printf(userState.ChannelID)
		voiceState, err := s.State.VoiceState(cfg.TargetServer, cfg.Target)
		if err != nil {
			log.Println("Error trying to access voice channel: ", err)
		}
		if voiceState == nil {
			message := fmt.Sprintf(TargetNotFoundMessage, targetMember.Mention())
			logHistory(m.Author.ID, targetMember.User.ID, RollDiceCommand, false, 0)
			SendMessageChannel(s, m, message)
			return
		}

		// removing ticket before rolling dice
		err = updateTickets(user, -1)
		if err != nil {
			log.Println("Error trying to update tickets: ", err)
		}

		// roll dice and print to channel
		dice := rollTheDices()
		message := fmt.Sprintf(RollDiceResultMessage, strconv.Itoa(dice))
		SendMessageChannel(s, m, message)
		if dice == 10 {
			message := fmt.Sprintf(DiceSuccessMessage, m.Author.Mention(), targetMember.Mention())
			SendMessageChannel(s, m, message)
			channel := ""
			data := discordgo.GuildMemberParams{
				ChannelID: &channel,
			}
			_, _ = s.GuildMemberEdit(cfg.TargetServer, cfg.Target, &data)
		} else if dice == 1 {
			// user is kicked if rolls critical failure
			SendMessageChannel(s, m, DiceCriticalFailureMessage)
			channel := ""
			data := discordgo.GuildMemberParams{
				ChannelID: &channel,
			}
			_, _ = s.GuildMemberEdit(cfg.TargetServer, m.Author.ID, &data)
		} else {
			SendMessageChannel(s, m, DiceFailureMessage)
		}
	case strings.Contains(m.Content, RollDiceCommand):
		tickets := QueryTickets(m.Author.ID)
		message := fmt.Sprintf(TicketCountMessage, m.Author.Mention(), tickets)
		SendMessageChannel(s, m, message)
	}

}

func QueryTickets(userID string) int {
	query := fmt.Sprintf("SELECT tickets from users where id = %v", userID)
	var tickets int
	err := DBCon.QueryRow(query).Scan(&tickets)
	if err != nil {
		log.Println("Query Tickets", err)
	}
	return tickets
}

func SendMessageChannel(session *discordgo.Session, channel *discordgo.MessageCreate, message string) {
	_, err := session.ChannelMessageSend(channel.ChannelID, message)
	if err != nil {
		log.Println("Error trying to post message in channel", err)
	}
}
