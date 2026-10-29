# AGENTS.md

Guia para agentes de IA que vão continuar o CLIdle. Leia junto com [`docs/design.md`](docs/design.md), que guarda as decisões de design já tomadas com o dono do projeto.

## O projeto

Idle game de terminal em Go: um herói (`@`) explora sozinho uma dungeon estilo Rogue em expedições, volta à cidade com loot ou morre e perde o loot. O jogador configura expedições e gasta ouro entre elas. Há progresso offline de até 12h.

## Comandos

```sh
go build -o clidle .     # binário (ignorado pelo git)
go test ./...            # todos os testes; inclui ~10s do teste de 12h offline
go test -short ./...     # pula o teste de desempenho
go vet ./...
gofmt -l .               # precisa sair vazio
```

## Arquitetura

```
main.go                 carrega o save, calcula o tempo fora, abre a UI
internal/game/          lógica pura, sem I/O de terminal
  game.go               Game, Expedition, Config, loop (Advance/step), IA do herói, combate, retirada
  class.go              Class, atributos de dano, o que cada classe pode usar, escolha da classe
  ability.go            habilidades (Has), mana, golpe forte, segundo fôlego
  spell.go              magias do mago e do clérigo (cast, castTarget, blink, cura, bênção)
  migrate.go            migração de saves antigos (um passo por versão)
  dungeon.go            Level, tiles, geração estilo Rogue (grade 3x3 de salas), visibilidade/névoa
  path.go               BFS do herói (search, multi-objetivo) e mapa de distância dos monstros
  entity.go             Hero, Monster, spawn de monstros e chefes, dano
  items.go              Item, raridade, FloorItem, geração de loot
  town.go               loja, ferreiro, treino, poções, fila de compras, chegada na cidade
  offline.go            CatchUp (simulação do tempo fora) e Summary
  save.go               save/load JSON atômico
  balance.go            TODOS os números de balanceamento e tabelas (monstros, chefes, nomes de itens)
  log.go                log do jogo (com contador ×N)
  rng.go                RNG PCG serializável
internal/lang/          textos do jogo (en.json, pt.json embutidos), lang.T/Set
internal/ui/            Bubble Tea
  model.go              Model, ticks, teclas, autosave, catch-up em goroutine, confirmações
  view.go               layout adaptável, mapa, log, painel do herói, legenda, cidade, resumo
  prep.go               aba Preparação (linhas navegáveis)
  style.go              estilos/cores
```

## Invariantes que não podem quebrar

1. **Determinismo.** Toda aleatoriedade do jogo passa por `*RNG` (`Game.RNG` para coisas da cidade, `Expedition.RNG` dentro da expedição). Nunca use `math/rand` global nem `time.Now()` dentro de `internal/game`.
2. **`Advance` é independente do fatiamento.** `Advance(120000)` precisa dar o mesmo estado que 2400× `Advance(50)`. O tempo é contado em milissegundos inteiros (`AccMs`). Não zere nem arredonde `AccMs` no meio do loop. Há testes para isso (`TestAdvanceChunkingIsDeterministic`, `TestCatchUpMatchesOnlinePlay`).
3. **Offline usa a mesma lógica do jogo ao vivo.** `CatchUp` só chama `Advance`. Não crie atalhos estatísticos.
4. **Save/load preserva a simulação.** `TestSaveLoadRoundTripKeepsSimulationIdentical` garante que carregar e continuar dá o mesmo resultado. Ponteiros não sobrevivem ao JSON: identidade de item é por `Item.ID`, não por ponteiro.
5. **Compatibilidade do save.** `saveVersion = 2`. Campos novos devem funcionar com valor zero (ex.: `Shortcuts` vazio vira `ShortcutList() == [1]`). O que valor zero não resolve ganha um passo em `migrate.go`; nunca edite um passo já publicado. Não renomeie campos exportados de structs salvas sem migração.
6. **Concorrência.** `path.go` usa um buffer global (`scratch`) e o jogo não é thread-safe. Enquanto `Model.catching` for verdadeiro, a UI não pode tocar em `m.g` (a goroutine do `CatchUp` está rodando). `View` retorna cedo nesse estado.
7. **Desempenho.** 12h offline devem simular em poucos segundos (`TestTwelveHoursIsFastEnough`, limite de 15s). O custo está no `search` por passo. Evite BFS extras por passo.
8. **Bubble Tea:** `Init` tem receiver por valor; mudanças no model feitas ali se perdem. Configure o estado no construtor (`NewWithSave`).

## Mecânicas importantes (resumo)

