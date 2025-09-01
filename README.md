# Desafio Go: Cotação do Dólar - Client & Server

Este projeto implementa dois sistemas em Go que se comunicam via HTTP para consultar e registrar a cotação do dólar em relação ao real. O desafio envolve uso de webserver, contextos, banco de dados SQLite e manipulação de arquivos.

## Estrutura

- `Server/server.go`: Servidor HTTP que expõe o endpoint `/cotacao` na porta 8080.
- `Client/client.go`: Cliente que consome o endpoint `/cotacao` e salva o valor da cotação em um arquivo.

## Funcionamento

### Server

- O servidor consulta a API [AwesomeAPI](https://economia.awesomeapi.com.br/json/last/USD-BRL) para obter a cotação do dólar.
- Utiliza `context` para limitar o tempo de chamada da API a 200ms.
- Persiste cada cotação recebida em um banco SQLite (`desafio_cotacao.db`), com timeout de 10ms para a operação de escrita.
- Retorna ao cliente o resultado em JSON.

### Client

- O cliente faz uma requisição HTTP ao servidor (`/cotacao`), com timeout de 300ms usando `context`.
- Recebe apenas o valor atual do câmbio (`bid`) e salva em um arquivo `cotacao.txt` no formato:  
