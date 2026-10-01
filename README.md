# ⚡ Cursor Cost & Usage Monitor (`curcost`)

Ferramenta CLI e TUI em **Golang** de alta performance para monitorar consumo, cotas de requisições rápidas e custos acumulados da sua conta do **Cursor** em tempo real no terminal.

---

## 🌟 Funcionalidades

- **Auto-detecção Zero-Config**: Detecta automaticamente a sessão e o token do Cursor instalado localmente no seu computador (`state.vscdb`), sem necessidade de copiar tokens manualmente.
- **Métricas em Tempo Real**:
  - Gasto do ciclo atual em USD ($) vs. Hard Limit configurado no dashboard.
  - Cota e consumo de requisições rápidas (*fast requests*, ex: `0/500`).
  - Dias restantes até a renovação da assinatura.
  - Detalhamento de consumo individual e por membro da equipe (quando plano Enterprise/Team).
- **Modos de Visualização**:
  - **One-shot**: Exibição visual com cartões e barras de progresso formatadas via Lip Gloss.
  - **Watch / TUI Interativa (**`curcost watch`**)**: Interface reativa em Bubbletea com atualização contínua, atalhos de teclado e status em tempo real.
  - **Statusbar / Short (**`curcost --short`**)**: Formato ultra compacto de uma linha (ideal para `tmux`, Waybar, i3blocks ou prompt Zsh/Bash).
  - **JSON (**`curcost --json`**)**: Saída bruta para scripts de automação e alertas.

---



## 🚀 Instalação e Compilação

Para compilar o binário localmente:

```bash
cd /home/denilson.oliveira/dev/monitor
go build -o curcost ./cmd/curcost
```

(Opcional) Para mover o binário para seu `$PATH`:

```bash
cp curcost ~/.local/bin/
```

---



## 📖 Como Usar



### 1. Visualização Padrão (One-Shot)

```bash
./curcost
```



### 2. Modo Dashboard Interativo (TUI / Watch)

Inicia o dashboard no terminal com atualização periódica:

```bash
./curcost watch
# ou com intervalo personalizado (ex: a cada 15 segundos)
./curcost watch --refresh 15
```

**Atalhos no modo interativo:**

- `r`: Força a atualização imediata dos dados
- `t`: Alterna a exibição da tabela de membros da equipe
- `q` ou `Ctrl+C`: Sai do monitor



### 3. Integração com Tmux / Waybar / Prompts

Exibe formato de linha única:

```bash
./curcost --short
# Saída de exemplo: ⚡ 0/500 reqs | 💰 $203.44
```



### 4. Saída em JSON

```bash
./curcost --json
```



### 5. Status da Autenticação

Verifique de onde as credenciais ativas estão sendo carregadas:

```bash
./curcost auth status
```

---



## ⚙️ Configuração Personalizada (Opcional)

Se estiver rodando em uma máquina onde o Cursor não está instalado, você pode fornecer as credenciais manualmente:

1. **Via Variável de Ambiente**:
  ```bash
   export CURSOR_ACCESS_TOKEN="seu_token_aqui"
   export CURSOR_AUTH_ID="google-oauth2|user_xxx"
  ```
2. **Via Comando de Configuração**:
  ```bash
   ./curcost auth set-token "seu_token_aqui" --auth-id "google-oauth2|user_xxx"
  ```
   *As credenciais são salvas com permissões restritas (*`0600`*) em* `~/.config/cursor-monitor/config.yaml`*.*
3. **Via Flags**:
  ```bash
   ./curcost --token "seu_token" --auth-id "seu_auth_id"
  ```

---

