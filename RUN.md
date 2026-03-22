# Como Executar o Projeto 🚀

Neste documento, você encontra as instruções necessárias para inicializar a infraestrutura, testar os endpoints e executar testes automatizados ou de estresse em sua máquina.

### Requisitos

Para que o projeto funcione conforme o esperado em ambiente local, garanta que os seguintes softwares estão instalados:
- [Docker](https://docs.docker.com/engine/install/) e [Docker Compose](https://docs.docker.com/compose/install/) (para a infraestrutura e serviços).
- [Golang](https://go.dev/doc/install) v1.24+ (apenas se for rodar testes unitários localmente).
- [K6](https://k6.io/docs/get-started/installation/) (apenas se for rodar os Testes de Carga).

---

## 1. Subindo a Infraestrutura com Docker Compose

Toda a arquitetura (API, Consumer, RabbitMQ e PostgreSQL) é gerenciada e orquestrada via Docker Compose, com limites de CPU e Memória já configurados.

1. Na raiz do repositório, faça o *build* e inicie todos os contêineres em *background*:
   ```bash
   docker-compose up -d --build
   ```

2. Aguarde que os contêineres confirmem o estado `Running` e `Healthy`. Você pode validar o estado de todos utilizando o comando:
   ```bash
   docker ps
   ```

3. (Opcional) Você pode acompanhar os logs do produtor ou do consumidor via terminal utilizando:
   - Produtor (API): `docker-compose logs -f app`
   - Consumidor (Worker): `docker-compose logs -f consumer`

---

## 2. Acessando as Interfaces e Consumindo a API

Com os recursos no ar, as seguintes portas estarão atentas no seu host:

- **API Golang:** `http://localhost:8080/ingest`
- **RabbitMQ Dashboard:** `http://localhost:15672` (Credenciais: `guest` / `guest`)
- **PostgreSQL Database:** `localhost:5432` (Usuário: `postgres` / Senha: `postgres` / DB: `telemetry`)

É possível disparar uma requisição manual avulsa usando o `curl` para conferir se o pipeline de ponta a ponta (API -> Fila -> Worker -> DB) está funcionando:

```bash
curl -X POST http://localhost:8080/ingest \
-H "Content-Type: application/json" \
-d '{
   "device_id": "sensor-123",
   "timestamp": "2023-10-31T14:20:00Z",
   "sensor_type": "temperature",
   "reading_nature": "analog",
   "value": 25.5
}'
```

---

## 3. Testes Unitários

O projeto possui cobertura de testes unitários separada para regras da API e tratamento do Consumer através de simulações (Mocks). 

1. Certifique-se de baixar as dependências locais utilizando o comando:
   ```bash
   go mod download
   ```
2. Processe toda a bateria de testes de forma recursiva:
   ```bash
   go test -v ./...
   ```
   *O output confirmará os cenários testados na API (erros de parser, falhas de publisher) e no Consumidor (sucessos, mensagens inválidas e falhas de DB).*

---

## 4. Testes de Estresse com K6

Para validar as premissas de **throughput** e alta **disponibilidade**, o repositório traz um script de carga K6 em `k6/load_test.js` que engatilha centenas de requisições simultâneas em rampa. 

Execute no terminal apontado para a raiz do repositório:
```bash
k6 run k6/load_test.js
```

Você pode observar enquanto o script roda:
1. No dashboard do RabbitMQ, a medição "Messages in queue" subindo (capacidade de retenção do Message Broker).
2. Os logs do Consumidor persistindo registros ao mesmo tempo.
3. No terminal do k6, o retorno confirmando o P95 na ordem de milissegundos entregue pelo protocolo HTTP.
