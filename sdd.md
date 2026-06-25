# SDD — Indicador Físico de Status de IA ("Status Beacon")

**Versão:** 0.1
**Autor:** _(você)_
**Data:** 2026-06-24
**Hardware-alvo:** Waveshare RP2040 Zero
**Host:** macOS (Apple Silicon ou Intel)
**Integração principal:** Claude Code (via hooks)

---

## 1. Visão geral

Um pequeno dispositivo USB que dá **feedback físico** (cor + som) sobre o estado
de um agente de IA rodando no Mac. O objetivo é não precisar ficar olhando o
terminal: o LED na mesa diz o que está acontecendo.

Estados visuais:

| Estado     | Significado                                  | Cor / Som                                   |
|------------|----------------------------------------------|---------------------------------------------|
| `thinking` | IA processando / gerando resposta            | Amarelo fixo (ou pulsando)                  |
| `waiting`  | IA precisa de input do usuário               | Pisca cor de destaque + beep do buzzer      |
| `done`     | Tarefa concluída, sem mais ação necessária   | Verde piscando ~30s, depois apaga           |
| `idle`     | Ocioso / sessão encerrada                     | Apagado                                     |
| `error`    | Falha / tool error                           | Vermelho (opcional)                         |

O dispositivo é **burro de propósito**: ele só recebe nomes de estado por
serial e executa a animação correspondente. Toda a lógica de "quando" mostrar
cada estado vive no host.

---

## 2. Requisitos

### 2.1 Funcionais
- **RF1** — O dispositivo recebe comandos de estado via USB serial (CDC).
- **RF2** — Cada estado tem uma animação de cor e, opcionalmente, um padrão de som.
- **RF3** — `waiting` deve chamar atenção ativamente (piscar + som) por uma
  duração configurável (~60s) ou até receber novo comando.
- **RF4** — `done` pisca em verde por ~30s e então vai para `idle`.
- **RF5** — Um novo comando interrompe imediatamente a animação anterior.
- **RF6** — O host expõe uma forma simples de disparar estados (CLI + função).
- **RF7** — Integração automática com Claude Code via hooks (sem chamada manual).

### 2.2 Não funcionais
- **RNF1** — Sem driver no Mac: usar a porta CDC nativa do RP2040.
- **RNF2** — Alimentação 100% via USB. Sem bateria.
- **RNF3** — Latência host→LED < 200 ms no caminho feliz.
- **RNF4** — Reconexão automática se o dispositivo for desplugado/replugado.
- **RNF5** — Falha do dispositivo nunca pode travar ou atrasar o Claude Code.
  Hooks devem ter timeout curto e falhar silenciosamente.

---

## 3. Arquitetura

```
┌─────────────────────────────────────────────┐
│ macOS                                         │
│                                               │
│  Claude Code ──(hooks)──> beacon CLI          │
│                              │                │
│                              ▼                │
│                        beacon daemon          │   (segura a porta serial aberta)
│                              │                │
│                       Unix socket / serial    │
└──────────────────────────────┼───────────────┘
                                │  USB (CDC serial)
                                ▼
                    ┌───────────────────────┐
                    │ RP2040 Zero            │
                    │  ├── NeoPixel (GP16)   │  LED RGB integrado
                    │  └── Buzzer (GP15)     │  buzzer passivo (PWM)
                    └───────────────────────┘
```

### 3.1 Decisão: daemon vs. one-shot

Há dois modos de o host falar com o dispositivo:

- **One-shot** (simples): cada hook abre a porta serial, envia um comando, fecha.
  Problema: abrir a porta pode togglar DTR e *resetar* o RP2040; além disso,
  abrir/fechar a cada evento adiciona latência e pode colidir se dois hooks
  dispararem juntos.
- **Daemon** (recomendado): um processo leve fica residente, segura a porta
  serial aberta, e escuta um Unix domain socket. A CLI só escreve uma linha no
  socket. Robusto, rápido, sem reset, sem colisão.

**Decisão:** implementar o **daemon** como caminho principal, mas a CLI deve
ter um *fallback* one-shot (`--direct`) caso o daemon não esteja rodando, pra
nunca bloquear o desenvolvimento inicial.

---

## 4. Hardware

| Componente              | Pino RP2040 Zero | Observação                                  |
|-------------------------|------------------|---------------------------------------------|
| NeoPixel WS2812B        | GP16             | Já integrado na placa Waveshare             |
| Buzzer passivo          | GP15 (PWM)       | Outro lado no GND; passivo p/ controlar tom |
| Alimentação             | USB-C            | Fornecida pelo Mac                          |

