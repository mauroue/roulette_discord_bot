package main

import (
	"database/sql"
	"log"
	"log/slog"
)

var DBCon *sql.DB

func PrepareDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		log.Fatal("Erro ao abrir conexão com db: ", err)
		return nil, err
	} else {
		slog.Info("DB connection opened: ", db.Stats())
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
	createHistoryTable := `
		CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY,
			user INTEGER NOT NULL,
			target INTEGER,
			command TEXT,
			success BOOLEAN,
			roll INTEGER,
			time DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err = db.Exec(createTableSql)
	if err != nil {
		err := db.Close()
		if err != nil {
			return nil, err
		}
		log.Fatal("Erro ao criar tabela de usuario: ", err)
		return nil, err
	}
	_, err = db.Exec(createHistoryTable)
	if err != nil {
		err := db.Close()
		if err != nil {
			return nil, err
		}
		log.Fatal("Erro ao criar tabela de historico: ", err)
		return nil, err
	}

	return db, nil
}
