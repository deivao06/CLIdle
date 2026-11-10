# AGENTS.md

Guia para agentes de IA que vão continuar o CLIdle. Leia junto com [`docs/design.md`](docs/design.md), que guarda as decisões de design já tomadas com o dono do projeto.

## O projeto

Idle game de terminal em Go: um herói (`@`) explora sozinho uma dungeon estilo Rogue em expedições, volta à cidade com loot ou morre e perde o loot. O jogador configura expedições e gasta ouro entre elas. Há progresso offline de até 12h.

## Comandos

```sh
go build -o clidle .     # binário (ignorado pelo git)
go test ./...            # todos os testes; inclui ~8s do teste de 12h offline
go test -short ./...     # pula o teste de desempenho
go vet ./...
gofmt -l .               # precisa sair vazio
GOOS=windows go vet ./...   # o update tem código só de Windows
./web/build.sh           # versão web em web/dist (servir com: cd web/dist && python3 -m http.server)
git tag vX.Y.Z && git push origin vX.Y.Z   # publica uma release (pergunte ao dono antes)
```

## Arquitetura

```
main.go                 carrega o save, calcula o tempo fora, abre a UI
internal/game/          lógica pura, sem I/O de terminal
  game.go               Game, Expedition, Config, loop (Advance/step), IA do herói, combate, retirada
  class.go              Class, atributos de dano, o que cada classe pode usar, escolha da classe
  ability.go            habilidades (Has), mana, golpe forte, segundo fôlego
  gambit.go             táticas (gambits): Gambit/Cond/Action, lista padrão (`gambitsV5`), escolha do comportamento (`behavior`), ações (`instant`/`act`), comportamentos (`perform`), editar a lista
  spell.go              magias do mago e do clérigo (cast, castTarget, blink, cura, bênção)
  migrate.go            migração de saves antigos (um passo por versão)
  dungeon.go            Level, tiles, geração estilo Rogue (grade 3x3 de salas), visibilidade/névoa
  path.go               BFS do herói (search, multi-objetivo) e mapa de distância dos monstros
  entity.go             Hero, Monster, spawn de monstros e chefes, dano
  items.go              Item, raridade, FloorItem, geração de loot, preço de venda (SellPrice)
  death.go              DeathScene: recorte do mapa guardado na morte (aba Herói)
  town.go               loja, ferreiro, treino, mochila, baú, poções, chegada na cidade (tudo só na cidade)
  offline.go            CatchUp (simulação do tempo fora) e Summary
  save.go               save/load JSON (store_file.go: arquivo atômico; store_js.go: localStorage no navegador)
  balance.go            TODOS os números de balanceamento e tabelas (monstros, chefes, nomes de itens)
  log.go                log do jogo (com contador ×N)
  rng.go                RNG PCG serializável
internal/update/         releases do GitHub: Check, Install (checksum + troca do binário), Restart; Version via -ldflags
internal/lang/          textos do jogo (en.json, pt.json embutidos), lang.T/Set
internal/ui/            Bubble Tea
  model.go              Model, ticks, teclas, autosave, catch-up em goroutine, confirmações
  view.go               layout adaptável, mapa, log, painel do herói, legenda, cidade, resumo
  prep.go               Preparação (só config), lista da cidade (`townRows`), Config, resetar save
  tactics.go            seção Táticas da Preparação e editor de regra (`ruleRows`, `Model.edit`)
  style.go              estilos/cores
  updater.go            aviso de versão nova, tecla u, confirmação e reinício (checa ao chegar na cidade, máx. 1×/hora)
  news.go               página Novidades da ajuda: notas das releases do GitHub (`update.Notes`); abre sozinha uma vez por versão (`Game.SeenNews`, jogo novo não abre)
  nerd.go               aba 5 Nerd (opção "Coisas de nerd" na Config, Game.Nerd, zero = desligada): lê os números de balance.go ao vivo, um texto por linha em lang
  touch.go              mouse/toque (bubblezone: zonas `tab:`, `row:i.j`, `val`, `btn:`, `key:`, `help:`, `class:` viram teclas) e layout estreito (< 60 colunas): status no topo, ações + abas embaixo; arrasto no celular e rodinha do mouse mandam `ScrollMsg` (scroll.go), que rola o conteúdo sem mover o cursor (`Model.shown` guarda onde a última View parou)
  yield*.go             no navegador, a simulação offline pausa 1 ms por minuto simulado para a página redesenhar
  crt.go                efeito CRT: envelhece as cores ANSI da saída da View (tom verde-fósforo, desbota, scanlines, flicker dos cinzas, borda rasgada) (Game.CRT, zero = desligado)
web/                    versão de navegador (WASM + xterm.js): main.go (ponte JS ↔ Bubble Tea), index.html, build.sh;
                        _bubbletea/tty_js.go completa o Bubble Tea v1 (sem suporte a js) só nesse build
                        PWA: manifest.webmanifest, sw.js (cache-first; build.sh carimba a versão pelo hash dos arquivos), icons/, vendor/ (xterm.js fixado, sem CDN, para abrir offline)
                        publicado no GitHub Pages por .github/workflows/pages.yml (a cada tag ou manual)
```

