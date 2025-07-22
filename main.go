// server.go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Quote struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

type APIResponse struct {
	USDBRL Quote `json:"USDBRL"`
}

func cotacaoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Timeout para chamada à API
	ctxAPI, cancelAPI := context.WithTimeout(ctx, 20000*time.Millisecond)
	defer cancelAPI()

	reqAPI, err := http.NewRequestWithContext(ctxAPI, http.MethodGet, "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		log.Printf("erro criando request: %v", err)
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}

	respAPI, err := http.DefaultClient.Do(reqAPI)
	if err != nil {
		log.Printf("erro ao chamar API de câmbio: %v", err)
		http.Error(w, "erro ao obter cotação", http.StatusGatewayTimeout)
		return
	}
	defer respAPI.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(respAPI.Body).Decode(&apiResp); err != nil {
		log.Printf("erro decodificando resposta da API: %v", err)
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}

	quote := apiResp.USDBRL

	// Persistência no SQLite
	db, err := sql.Open("sqlite3", "cotacoes.db")
	if err != nil {
		log.Printf("erro ao abrir banco: %v", err)
	} else {
		defer db.Close()

		ctxDB, cancelDB := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancelDB()

		// garante tabela
		db.ExecContext(ctxDB, `CREATE TABLE IF NOT EXISTS cotacao (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bid TEXT,
			timestamp DATETIME
		)`)

		_, err = db.ExecContext(ctxDB,
			"INSERT INTO cotacao(bid, timestamp) VALUES(?, ?)",
			quote.Bid, time.Now(),
		)
		if err != nil {
			log.Printf("erro ao inserir cotação no banco: %v", err)
		}
	}

	// Retorna apenas o JSON do quote
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(quote); err != nil {
		log.Printf("erro ao escrever resposta: %v", err)
	}
}

func main() {
	http.HandleFunc("/cotacao", cotacaoHandler)
	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("falha ao iniciar servidor: %v", err)
	}
}
