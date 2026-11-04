# Product

<!-- impeccable:product-schema 1 -->

## Platform

terminal

TUI em Go + Bubble Tea, roda em Linux e Windows (Windows Terminal). Não é web nem app nativo: só texto, cores ANSI (256 recomendado) e teclado.

## Users

- **Primário: jogadores do público** que baixam o binário da release no GitHub e nunca viram o código. O jogo precisa se explicar sozinho.
- **Experiência mista:** parte conhece roguelikes (Rogue/NetHack) e idle games, parte não. A explicação fica disponível sob demanda (ajuda, legenda, detalhes), sem poluir a tela de quem já sabe.
- **Uso em dois modos:**
  - **Acompanhar:** terminal pequeno num canto, olhado de relance.
  - **Decidir:** terminal grande ou tela cheia para configurar expedições, comprar e equipar.

## Product Purpose

Idle game de terminal com fantasia medieval: um herói (`@`) explora sozinho uma dungeon estilo Rogue em expedições, volta à cidade com loot ou morre e perde o loot. O jogador entra a cada 10–30 min para decidir (config da expedição, compras, treino, equipamento). Há progresso offline de até 12h.

Sucesso: o jogador entende o que o herói está fazendo e por quê, e sente progresso satisfatório sem precisar ler código ou documentação externa.

## Positioning

Um idle game que é um roguelike de verdade rodando no terminal: a dungeon ASCII é simulada passo a passo (mesma lógica online e offline), e o jogador vê o `@` jogar em vez de ver só números subindo.

## Operating Context

- Janela de terminal aberta o dia todo, ao lado do trabalho.
- Teclado apenas: telas `1` Principal · `2` Preparação · `3` Herói · `4` Config · `?` legenda/ajuda · `q` sair.
- Ao reabrir, mostra um resumo do tempo offline.
- Auto-update pela própria interface (`u`).

## Capabilities and Constraints

- Layout adaptável de 40×10 até telas grandes; abaixo disso, aviso de janela pequena. A câmera segue o `@` quando o mapa (~80×21) não cabe.
- Todo texto vem de `internal/lang/en.json` e `pt.json` (inglês padrão, português opcional). Todo texto novo precisa existir nos dois.
- Raridade de item é comunicada por cor (comum, mágico, raro, épico, lendário).
- Efeito CRT opcional (desligado por padrão), puramente estético.
- Decisões de design e regras de jogo ficam em `docs/design.md`.
- **Em aberto:** onde e como explicar atributos (FOR/DES/INT/CON), a IA de exploração automática e as regras de retirada para quem não conhece o jogo.

## Brand Commitments

- Nome: **CLIdle**.
- Estética de roguelike clássico em ASCII (Rogue/NetHack): `@` herói, letras para monstros, `<` `>` escadas, névoa de guerra.
- README declara que o projeto foi feito com uso de IA.

## Evidence on Hand

- Release pública v0.1.0 (Linux e Windows) no GitHub: https://github.com/deivao06/CLIdle/releases
- Não há depoimentos, métricas de uso nem testes com jogadores além do dono. Não inventar.

## Product Principles

1. **O jogo se explica sozinho.** Quem baixou a release entende o que vê sem documentação externa.
2. **Explicação sob demanda.** Detalhe aparece quando pedido (ajuda, legenda, item selecionado), não ocupando a tela de quem já sabe.
3. **Legível de relance.** No terminal pequeno, o estado do herói (HP, andar, o que está fazendo) se lê em um segundo.
4. **Decidir é rápido.** Na visita de 10–30 min, configurar e comprar exige poucas teclas e deixa claro o efeito de cada escolha.
5. **O `@` é o protagonista.** O mapa e o log contam a história; painéis servem a ela.

## Accessibility & Inclusion

- Só teclado.
- **Em aberto (sugestão, não confirmada):** informação passada só por cor (raridade) ganhar também um sinal em texto, para terminais com poucas cores e jogadores daltônicos.