## Invariantes que não podem quebrar

1. **Determinismo.** Toda aleatoriedade do jogo passa por `*RNG` (`Game.RNG` para coisas da cidade, `Expedition.RNG` dentro da expedição). Nunca use `math/rand` global nem `time.Now()` dentro de `internal/game`.
2. **`Advance` é independente do fatiamento.** `Advance(120000)` precisa dar o mesmo estado que 2400× `Advance(50)`. O tempo é contado em milissegundos inteiros (`AccMs`). Não zere nem arredonde `AccMs` no meio do loop. Há testes para isso (`TestAdvanceChunkingIsDeterministic`, `TestCatchUpMatchesOnlinePlay`).
3. **Offline usa a mesma lógica do jogo ao vivo.** `CatchUp` só chama `Advance`. Não crie atalhos estatísticos.
4. **Save/load preserva a simulação.** `TestSaveLoadRoundTripKeepsSimulationIdentical` garante que carregar e continuar dá o mesmo resultado. Ponteiros não sobrevivem ao JSON: identidade de item é por `Item.ID`, não por ponteiro.
5. **Compatibilidade do save.** `saveVersion = 8`. Campos novos devem funcionar com valor zero (ex.: `Shortcuts` vazio vira `ShortcutList() == [1]`). O que valor zero não resolve ganha um passo em `migrate.go`; nunca edite um passo já publicado. Não renomeie campos exportados de structs salvas sem migração.
6. **Concorrência.** `path.go` usa um buffer global (`scratch`) e o jogo não é thread-safe. Enquanto `Model.catching` for verdadeiro, a UI não pode tocar em `m.g` (a goroutine do `CatchUp` está rodando). `View` retorna cedo nesse estado.
7. **Desempenho.** 12h offline devem simular em poucos segundos (`TestTwelveHoursIsFastEnough`, limite de 15s). O custo está no `search` por passo. Evite BFS extras por passo.
8. **Bubble Tea:** `Init` tem receiver por valor; mudanças no model feitas ali se perdem. Configure o estado no construtor (`NewWithSave`).

## Mecânicas importantes (resumo)