- **Passo do herói** (`step`): mana regenera, monstros rápidos agem, herói checa retirada e faz **uma** ação na ordem: segundo fôlego, cura por magia, poção, magia à distância (mago), bênção, ataque, andar. Depois monstros lentos agem. Tudo de um passo vira **uma linha de log** (`round` + `flush`).
- **Classes:** `Hero.Class` (zero = Guerreiro). Habilidades liberam por nível (`AbilityInfo` em `balance.go`, `Hero.Has`). Estado de habilidade da expedição (`Attacks`, `SecondWindUsed`, `ManaAcc`, `Blessed`) fica na `Expedition`; sono/lentidão ficam no `Monster`. Mana regenera em inteiros (`ManaAcc`) para não quebrar o determinismo.
- **Ritmo:** explorar segue `StepsPerSec(Velocidade)`; combate é fixo em `CombatRoundMs` (também quando o mago tem alvo e mana para atacar de longe). Velocidade em combate = iniciativa e ataques extras (sistema de energia).
- **IA do herói** (`chooseStep`): uma BFS com prioridade: monstro visível > item desejado > escada (modo descer direto) > fronteira inexplorada > chefe ou escada.
- **Config da expedição** é copiada em `Expedition.Config` ao partir; a IA lê dela, não de `Game.Config`.
- **Compras com o herói fora:** loja e ferreiro vão para `Pending`/`Upgrades` e são aplicados em `arriveInTown` (que roda tanto na volta quanto na morte). Treino vale na hora.
- **Morte:** restaura `StartEquip`, perde ouro e mochila da expedição; mantém poções e XP.
- **Chefe:** a cada `BossEvery` andares, na sala da `>`; bloqueia a descida; ao morrer dropa item épico e libera atalho para `andar+1`.

## Testando a UI

Quase não há testes de UI (só `internal/ui/model_test.go`). Para ver telas:

- Teste temporário em `internal/ui` que monta `New(g)`, define `m.w, m.h` e imprime `m.View()`. Apague depois.
- Ou `tmux`:
  ```sh
  tmux new-session -d -s cl -x 100 -y 30 "XDG_DATA_HOME=/tmp/algum-dir ./clidle"
  tmux send-keys -t cl e; sleep 0.3; tmux send-keys -t cl Enter
  tmux capture-pane -p -t cl
  ```
  - **Sempre** use um `XDG_DATA_HOME` isolado para não sobrescrever o save real do usuário.
  - Mande **uma tecla por `send-keys`**: várias juntas chegam como um único `KeyMsg` (ex.: `"ar1"`) e não são reconhecidas.
- Para testar offline: salve, edite `SavedAt` no JSON para horas atrás e abra de novo.
- Para balanceamento: teste temporário que roda `CatchUp` com algumas seeds e imprime nível, andar máximo, mortes e ouro.

## Convenções

- **Texto do jogo só em `internal/lang/en.json` e `pt.json`**, acessado por `lang.T("chave", args...)`. Toda chave nova vai nos dois arquivos (um teste confere). Identificadores e comentários de código em inglês.
- **Comentários só quando o porquê não é óbvio.** Regras de negócio ficam no `docs/design.md`, não em comentários.
- **Mudanças mínimas.** Diffs pequenos e focados.
- **Números de balanceamento só em `balance.go`.**
- Commits no estilo `feat: ...` / `docs: ...` em português, terminando com a linha `Co-Authored-By` do agente.
- Rode `gofmt`, `go vet` e `go test ./...` antes de commitar.

## Como trabalhar com o dono do projeto

- Uma etapa por vez: termine, mostre o resultado, depois siga.
- Respostas curtas, em listas, com **negrito** no que importa.
- Quando precisar de uma decisão, faça a pergunta no **final** da mensagem, precedida de 🗿.
- Ao construir algo grande em marcos, **pare ao fim de cada marco** para ele testar, a não ser que ele peça para ir até o fim.
- Decisões de design são dele: proponha com uma recomendação e espere a resposta. Pequenas decisões de implementação podem ser tomadas, mas avise.
- Confirme antes de ações externas (push, criar issues, etc.).
- Ao dar instruções de teste, diga exatamente o que observar.

## Estado atual

Os 5 marcos do design estão prontos: exploração, combate, loot/retirada, cidade/save, chefes/atalhos/offline. Depois vieram idioma (inglês padrão, português opcional) e classes (4 classes, atributos FOR/DES/INT/CON, capacete/escudo, habilidades, mana e magias).

**Problemas conhecidos / ajustes pendentes:**
- Com a config padrão, a mochila (8 vagas) enche por volta do andar 6–8, o herói volta e recomeça do andar 1, e a progressão trava até o jogador mudar a config ou usar atalhos.
- O herói quase não usa poções: 0 mortes em 12h com a config padrão. O excesso acima de 10 é vendido na cidade.
- Com "descer direto" e sem recuar com mochila cheia, ele chega ao andar 30 em ~1h. O ritmo geral ainda não foi ajustado jogando.
- Os andares iniciais parecem parecidos (mesma grade 3×3, só rato e morcego no andar 1).
- Números das classes não-guerreiro são provisórios. O ladino é rápido demais (12h offline levam ~10s e ele faz ~3× mais expedições).
- DES ainda não afeta acerto nem crítico; o respingo da bola de fogo ignora paredes; segundo fôlego quase nunca dispara (a poção vem antes).

**Ideias já combinadas para depois** (ver `docs/design.md` → "Para depois"):
- Grupo de heróis (vários `@` com cores diferentes)
- Prestígio (com troca de classe)
- Baú na cidade e equipar manualmente
- Itens lendários únicos
- Cavernas e outras variações de andar
- Mais atributos, materiais, outros prédios
- Regras de retirada mais elaboradas
