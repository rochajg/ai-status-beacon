# AI Status Beacon

Um indicador físico de status USB para o Claude Code (e outros agentes de IA). Uma pequena placa RP2040 Zero na sua mesa mostra o que a IA está fazendo via cor do LED e som — sem precisar olhar o terminal.

| Estado | Significado | LED | Som |
|--------|-------------|-----|-----|
| `thinking` | IA processando | Amarelo, respiração suave | — |
| `waiting` | IA precisa da sua atenção | Pisca rápido | Beep a cada 8s |
| `done` | Tarefa concluída | Verde piscando, 30s | 2 beeps |
| `idle` | Sessão encerrada | Apagado | — |
| `error` | Erro de ferramenta | Vermelho fixo | — |

## Hardware

| Componente | Onde | Observação |
|------------|------|-----------|
| [Waveshare RP2040 Zero](https://www.waveshare.com/rp2040-zero.htm) | Qualquer loja de eletrônicos | Já tem NeoPixel integrado (GP16) |
| Buzzer passivo | GP15 + GND | Opcional — os LEDs funcionam sem ele |
| Cabo USB-C | — | Alimenta a placa pelo Mac |

> **Buzzer passivo.** Buzzers ativos produzem um tom fixo; os passivos precisam de PWM para controlar o pitch.

## Início Rápido

### 1 — Instalar o beacon CLI

```bash
curl -fsSL https://raw.githubusercontent.com/rochajg/ai-status-beacon/main/scripts/install.sh | bash
```

Isso baixa o binário pré-compilado para o seu Mac (Apple Silicon ou Intel) e coloca em `~/.local/bin/beacon`.

### 2 — Flash do firmware

Baixe o `beacon.uf2` da [última release](https://github.com/rochajg/ai-status-beacon/releases/latest).

Segure BOOT no RP2040 Zero enquanto pluga o cabo. Arraste o `beacon.uf2` para o drive `RPI-RP2` que aparecer. A placa reinicia automaticamente.

Pronto — sem MicroPython, sem ferramentas extras.

### 4 — Configurar os hooks do Claude Code

Adicione ao `~/.claude/settings.json` (mescle com o conteúdo existente):

```json
{
  "hooks": {
    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "beacon thinking", "timeout": 1 }] }],
    "Notification":     [{ "hooks": [{ "type": "command", "command": "beacon waiting",  "timeout": 1 }] }],
    "Stop":             [{ "hooks": [{ "type": "command", "command": "beacon done",     "timeout": 1 }] }],
    "SessionEnd":       [{ "hooks": [{ "type": "command", "command": "beacon idle",     "timeout": 1 }] }]
  }
}
```

### 5 — Subir o daemon

```bash
beacon daemon
```

O daemon mantém a porta serial aberta e roteia os eventos dos hooks para o LED. Deixe rodando em um terminal, ou configure com launchd (veja [Avançado](#avançado)).

### 6 — Testar

```bash
beacon status       # mostra saúde do daemon e dispositivo
beacon thinking     # LED fica amarelo
beacon done         # LED fica verde + 2 beeps
beacon idle         # LED apaga
```

---

## Como Funciona

```
Claude Code ──(hooks)──▶ beacon CLI
                              │
                        beacon daemon  ──(USB serial)──▶ RP2040 Zero
                              │                               │
                         Unix socket                    LED NeoPixel
                                                        Buzzer (opt.)
```

A CLI envia o nome do estado para o daemon via Unix socket (`~/.beacon/beacon.sock`). O daemon mantém a porta serial aberta (evitando resets USB) e encaminha o comando ao RP2040, que executa a animação.

Os hooks sempre saem com `0` — nunca bloqueiam o Claude Code, mesmo sem o daemon rodando.

---

## Referência da CLI

```
beacon <estado>            # envia via daemon (padrão)
beacon <estado> --direct   # ignora o daemon, escreve direto na serial
beacon daemon              # sobe o daemon (bloqueia)
beacon status              # saúde do daemon + dispositivo
```

Estados: `thinking`, `waiting`, `done`, `idle`, `error`

---

## Avançado

### Subir o daemon automaticamente no login (launchd)

```bash
cat > ~/Library/LaunchAgents/com.beacon.daemon.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>             <string>com.beacon.daemon</string>
  <key>ProgramArguments</key>  <array><string>/Users/SEU_USUARIO/.local/bin/beacon</string><string>daemon</string></array>
  <key>RunAtLoad</key>         <true/>
  <key>KeepAlive</key>         <true/>
  <key>StandardOutPath</key>   <string>/tmp/beacon-daemon.log</string>
  <key>StandardErrorPath</key> <string>/tmp/beacon-daemon.log</string>
</dict>
</plist>
EOF
launchctl load ~/Library/LaunchAgents/com.beacon.daemon.plist
```

Substitua `SEU_USUARIO` pelo seu nome de usuário.

### Usando uma placa diferente

O RP2040 Zero tem o NeoPixel no **GP16**. Se você usar outra placa, edite `firmware/src/config.h`:

```c
#define LED_PIN     16   /* mude para o pino do LED da sua placa */
#define BUZZER_PIN  15   /* mude para o pino do buzzer da sua placa */
```

Recompile e reflashe com `./scripts/flash.sh`.

### Personalizar cores e timings

Edite `firmware/src/config.h` e recompile.

**Mudar cores** (RGB, 0–255):
```c
#define DEF_THINKING_R  200
#define DEF_THINKING_G  140
#define DEF_THINKING_B    0   /* amarelo */

#define DEF_DONE_R        0
#define DEF_DONE_G      200
#define DEF_DONE_B        0   /* verde */
```

**Mudar durações**:
```c
#define DEF_DONE_MS     30000   /* quanto tempo o "done" fica antes de ir para idle (ms) */
#define DEF_WAITING_MS  60000   /* quanto tempo o "waiting" pisca antes de ir para idle (ms) */
```

**Desativar o buzzer**:
```c
#define DEF_BUZZER_ENABLED  0
```

**Mudar a frequência do buzzer** (pitch):
```c
#define DEF_BUZZER_FREQ     880   /* Hz — pitch mais grave */
#define DEF_BUZZER_DUR_MS   100   /* ms por beep — mais longo */
```

### Sobrescrever a porta serial

Se a auto-descoberta não encontrar a placa:

```bash
export BEACON_SERIAL_PORT=/dev/cu.usbmodem1234
beacon thinking
```

---

## Estrutura do Projeto

```
firmware/      Firmware em C (pico-sdk, gera beacon.uf2)
  src/         Arquivos fonte (config.h, parser, led, buzzer, states, main)
  test/        Testes unitários nativos para o parser (sem RP2040)
  pico-sdk/    Submodule pico-sdk (pinado na versão 2.1.1)
host/          Código Go do beacon CLI
scripts/       flash.sh, install.sh
claude/        settings.example.json para hooks do Claude Code
```

---

## Contribuindo

Issues e PRs são bem-vindos. O firmware é C (pico-sdk); o host é um binário Go sem dependências CGO.

## Licença

MIT
