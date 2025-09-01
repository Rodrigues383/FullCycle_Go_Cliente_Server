package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type cotacao struct {
	Name string `json:"name"`
	Bid  string `json:"bid"`
}

func main() {
	// Permite customizar a URL do endpoint via flag
	url := flag.String("url", "http://localhost:8080/cotacao", "URL do endpoint /cotacao")
	timeout := flag.Duration("timeout", 3*time.Second, "timeout da requisição")
	jsonOut := flag.Bool("json", false, "imprimir saída em JSON (true) ou texto (false)")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *url, nil)
	if err != nil {
		fail("criando request", err)
	}

	client := &http.Client{Timeout: *timeout}
	resp, err := client.Do(req)
	if err != nil {
		fail("chamando endpoint", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fail("status inesperado", fmt.Errorf("HTTP status: %s", resp.Status))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fail("lendo corpo da resposta", err)
	}

	// 1) tenta decodificar como objeto simples
	var c cotacao
	if err := json.Unmarshal(body, &c); err == nil && (c.Bid != "" || c.Name != "") {
		printResult(c, *jsonOut)
		return
	}

	// 2) tenta decodificar como mapa (ex.: {"USDBRL": {...}})
	var m map[string]cotacao
	if err := json.Unmarshal(body, &m); err == nil {
		for _, v := range m { // pega o primeiro valor
			printResult(v, *jsonOut)
			return
		}
	}

	fail("formato JSON não reconhecido", fmt.Errorf("esperado objeto com campos name/bid ou mapa de cotações"))
}

func printResult(c cotacao, jsonOut bool) {
	if jsonOut {
		_ = json.NewEncoder(os.Stdout).Encode(struct {
			Name string `json:"name"`
			Bid  string `json:"bid"`
		}{Name: c.Name, Bid: c.Bid})
		return
	}
	fmt.Printf("name: %s\nbid: %s\n", c.Name, c.Bid)

	// Salva em arquivo
	f, err := os.Create("cotacao.txt")
	if err != nil {
		fmt.Printf("erro ao criar arquivo: %v", err)
	}
	defer f.Close()

	text := fmt.Sprintf("Dólar: %s", c.Bid)
	if _, err := f.WriteString(text); err != nil {
		fmt.Printf("erro ao escrever no arquivo: %v", err)
	}

	fmt.Printf("Cotação salva em cotacao.txt")
}

func fail(msg string, err error) {
	fmt.Fprintf(os.Stderr, "erro %s: %v\n", msg, err)
	os.Exit(1)
}
