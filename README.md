# Desafio rate-limiter

## Descrição Desafio

Objetivo: Desenvolver um rate limiter em Go que possa ser configurado para limitar o número máximo de requisições por segundo com base em um endereço IP específico ou em um token de acesso.

Descrição: O objetivo deste desafio é criar um rate limiter em Go que possa ser utilizado para controlar o tráfego de requisições para um serviço web. O rate limiter deve ser capaz de limitar o número de requisições com base em dois critérios:

1. **Endereço IP:** O rate limiter deve restringir o número de requisições recebidas de um único endereço IP dentro de um intervalo de tempo definido.
2. **Token de Acesso:** O rate limiter deve também poderá limitar as requisições baseadas em um token de acesso único, permitindo diferentes limites de tempo de expiração para diferentes tokens. O Token deve ser informado no header no seguinte formato:
    1. API_KEY: \<TOKEN\>
3. As configurações de limite do token de acesso devem se sobrepor as do IP. Ex: Se o limite por IP é de 10 req/s e a de um determinado token é de 100 req/s, o rate limiter deve utilizar as informações do token.

## Requisitos Desafio:

- [X] O rate limiter deve poder trabalhar como um middleware que é injetado ao servidor web
- [X] O rate limiter deve permitir a configuração do número máximo de requisições permitidas por segundo.
- [X] O rate limiter deve ter ter a opção de escolher o tempo de bloqueio do IP ou do Token caso a quantidade de requisições tenha sido excedida.
- [X] As configurações de limite devem ser realizadas via variáveis de ambiente ou em um arquivo “.env” na pasta raiz.
- [X] Deve ser possível configurar o rate limiter tanto para limitação por IP quanto por token de acesso.
- [X] O sistema deve responder adequadamente quando o limite é excedido:
    - Código HTTP: 429
    - Mensagem: you have reached the maximum number of requests or actions allowed within a certain time frame
- [X] Todas as informações de "limiter” devem ser armazenadas e consultadas de um banco de dados Redis. Você pode utilizar docker-compose para subir o Redis.
- [X] Crie uma “strategy” que permita trocar facilmente o Redis por outro mecanismo de persistência.
- [X] A lógica do limiter deve estar separada do middleware.

## Configuração Projeto

### Pré-requisitos (testado no linux)

- Make versão 4.3
- Go versão 1.22.2
- Docker versão 24.0.7
- Docker Compose versão v2.3.3

### Variáveis de Ambiente

Para executar este projeto, você precisará adicionar as seguintes variáveis de ambiente:

As variáveis de ambiente abaixo são tanto para o limit-rate por IP ou TOKEN:

- `MAX_REQUESTS`: Máximo de requisições permitidas dentro da janela de tempo, valor padrão é `10`
- `WINDOW_SECONDS`: Intevalo, em segundos, a ser considerado para o cálculo da quantidade de requisições, valor padrão é `1`
- `BLOCK_SECONDS`: Tempo em segundos do bloqueio caso exceda a quantidade de request na janela, valor padrão é `5`

Caso precise de maior controle as variáveis abaixo têm precedencia:
- `MAX_REQUESTS_IP`: Específica para IP
- `WINDOW_SECONDS_IP`: Específica para IP
- `BLOCK_SECONDS_IP`: Específica para IP
- `MAX_REQUESTS_TOKEN`: Específica para Token
- `WINDOW_SECONDS_TOKEN`: Específica para Token
- `BLOCK_SECONDS_TOKEN`: Específica para Token

### Execução

1 - Clone o repositório:

```bash
git clone https://github.com/username/project.git
cd project
go mod tidy
```
2 - Executar testes:
```bash
make test-integration
```
3 - Executar Api:
```bash
make run
```