> Buzzer **passivo** (não ativo) — o passivo precisa de PWM pra gerar o tom,
> o que dá controle de frequência/melodia. O ativo só faz "biip" fixo.

---

## 5. Protocolo serial (o contrato)

Comunicação por linhas de texto, terminadas em `\n`, a **115200 baud**.
Texto puro mantém o debug trivial (dá pra testar com qualquer terminal serial).

### 5.1 Host → Dispositivo
```
<estado>\n
```
Estados válidos: `thinking`, `waiting`, `done`, `idle`, `error`, `ping`.

Forma estendida opcional (futura), parse tolerante:
```
waiting duration=60\n
thinking color=255,180,0\n
```

### 5.2 Dispositivo → Host
- Ao bootar: `READY\n`
- Resposta a `ping`: `PONG\n`
- Comando desconhecido: `ERR unknown <cmd>\n`

O host **não depende** de respostas pro caminho feliz (fire-and-forget), mas o
daemon pode usar `ping`/`PONG` como heartbeat de saúde da conexão.

---

## 6. Firmware (RP2040 / MicroPython)

### 6.1 Modelo de execução
Loop não-bloqueante baseado em máquina de estados. **Nada de `sleep()` longo**
no loop principal — senão um comando novo (ex.: `waiting` → `thinking`) demora a
ser atendido. As animações avançam por *tick* usando `time.ticks_ms()`.

```
loop:
    if há linha no stdin (não-bloqueante):
        novo_estado = parse(linha)
        trocar estado (reseta fase da animação)
    avançar animação do estado atual em 1 tick
    sleep curto (~10ms)
```

### 6.2 Animações por estado
- `thinking` — amarelo com respiração senoidal (pulsa suave).
- `waiting` — pisca on/off rápido na cor de destaque; buzzer toca um padrão
  curto a cada N segundos (não contínuo, pra não enlouquecer ninguém); expira
  após `duration` → vai pra `idle`.
- `done` — verde piscando por 30s + dois beeps curtos no início → `idle`.
- `idle` — LED apagado, buzzer mudo.
- `error` — vermelho fixo.

### 6.3 Arquivos
- `main.py` — loop principal + leitura serial.
- `states.py` — definição de cores, durações e geradores de animação.
- `hardware.py` — wrappers de NeoPixel e Buzzer (facilita teste/troca de pino).

---

## 7. Host (Python)

### 7.1 `beacon/daemon.py`
- Abre a porta serial (auto-descoberta de `/dev/tty.usbmodem*`).
- Escuta um Unix socket em `~/.beacon/beacon.sock`.
- Repassa cada linha recebida no socket para a serial.
- Reabre a serial automaticamente em caso de desconexão (RNF4).
- Heartbeat opcional via `ping`.

### 7.2 `beacon/cli.py`
Interface chamada pelos hooks e pelo usuário:
```
beacon thinking
beacon waiting
beacon done
beacon idle
beacon --direct done     # fallback sem daemon
```
- Tenta escrever no socket; se falhar e `--direct`, abre a serial diretamente.
- **Timeout agressivo (~300ms)** e exit code 0 *mesmo em falha*, pra nunca
  segurar o Claude Code (RNF5).

### 7.3 Descoberta de porta
Procurar por `/dev/tty.usbmodem*`. Permitir override por env
`BEACON_SERIAL_PORT`. Documentar como achar a porta:
`ls /dev/tty.usbmodem*`.

---

## 8. Integração com Claude Code (hooks)

Esse é o coração da automação. O Claude Code dispara **hooks** em eventos do
ciclo de vida; cada hook é só um comando de shell. Mapeamos eventos → estados.

> ⚠️ Verifique os nomes/comportamentos exatos dos hooks na documentação atual do
> Claude Code antes de implementar — a API de hooks evolui. O mapeamento abaixo
> é a intenção de design; ajuste os nomes conforme a versão instalada.

| Evento (intenção)               | Hook provável        | Comando            |
|---------------------------------|----------------------|--------------------|
| Usuário enviou prompt / IA começou a trabalhar | `UserPromptSubmit` | `beacon thinking` |
| IA precisa de atenção/permissão / ociosa esperando você | `Notification` | `beacon waiting` |
| IA terminou de responder        | `Stop`               | `beacon done`      |
| Sessão encerrada                | `SessionEnd`         | `beacon idle`      |
| (opcional) erro de tool         | `PostToolUse` c/ erro| `beacon error`     |

