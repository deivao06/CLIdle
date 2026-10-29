# CLIdle

Jogo **idle de terminal** com pegada de fantasia medieval e dungeon explorer no estilo dos roguelikes clássicos (Rogue, NetHack).

A ideia é deixar um terminal pequeno aberto num canto e ver o `@` explorando a dungeon sozinho, enquanto você entra de vez em quando para preparar a próxima expedição.

```
 1 Principal  2 Preparação  3 Herói              HP 38/43 · nv 4 · andar 3 · 120 ouro
            ------------------
            |................|    ------------------
            |.........$......+####+.....>....r.....|
            |....@...........|    ---+--------------
            -------------+----       #
Você acerta o rato (5) · o rato te acerta (2)
Você mata o rato! +3 XP, +1 ouro
Você equipa Espada Curta (raro, Atq +8).
```

## Como funciona

- **Classes:** guerreiro, mago, ladino ou clérigo, escolhida no começo. Cada uma usa armas e armaduras próprias e libera habilidades e magias por nível.
- **Expedições:** o herói sai da cidade, desce andar por andar e volta com o que achou.
- **Sair ou morrer:** se voltar, guarda o ouro e vende a mochila. Se morrer, perde o loot da expedição, mas mantém nível, poções e o equipamento com que saiu.
- **Tudo automático:** ele explora, luta, pega itens, equipa o que for melhor e bebe poções sozinho.
- **Retirada automática:** você define quando ele volta (HP baixo sem poções, mochila cheia, último andar).
- **Cidade:** loja, ferreiro e treino permanente, pagos com o ouro das expedições.
- **Chefes a cada 10 andares:** guardam a escada, deixam cair um item épico e liberam um atalho (começar no andar 11, 21...).
- **Progresso offline:** ao reabrir, o jogo simula o tempo em que ficou fechado (até 12h) e mostra um resumo.
- **Layout adaptável:** funciona de 40×10 até telas grandes; a câmera segue o `@` quando o mapa não cabe.

## Requisitos

- Go 1.25 ou mais novo
- Terminal com suporte a cores (256 cores recomendado)

## Instalar e rodar

```sh
git clone git@github.com:deivao06/CLIdle.git
cd CLIdle
go build -o clidle .
./clidle
```

## Controles

| Tecla | Ação |
|---|---|
| `1` | Tela principal (dungeon ou cidade) |
| `2` | Preparação (config, loja, ferreiro, treino) |
| `3` | Herói (atributos, equipamento, mochila, recordes) |
| `e` | Partir para a expedição (na cidade) |
| `r` | Mandar o herói voltar (na dungeon) |
| `a` | Ligar/desligar a auto-expedição |
| `?` | Legenda dos símbolos |
| `q` | Salvar e sair |

Na **Preparação**: `↑↓` ou `j/k` escolhem, `←→` ou `h/l` ajustam, `enter` compra ou alterna.

Nas **confirmações**: `s`, `y` ou `enter` confirmam; `n` ou `esc` cancelam.

## Legenda

| Símbolo | Significado |
|---|---|
| `@` | herói |
| `.` `#` `+` | chão, corredor, porta |
| `>` `<` | escada para baixo, saída |
| `$` `!` | ouro, poção |
| `)` `[` `]` `^` `}` | arma, armadura, botas, capacete, escudo |
| `r` `b` `k` `g`... | monstros (letra minúscula) |
| `G` `O` `L` `D` | chefes (letra maiúscula) |

Cor do item = raridade: branco comum, azul raro, roxo épico.

## Save

- Fica em `~/.local/share/clidle/save.json` (ou `$XDG_DATA_HOME/clidle/save.json`).
- Salva a cada 30 segundos e ao sair.
- Para recomeçar do zero, apague o arquivo.
- O jogo abre em inglês; o idioma muda na Preparação (`2`).

## Desenvolvimento

```sh
go test ./...          # testes da lógica do jogo
go test -short ./...   # pula o teste de desempenho da simulação de 12h
go vet ./...
```

- **Estrutura:**
  - `main.go`: carrega o save e abre a interface.
  - `internal/game`: toda a lógica do jogo, sem interface e determinística.
  - `internal/ui`: interface em [Bubble Tea](https://github.com/charmbracelet/bubbletea) e [Lip Gloss](https://github.com/charmbracelet/lipgloss).
- **Balanceamento:** todos os números ficam em `internal/game/balance.go`.
- **Design:** decisões de design em [`docs/design.md`](docs/design.md).
- **Para agentes de IA:** veja [`AGENTS.md`](AGENTS.md).