- **Passo do herói** (`step`): mana regenera, monstros rápidos agem, `behavior()` escolhe o comportamento que dirige (pode iniciar a retirada), `instant()` tenta as ações **acima** dele na lista de táticas e, se nenhuma agir, `perform()` faz o comportamento. Depois monstros lentos agem. Sem comportamento ativo o herói anda pelo `Config.Mode`; a descida na `>` passa antes por `atStairsRule()` (regras "terminou o andar ≥ N → Recuar").
- **Táticas (gambits):** `Config.Gambits`, lidas ao vivo. A lista padrão (`gambitsV5`) é a ordem fixa antiga (segundo fôlego, cura, poção, retiradas, magias auto, bênção, ataque) sem a poção antes de golpe letal. Táticas não mudam regras básicas: `Action.Usable`/`CondKind.Usable`/`Gambit.valid` tiram Explorar, Descer, Descansar, `CondLethal` e "terminou o andar" fora do Recuar (as constantes ficam por causa do save); o passo v7 apaga essas regras. mudanças no motor não podem mudar o comportamento de uma lista (compare o estado final de várias seeds/classes antes e depois). Condição = `Cond{Kind, Op, N}` (o que medir, `<` `>` `≤` `≥`, número). Estado na `Expedition`: `ActiveRule`, `ActiveMs`, `RetreatRule`, `Disarmed` (índice+1, zero = nenhum). O passo v6 de `migrate.go` monta a lista a partir das opções antigas (`configV5`). Lista padrão e editor mostram só o que serve para a classe (`Action.Fits`, `CondKind.Fits`, `DefaultGambits(class)`); `ChooseClass` refaz a lista. Tudo de um passo vira **uma linha de log** (`round` + `flush`).
- **Classes:** `Hero.Class` (zero = Guerreiro). Habilidades liberam por nível (`AbilityInfo` em `balance.go`, `Hero.Has`). Estado de habilidade da expedição (`Attacks`, `SecondWindUsed`, `ManaAcc`, `Blessed`) fica na `Expedition`; sono/lentidão ficam no `Monster`. Mana regenera em inteiros (`ManaAcc`) para não quebrar o determinismo.
- **Ritmo:** explorar segue `StepsPerSec(Velocidade)`; combate é fixo em `CombatRoundMs` (também quando o mago tem alvo e mana para atacar de longe). Velocidade em combate = iniciativa e ataques extras (sistema de energia).
- **IA do herói** (`chooseStep` → `plan`): uma BFS com prioridade: monstro visível > item desejado > escada (modo descer direto) > fronteira inexplorada > chefe ou escada. `Game.Doing()` usa o mesmo `plan` para a linha de status (`▸ ...`) acima do log; só a UI chama.
- **Config da expedição:** a IA lê `Game.Config` ao vivo (inclusive as táticas); mudanças valem na hora. Só poções e andar inicial são usados na partida. Mexer na ordem da lista chama `TacticsChanged()`.
- **Compras só na cidade:** loja, ferreiro, treino, mochila e baú não fazem nada com o herói fora; não há fila. Na tela 1, com o herói na cidade, `viewTownScreen` mostra a fogueira (`campfire`, animada pelo relógio da UI, nunca pelo `RNG`) e a lista `townRows`. O passo v4 de `migrate.go` devolve o ouro da antiga fila (`queueV3`).
- **Efeitos de item:** `Item.Effects` (raro 1, épico 2, lendário 3: `EffectCount`), somados por `Hero.Effect` até o teto `EffectCap`. Aparecem no `StatLabel`. Crítico só gasta RNG se o herói tiver o efeito, para não mudar a sequência de quem não tem.
- **Baú:** `Game.Stash`. Só enche com `Config.KeepLoot` (vender loot desligado): `returnToTown` guarda a mochila no baú em vez de vender. `Config.ManualEquip` (auto-equipar desligado) é separado: `better()` devolve falso. O passo v5 de `migrate.go` liga `KeepLoot` em quem tinha `ManualEquip`.
- **Mochila:** tamanho é `Game.BagCap()` (`BagSize` + `BagLevel`×2), nunca a constante `BagSize` direto. `Config.MinRarity` filtra o que vai para a mochila em `wants`.
- **Morte:** restaura `StartEquip`, perde ouro e mochila da expedição; mantém poções e XP.
- **Chefe:** a cada `BossEvery` andares, na sala da `>`; bloqueia a descida; ao morrer dropa item raro, épico ou lendário (`BossEpicChance`, `BossLegendaryChance`) e libera atalho para `andar+1`. Dorme até o herói encostar (`BossWakeDist`). O herói não estima a luta com o chefe (o dono quis que ele não soubesse): não há recuo por chefe forte. Recuando (por qualquer motivo), só rodam as regras acima do recuo (por padrão, a cura) e ele anda para `<`; ataca apenas se um monstro bloqueia o único caminho.
- **Poção:** regras de tática. O herói nunca sabe se um golpe vai matá-lo (não há condição de golpe letal).
- **Morte:** `LastDeath` guarda as últimas linhas do log (aba Herói); 25% do ouro da expedição fica (`DeathKeepGoldPct`).

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
- Para balanceamento: teste temporário que roda `Advance` hora a hora com algumas seeds e imprime nível, andar máximo, mortes e ouro. A cada hora ponha `Config.StartFloor` no atalho mais fundo, como um jogador faria; sem isso o herói sempre recomeça do andar 1 e nunca é testado fundo.