Exemplo de configuração em `.claude/settings.json` (forma ilustrativa — confira
o schema atual):
```json
{
  "hooks": {
    "UserPromptSubmit": [
      { "hooks": [{ "type": "command", "command": "beacon thinking" }] }
    ],
    "Notification": [
      { "hooks": [{ "type": "command", "command": "beacon waiting" }] }
    ],
    "Stop": [
      { "hooks": [{ "type": "command", "command": "beacon done" }] }
    ],
    "SessionEnd": [
      { "hooks": [{ "type": "command", "command": "beacon idle" }] }
    ]
  }
}
```

Regras de robustez para os hooks:
- O comando precisa estar no `PATH` (instalar a CLI com `pipx`/`uv` ou usar
  caminho absoluto).
- Hooks **nunca** devem bloquear: a CLI já garante timeout curto + exit 0.
- O hook `Notification` é o que ativa o "me chama" (piscar + som por 60s).

---

## 9. Estrutura de diretórios

```
status-beacon/
├── README.md
├── SDD.md                      # este documento
├── firmware/                   # roda no RP2040
│   ├── main.py
│   ├── states.py
│   └── hardware.py
├── host/                       # roda no Mac
│   ├── pyproject.toml
│   └── beacon/
│       ├── __init__.py
│       ├── cli.py
│       ├── daemon.py
│       └── serialio.py         # descoberta de porta + I/O
├── claude/
│   └── settings.example.json   # hooks de exemplo p/ .claude/settings.json
└── scripts/
    ├── flash.sh                # copia firmware p/ o RP2040 (via mpremote)
    └── install-daemon.sh       # (opcional) launchd plist p/ subir o daemon
```

---

## 10. Plano de iteração (milestones)

Pensado para fatias verticais, cada uma testável de ponta a ponta com o
Claude Code.

- **M0 — Bring-up do hardware.** Firmware mínimo: ao bootar manda `READY`,
  aceita `thinking`/`idle`, acende/apaga o LED da placa. Validar que o Mac vê a
  porta serial.
- **M1 — Protocolo completo no firmware.** Todos os estados + animações
  não-bloqueantes + buzzer. Testável mandando texto por um terminal serial.
- **M2 — CLI host (modo `--direct`).** `beacon <estado>` abre a serial e envia.
  Já dá pra usar manualmente.
- **M3 — Daemon + socket + reconexão.** CLI passa a falar com o daemon;
  fallback `--direct` mantido.
- **M4 — Hooks do Claude Code.** Fiação dos 4 eventos. Ajuste fino das
  durações e cores na prática.
- **M5 — Polimento.** `launchd` pra subir o daemon no login, README, padrões
  de som/animação configuráveis.

---

## 11. Setup / bootstrap

```bash
# Firmware: instalar MicroPython no RP2040 Zero (segurar BOOT, plugar, copiar UF2),
# depois enviar os arquivos:
pip install mpremote
mpremote connect auto fs cp firmware/main.py :main.py
mpremote connect auto fs cp firmware/states.py :states.py
mpremote connect auto fs cp firmware/hardware.py :hardware.py

# Host:
cd host && pipx install .     # expõe o comando `beacon`
beacon --direct done          # smoke test

# Descobrir a porta serial:
ls /dev/tty.usbmodem*
```

---

## 12. Testes

- **Firmware:** terminal serial manual; enviar cada estado e observar.
- **`serialio`:** mock de porta; testar descoberta e reconexão.
- **CLI:** testar caminho com daemon vivo, daemon morto (+`--direct`), e
  dispositivo ausente (deve sair 0 sem travar).
- **Integração:** rodar uma sessão real do Claude Code e verificar as transições
  thinking → waiting → done.

---

## 13. Riscos e mitigações

| Risco                                              | Mitigação                                            |
|----------------------------------------------------|------------------------------------------------------|
| Abrir a serial reseta o RP2040 (toggle DTR)        | Daemon segura a porta aberta; evitar abrir/fechar    |
| Nomes/contrato de hooks mudaram na sua versão      | Confirmar na doc atual; isolar mapeamento em 1 arquivo |
| Hook lento trava o Claude Code                     | CLI com timeout curto + exit 0 sempre                |
| Porta serial muda de nome entre boots              | Auto-descoberta `tty.usbmodem*` + override por env   |
| Dois eventos disparam juntos                        | Daemon serializa a escrita na porta                  |

---

## 14. Extensões futuras (fora do escopo v0.1)
- Suporte a múltiplos agentes (cores diferentes por sessão).
- Brilho adaptativo conforme luz ambiente.
- Modo "foco" que silencia o buzzer em horários definidos.
- Backend agnóstico: mesmo daemon servindo outras IDEs/ferramentas além do Claude Code.