# Telemetry Data Ingestion System

Esta aplicação é um backend distribuído desenvolvido em Go focado em lidar com cenários de intenso tráfego de dados de telemetria emitidos por Sensores Industriais. Utilizando uma arquitetura orientada a eventos (Event-Driven), um padrão Pub/Sub e persistência transacional, este sistema garante baixo tempo de resposta em chamadas engarrafadas e previne vazamento de dados, promovendo escalabilidade, durabilidade e resiliência.

## 🏗️ Design e Arquitetura

O ecossistema é baseado no paradigma do desacoplamento de serviços, o que permite o balanceamento de consumo de banco de dados e livra o endpoint principal da dependência sincrona transacional.

```text
    [Dispositivos Embarcados / Sensores]
        │              │              │
        │(HTTP POST)   │              │
        ▼              ▼              ▼
   ┌────────────────────────────────────────┐
   │                                        │
   │               API GIN                  │
   │      (Ingestion Endpoint / Produtor)   │
   │             (Port 8080)                │
   └───────────────────┬────────────────────┘
                       │
                       │ Payload Serialize
                       │ (AMQP 0.9.1)
                       ▼
   ┌────────────────────────────────────────┐
   │                                        │
   │             RabbitMQ Broker            │
   │             (Fila: telemetry)          │
   │                                        │
   └───────────────────┬────────────────────┘
                       │
                       │ Mensagens em Trânsito
                       │ (Acknowledge Engine)
                       ▼
   ┌────────────────────────────────────────┐
   │                                        │
   │            Go Consumer Worker          │
   │       (Desempilhamento Assíncrono)     │
   │                                        │
   └───────────────────┬────────────────────┘
                       │
                       │ Pool Conexões
                       │ (lib/pq Driver)
                       ▼
   ┌────────────────────────────────────────┐
   │                                        │
   │              PostgreSQL 15             │
   │           (Base Relacional)            │
   │                                        │
   └────────────────────────────────────────┘
```

### Componentes Técnicos

1. **API Golang (`app`):** Desenvolvido com o framework `Gin`, o microserviço atua como ponto de coleta operante em porta exposta (`:8080`). Ele realiza o "binding" e a varredura do JSON e submete o evento direto à instância do RabbitMQ através do protocolo AMQP, retornando a requisição o mais breve possível com um `HTTP 202 Accepted`. 
2. **Message Broker (`rabbitmq`):** Retém o pico massivo gerado nos cenários de concorrência. Seus contêineres e vCPUs atuam em gargalo. O envio para fila foi configurado como persistente para que cenários de pane interna não acarretem perda dos payloads transacionados pelos sensores.
3. **Worker Independente (`consumer`):** Possui sua própria virtualização separada da API principal, visando flexibilidade de escala se houver fila em demasiado acúmulo temporário. O software realiza o pre-fetch da fila, lê as informações, as consolida e submete a insert em modelo atômico dentro do banco de dados, reportando falhas através de `Nack(requeue=true)` em caso de não alcance da database.
4. **Relacional Database (`db`):** Armazena de fato as requisições, modelado para suportar `SensorTypes` variáveis sob timestamp de gravação.

## 📄 Guia de Instalação e Testes

Todo manual de execução Docker, passos a passos para uso e métodos de testes (Manuais, Testes Unitários de Ambiente e Estresse por K6) foram segregados para um documento a parte e muito mais sucinto.

🔗 **Acesse as instruções em [Comandos de Execução e Testes (RUN.md)](./RUN.md)**

## 📊 Relatório e Análise Experimental sob Estresse (K6)

Para justificar a topologia, o K6 propõe injetar instabilidades com dezenas de virtuais usuários, onde as respostas do comportamento tornam-se críticas para o sistema. O modelo atual gera as seguintes defesas e constatações perante picos:

1. **Throughput e Latência Otimizados:** O uso de goroutines internas para multiplexar o Gin framework atrelado à conversão serializável imediata para protocolo AMQP causa respostas na ordem da fração de Milissegundos em requisições, deixando a métrica *http_req_duration* estonteante. O banco de dados (o agente mais lento) passa a ser secundário em tempos de resposta.
2. **Prevenção de Timeout em Cenários Fechados:** Como a gravação não foi condicionada a dependência síncrona aos locks e transações pesadas do PostgreSQL (isolados por tabela e pools de TCP), o "Circuit Breaker" dos sensores remotos nunca será atingido (virtualização de latência 0), contendo quebra de envio original e repetições duplas indesejadas pelo dispositivo fonte.
3. **Observação de Isolamento de Recursos Reais:** O `docker-compose.yml` provê uma isolação restritiva por CPU (limitados em frações de vCPU como `0.25` até `0.50`) intencionalmente, para provar que a capacidade de absorção do broker RabbitMQ continua viável ao acúmulo de requisições, enquanto que o Consumer fará o "drag-out" lento e sistemático de acordo com a sua capacidade.
 
### 💡 Evolução e Possíveis Melhorias 
- **Bulk Inserts System:** O consumidor no Go está inserindo os registros unicamente e individualmente conforme consume do `delivery` (linha-a-linha de execução no SQL), o que em um volume massivo força IOPS de Database absurdos. A aplicação Consumer poderia ser atualizada agregando um "Batched Buffer" — acumulando X eventos em instâncias num *Slice* e realizando o descarrego das informações para o Postgres utilizando Múltiplos Value Inserts.
- **Auto-Scaling no Consumer:** Em um provisionamento baseado em Kubernetes, a API principal permaneceria estabilizada, mas através de gatilhos acionados pela "Queue Depth" (tamanho da fila do RabbitMQ), mais pods sub-relacionados ao "Consumer" poderiam ser "spawandos", secando rapidamente as restrições da fila e se desligando logo em seguida, baixando a volumetria de hardware final do Datacenter em períodos inativos. 