## Convenções

- **Texto do jogo só em `internal/lang/en.json` e `pt.json`**, acessado por `lang.T("chave", args...)`. Toda chave nova vai nos dois arquivos (um teste confere). Identificadores e comentários de código em inglês.
- **Comentários só quando o porquê não é óbvio.** Regras de negócio ficam no `docs/design.md`, não em comentários.
- **Mudanças mínimas.** Diffs pequenos e focados.
- **Números de balanceamento só em `balance.go`.** Número novo ou fórmula nova ganha uma linha na aba Nerd (`ui/nerd.go`, chaves `nerd.*` nos dois idiomas). Funções aparecem por valores calculados (base e razão), nunca com o número copiado.
- Commits no estilo `feat: ...` / `fix: ...` / `docs: ...` **em inglês**, terminando com a linha `Co-Authored-By` do agente. `feat:` e `fix:` viram as notas da release, então escreva para o jogador. Notas de release sem ícones/emojis nos títulos.
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

Os 5 marcos do design estão prontos: exploração, combate, loot/retirada, cidade/save, chefes/atalhos/offline. Depois vieram idioma (inglês padrão, português opcional) e classes (4 classes, atributos FOR/DES/INT/CON, capacete/escudo, habilidades, mana e magias). Por último: balanceamento (menos poções, monstros mais fortes), raridades mágico e lendário, efeitos em épicos/lendários, auto-equipar opcional com baú, limite de mortes e a tela Config (bip, idioma, resetar save). Depois: filtro de raridade para pegar itens, motivo da retirada na linha de volta e upgrade da mochila na cidade. Por fim, uma rodada de correções do teste do dono: painel que quebra linha, teclas `a`/`esc`, ataque mágico no painel, avaliação do chefe, poção contra golpe letal, última morte na aba Herói, itens de outra classe, mais monstros nos andares iniciais e ordem comum < mágico < raro. Em seguida: a cidade virou a tela 1 com fogueira animada e lista (compras só na cidade, sem fila; save v4), efeito CRT opcional (desligado por padrão) e releases com auto-update. Depois (v0.1.4 → v0.2.1): opção de venda separada do auto-equipar, baú, cores na cidade, versão no rodapé; DES dá precisão e crítico a herói e monstros (mesma fórmula, `MissChance`/`CritChance` em `balance.go`; monstros têm `Dex`), INT acelera a mana, ouro mais difícil (1–3 pilhas, venda por 60% + 20% do gasto no ferreiro), lendários nunca vendidos sozinhos (venda manual com confirmação), aba Herói reformulada, ajuda com páginas Atributos e Classes (montada de `AbilityInfo`), desenho da morte e remoção do recuo por chefe forte (o herói não estima a luta).

**Releases:** **v0.1.0**, **v0.1.1**, **v0.1.2**, **v0.1.3**, **v0.2.0**, **v0.2.1**, **v0.3.0**, **v0.3.1**, **v0.3.2**, **v0.4.0** e **v0.5.0** publicadas (v0.5.0: táticas só de combate e recuo — sem Explorar, Descer, Descansar e "um golpe pode me matar"; "terminou o andar" só com Recuar; andar pela dungeon segue o Modo; save v7; herói novo com 5 poções, poção na Loja e "Comprar automaticamente" na Preparação; v0.4.0: táticas/gambits — IA do herói programável com regras quando → fazer → até, operadores `<` `>` `≤` `≥`, filtro por classe, save v6; a lista padrão é o herói antigo sem a poção antes de golpe letal; versão web no GitHub Pages (WASM + xterm.js, offline, instalável no iPhone), toque e mouse, layout de celular; v0.3.2: aba Nerd opcional com todos os números de balanceamento e o impacto de cada um; no modo descer direto o herói pega os itens desejados antes da escada; v0.3.1: armadura, capacete e escudo somam metade da defesa ao HP máximo (`ArmorHPPct`); v0.3.0: auditoria de balanceamento em `docs/balance-audit/` com simulador `balance_audit_test.go`, monstros lineares `1 + 0,45n`, chefes mais fortes a cada chefe, mago com mais mana por nível e por INT, correção do congelamento em `RandomFloor`; v0.2.1: aba Herói em duas colunas com bloco Combate, habilidades com descrição, página Classes e efeitos de item na ajuda, desenho da última morte, sem recuo por chefe forte; v0.1.2: aviso de update só com a versão; v0.1.3: ícone, versão e manifesto no .exe; v0.2.0: DES dá precisão e crítico a herói e monstros, INT acelera a mana, economia mais difícil, venda separada do auto-equipar, lendários nunca vendidos sozinhos, save v5) (binários de Linux e Windows, `SHA256SUMS`). O workflow gera notas dos commits `feat:`/`fix:`; na v0.1.1 elas foram trocadas depois por notas escritas para o jogador (`gh release edit --notes-file`), agrupadas por tema. A v0.1.1 trouxe: arma inicial, config ao vivo, raridade monotônica no andar, fuga na retirada, guerreiro/clérigo com espadas e maças, linha de status `▸`, manual no `?`, linhas de descrição, tema de cores, lamparina, dano flutuante, tremor, momentos (morte, loot, nível) e a opção de efeitos. Publicar versão é ação externa: confirme com o dono antes de criar e enviar a tag. Commits `feat:` e `fix:` viram as notas da release, então escreva a mensagem pensando no jogador.

