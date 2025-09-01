package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)

type Cotacao struct {
	Code       string  `json:"code"`
	Codein     string  `json:"codein"`
	Name       string  `json:"name"`
	High       float64 `json:"high,string"`
	Low        float64 `json:"low,string"`
	VarBid     string  `json:"varBid"`
	PctChange  float64 `json:"pctChange,string"`
	Bid        string  `json:"bid"`
	Ask        string  `json:"ask"`
	Timestamp  int64   `json:"timestamp,string"` // epoch segundos (vem como string)
	CreateDate string  `json:"create_date"`
}

func main() {
	db, err := conectarBanco("./desafio_cotacao.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := criarTabelaCotacao(db); err != nil {
		log.Fatal(err)
	}

	// registra a rota /cotacao
	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
		defer cancel()

		cot, err := getCotacaoFromAPI(ctx)
		if err != nil {
			http.Error(w, "erro ao obter cotação", http.StatusBadGateway)
			return
		}

		// também insere no banco
		_ = inserirCotacao(db, *cot)

		w.Header().Set("Content-Type", "application/json")
		//json.NewEncoder(w).Encode("{'teste':'val 1', 'teste2':'val 2'}")
		json.NewEncoder(w).Encode(cot)

	})

	log.Println("Servidor rodando em http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

	if err := listarCotacoes(db); err != nil {
		log.Fatal("Erro ao listar cotações:", err)
	}
}

func conectarBanco(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("abrindo banco: %w", err)
	}
	_, _ = db.Exec(`PRAGMA journal_mode=WAL;`)
	_, _ = db.Exec(`PRAGMA busy_timeout=5000;`)
	return db, nil
}

func criarTabelaCotacao(db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS cotacao (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	Code        TEXT,
	Codein      TEXT,
	Name        TEXT,
	High        REAL,
	Low         REAL,
	VarBid      TEXT,
	PctChange   REAL,
	Bid         TEXT,
	Ask         TEXT,
	Timestamp   INTEGER,     -- epoch (segundos)
	CreateDate  TEXT
);`
	_, err := db.Exec(ddl)
	return err
}

func inserirCotacao(db *sql.DB, c Cotacao) error {
	const q = `
INSERT INTO cotacao (Code, Codein, Name, High, Low, VarBid, PctChange, Bid, Ask, Timestamp, CreateDate)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(q,
		c.Code, c.Codein, c.Name, c.High, c.Low, c.VarBid, c.PctChange, c.Bid, c.Ask, c.Timestamp, c.CreateDate,
	)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}
	return nil
}

func listarCotacoes(db *sql.DB) error {
	rows, err := db.Query(`SELECT Code, Codein, Name, High, Low FROM cotacao ORDER BY id DESC LIMIT 10`)
	if err != nil {
		return fmt.Errorf("query list: %w", err)
	}
	defer rows.Close()

	fmt.Println("\nÚltimas cotações:")
	for rows.Next() {
		var code, codein, name string
		var high, low float64
		if err := rows.Scan(&code, &codein, &name, &high, &low); err != nil {
			log.Println("Scan:", err)
			continue
		}
		fmt.Printf("- %s/%s | %s | High=%.4f Low=%.4f\n", code, codein, name, high, low)
	}
	return rows.Err()
}

// ----------- Lógica pura de busca na API (sem ResponseWriter) ----------------

func getCotacaoFromAPI(ctx context.Context) (*Cotacao, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, fmt.Errorf("nova request: %w", err)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chamando API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status inesperado: %s", resp.Status)
	}

	var m map[string]Cotacao
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("decodificando json: %w", err)
	}
	cot, ok := m["USDBRL"]
	if !ok {
		return nil, errors.New("chave USDBRL ausente")
	}
	return &cot, nil
}
