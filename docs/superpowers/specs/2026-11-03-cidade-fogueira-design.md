# Cidade na tela 1 (fogueira) — design

## Intenção

Hoje a Preparação mistura duas coisas: a **config da próxima expedição** e as **compras da cidade** (loja, ferreiro, treino, baú). As compras ainda funcionam com o herói na dungeon, por meio de uma **fila** aplicada na volta.

O dono quer separar isso:
- A cidade vira um **momento**: o herói volta, **descansa na fogueira** e só então você compra, melhora e treina.
- Para melhorar algo no meio de uma expedição, o jogador precisa **forçar a volta** (`r`). Isso dá peso à decisão de voltar.

Referência de clima: menu da fogueira do Dark Souls e o herói descansando ao lado do fogo (só como inspiração de clima; a arte é ASCII original no estilo do mapa).

## Objetivos

1. Com o herói **na cidade**, a **tela 1** mostra:
   - **à esquerda:** uma salinha no estilo do mapa da dungeon, com o `@` ao lado de uma **fogueira**;
   - **à direita:** uma **lista navegável** com as ações da cidade.
2. **Sem fila:** loja, ferreiro, treino, mochila e baú **só funcionam com o herói na cidade**.
3. A **Preparação** (tela 2) fica **só com a config da expedição**.

## Não-objetivos

- Não muda preços, números de balanceamento nem o que cada ação faz (só onde e quando ela pode ser feita).
- Não muda a tela 1 com o herói **na dungeon**: mapa, log e painel do herói continuam iguais.
- A auto-expedição **não espera** o jogador: com ela ligada, o herói parte depois de `TownWaitMs` como hoje.

## Anti-objetivos

- Quebrar o determinismo, o progresso offline ou saves existentes.
- A animação da fogueira usar o `RNG` do jogo (mudaria a simulação).

## Tela 1 na cidade

Layout em 110 colunas (rascunho aprovado pelo dono):

```
 1 Main  2 Preparation  3 Hero  4 Settings              HP 30/30 · lv 1 · town · 120 gold
                                         │ ▶ Depart now
                                         │
       - - - - - - - -                   │ Blacksmith
      |  .  .  .  .  . |                 │   ) Short Sword +1  → Atk +5      44 gold
      |  .  .  .  .  . |                 │   [ Leather Jerkin  → Def +2      18 gold
      |  .  . @&  .  . |                 │ Training
      |  .  .  .  .  . |                 │   Strength     lv 0  +1 STR     40 gold
       - - - - - - - -                   │ Shop
                                         │   ^ Leather Cap  Def +2         96 gold ▲2
     Resting at the campfire...          │   Bag           8 slots → 10   100 gold
                                         │ Chest 3/20
─────────────────────────────────────────│   ] Leather Boots  Spd +1   x sell
Expedition #4 returns to town (bag full) │
You buy 2 potions (20 gold).             │
1-4 screens · ↑↓ choose · enter buy · x sell · e depart · a auto · q quit
```

### Esquerda: fogueira

- Uma sala pequena desenhada com os mesmos caracteres e cores do mapa (paredes, chão `.`).
- O `@` fica **ao lado da fogueira**.
- **Fogueira:** um glifo (`&`, decisão do builder se outro ficar melhor) que **pisca** alternando amarelo, laranja e vermelho.
- **Iluminação:** o chão perto da fogueira é claro; quanto mais longe, mais escuro, até sumir. A luz também **oscila** junto com o fogo.
- A animação usa o **relógio da UI** (ticks do Bubble Tea), nunca o `RNG` do jogo.
- Abaixo da sala: o texto de descanso e, com a auto-expedição ligada, a contagem "próxima expedição em N s" (já existe hoje em `viewTown`).
- O **log** continua embaixo, como na dungeon.

### Direita: lista da cidade

Ordem das seções:
1. **Partir agora** (no topo, como o "Leave" da referência; pede confirmação como hoje)
2. **Ferreiro** (uma linha por espaço de equipamento)
3. **Treino** (FOR, DES, INT, CON)
4. **Loja** (itens à venda + **upgrade da mochila**)
5. **Baú** (equipados + guardados; enter equipa/desequipa, `x` vende), com as mesmas regras de quando aparece que hoje

As linhas reaproveitam os textos e formatos atuais da Preparação. A navegação usa as mesmas teclas: ↑↓ escolhe, enter compra/equipa, `x` vende.

- O **painel do herói não aparece na cidade**: a lista ocupa o lugar dele. HP e ouro continuam no topo; o resto está na tela 3.
- **Fora da cidade:** a tela 1 é a de hoje; a lista não aparece.

## Preparação (tela 2)

Fica só com a config da expedição (auto-expedição, limite de mortes, modo, poções, limites de HP, auto-equipar, filtro de raridade, itens de outra classe, recuar com mochila cheia, último andar, andar inicial). "Partir agora" sai daqui e vai para a lista da cidade.

## Regras do jogo

- `BuyShop`, `Upgrade`, `Train`, `UpgradeBag`, `EquipStash`, `Unequip`, `SellStash` só agem com o herói na cidade. Fora dela não fazem nada (a UI nem mostra as linhas).
- Remover a fila: `Game.Pending`, `Game.Upgrades`, `Game.BagQueued`, o código que aplica isso em `arriveInTown` e os textos "na fila".
- A loja continua sendo sorteada em `arriveInTown`.

## Save

- `saveVersion` sobe para **4**.
- Novo passo em `migrate.go` (3 → 4): a fila só existe com o herói na dungeon, então o passo **devolve o ouro** de tudo que estiver nela e esvazia a fila. Nada já pago se perde. *(Decisão minha.)*
  - `Pending`: devolve `ShopPrice(item)` de cada item.
  - `Upgrades`: devolve a soma dos custos de cada nível na fila, calculados como a UI calculava (sobre o item alvo: o último da loja na fila para aquele espaço, senão o equipado).
  - `BagQueued`: devolve a soma de `BagCost` de cada nível na fila.
  - Por que devolver em vez de aplicar: aplicar no meio da expedição quebra com a morte, porque `die()` restaura `StartEquip` (os itens comprados sumiriam e o item velho vendido voltaria duplicado).

## Fica a critério do builder

- Tamanho exato da sala, posição do `@`, glifo e paleta da fogueira, velocidade da animação.
- Como distribuir a largura entre a fogueira e a lista (a lista precisa caber nos formatos atuais, ~45 colunas).
- **Terminal estreito** (sem espaço para os dois lados): mostrar só a lista no lugar da sala. *(Decisão minha.)*
- Onde guardar o cursor da lista da cidade (separado do cursor da Preparação).

## Pronto quando

- Na cidade, a tela 1 mostra a fogueira animada e a lista; ↑↓/enter/`x` funcionam em todas as seções.
- Na dungeon, nenhuma ação da cidade é possível e a tela 1 é igual à de hoje.
- A Preparação só tem config.
- Um save v3 com fila abre sem perder nada.
- `gofmt -l .` vazio, `go vet ./...` e `go test ./...` passando; `docs/design.md` e `AGENTS.md` atualizados.