**Problemas conhecidos / ajustes pendentes:**
- Com a config padrão, a mochila (8 vagas) enche por volta do andar 6–8, o herói volta e recomeça do andar 1. O jogador destrava com o filtro de raridade, o upgrade da mochila (até 24) ou atalhos; o dono preferiu deixar o filtro em comum por padrão.
- O andar 1 ainda leva ~110 s: o tempo é de exploração (ritmo de passos), não de luta.
- Com "descer direto" e sem recuar com mochila cheia, ele chega ao andar 30 em ~1h. O ritmo geral ainda não foi ajustado jogando.
- Os andares iniciais parecem parecidos (mesma grade 3×3).
- O auto-update foi testado de ponta a ponta só com um servidor falso local; a primeira atualização real (v0.1.0 → v0.1.1) e o binário de Windows ainda precisam ser confirmados pelo dono.
- O binário passou de 3,7 para 7,2 MB por causa do `net/http`/TLS do auto-update. O de Windows tem ~11 MB: é compilado sem `-s -w` e com recursos (ícone `assets/icon.png`, versão e manifesto de `winres/winres.json`, gerados no workflow pelo `go-winres`) para o Windows Defender desconfiar menos. Os `.syso` gerados são ignorados pelo git. Mesmo assim o SmartScreen mostra "Unknown publisher" (sem assinatura); o README ensina o "Run anyway". Assinar de graça pela SignPath Foundation (open source) fica para depois.
- Modo gráfico (Ebitengine + shader CRT) foi avaliado e descartado pelo dono: ~14 MB, cgo obrigatório e ~25% de CPU num teste mínimo.
- Números das classes não-guerreiro são provisórios. O ladino ainda faz ~1,7× mais expedições que as outras classes.
- **HP do herói (R7 da auditoria):** feito na v0.3.1 com `ArmorHPPct = 50` (fator 1 e 2 deixaram o ladino fácil demais). O dono não quer HP por nível crescendo com o andar nem CON subindo com o nível. O ladino ainda vai mais fundo que as outras classes (andar ~80 contra ~50 em 48h simuladas).
- **Próximo na fila (dono):** balancear FOR. Investir em FOR não compensa: +1 FOR = +1 Ataque, enquanto a arma dá muito mais. Propor opções antes de mexer. (O dono descartou mexer na "DEX baixa" das classes.)
- Sem o recuo por chefe forte, mortes contra chefes (andares 10, 20, 30…) aumentaram de propósito.
- O respingo da bola de fogo ignora paredes; segundo fôlego quase nunca dispara (a poção vem antes).

**Ideias já combinadas para depois** (ver `docs/design.md` → "Para depois"):
- Grupo de heróis (vários `@` com cores diferentes)
- Prestígio (com troca de classe)
- Itens lendários únicos
- Cavernas e outras variações de andar
- Mais atributos, materiais, outros prédios
- Regras de retirada mais elaboradas
