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

### 1 — Flash do MicroPython

Segure BOOT no RP2040 Zero enquanto pluga o cabo. Ele aparece como um drive USB (`RPI-RP2`). Baixe o [MicroPython para RP2040](https://micropython.org/download/RPI_PICO/) e copie o arquivo `.uf2` para o drive. A placa reinicia automaticamente.

### 2 — Instalar o beacon CLI

```bash
curl -fsSL https://raw.githubusercontent.com/YOUR_USERNAME/ai-status-beacon/main/scripts/install.sh | bash
```

Isso baixa o binário pré-compilado para o seu Mac (Apple Silicon ou Intel) e coloca em `~/.local/bin/beacon`.

### 3 — Flash do firmware

```bash
pip install mpremote
./scripts/flash.sh
```

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

O RP2040 Zero tem o NeoPixel no **GP16**. Se você usar um Raspberry Pi Pico padrão (LED no GP25) ou outra placa, edite `firmware/hardware.py`:

```python
_np = neopixel.NeoPixel(machine.Pin(16), 1)  # ← mude o pino aqui
_buzzer = machine.PWM(machine.Pin(15))        # ← mude o pino aqui
```

Depois reflashe: `./scripts/flash.sh`

### Personalizar cores e timings

Edite `firmware/states.py` e reflashe.

**Mudar cores** (RGB, 0–255):
```python
YELLOW = (200, 140, 0)   # thinking
GREEN  = (0, 200, 0)     # done / waiting
RED    = (200, 0, 0)     # error
```

**Mudar durações**:
```python
DONE_DURATION_MS    = 30_000   # quanto tempo o "done" fica verde (ms)
WAITING_DURATION_MS = 60_000   # quanto tempo o "waiting" pisca antes de ir para idle (ms)
```

**Desativar o buzzer** — em `firmware/hardware.py`, substitua `beep()` por um no-op:
```python
def beep(freq=1000, duration_ms=80):
    pass  # buzzer desativado
```

**Mudar a frequência do buzzer** (pitch) — em `firmware/states.py`, o estado `done` chama `hw.beep(1200, 80)`. Ajuste a frequência (Hz) e duração (ms):
```python
hw.beep(880, 100)   # pitch mais grave, beep mais longo
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
firmware/      Código MicroPython (roda no RP2040)
host/          Código Go do beacon CLI
scripts/       flash.sh, install.sh
claude/        settings.example.json para hooks do Claude Code
```

---

## Contribuindo

Issues e PRs são bem-vindos. O firmware é MicroPython puro; o host é um binário Go sem dependências CGO.

## Licença

MIT
