# 01 — Arquitetura e sistemas

Mapa dos sistemas que afetam combate, dificuldade e progressão. Referências no formato `arquivo:função`.

> **Base analisada:** commit `2d21a25` (v0.2.1) **e** o experimento não commitado do dono na árvore de trabalho (ver abaixo). Nenhum arquivo de produção foi alterado nesta auditoria.

## Duas versões da curva dos monstros

| Nome nesta auditoria | Onde | `MonsterScale(depth, minDepth)` com `n = depth − MinDepth` | Extra |
|---|---|---|---|
| **A-exp** | commit `2d21a25` | `1,15ⁿ` até o andar 20, `1,12ⁿ` depois | — |
| **A-quad** | árvore de trabalho (não commitado) | `1 + 0,026·n + 0,035·n²` | `MonsterMaxAge = 20`: um tipo some do sorteio 20 andares depois do seu `MinDepth` (menos o último, o dragão jovem) |

A suspeita de "crescimento quadrático" vem do **A-quad**. As duas versões foram medidas.

## Fluxo de uma expedição

```
Town ──Advance (TownWaitMs 5s, Auto)──► StartExpedition ──► enterLevel(startFloor)
   ▲                                       │  HP/MP cheios, compra poções
   │                                       ▼
returnToTown ◄── retirada (mochila cheia / HP baixo sem poção / LastFloor)
   │            step() a cada StepMs: explora ou luta
   │            morte → die(): perde mochila e metade do ouro, restaura StartEquip
   ▼
cidade: ferreiro, treino, mochila, poções, baú (só aqui)
```

- **Tempo:** `game.go:Advance` consome `AccMs`; explorar custa `1000 / StepsPerSec(Spd − DEX/2)` ms; lutar custa `CombatRoundMs = 1000` ms por rodada (`StepMs`).
- **Offline:** `offline.go:CatchUp` chama só `Advance` (até `OfflineCap = 12h`). Todo o balanceamento vale igual online e offline.

## Sistemas e onde estão

| Sistema | Código | O que decide |
|---|---|---|
| **Geração de monstros** | `entity.go:spawnMonster`, `balance.go:MonsterKinds`, `MonsterScale`, `MonsterWindow = 4`, `MonstersPerLevel` | Só os **4 tipos mais novos** liberados são sorteados. HP, Atk, Def e XP sobem pelo **mesmo fator `s`**; Atk ainda × `MonsterAtkMult 1,3`; DEX +1 a cada 2 andares de idade |
| **Chefes** | `entity.go:spawnBoss`, `BossEvery = 10` | Maior de 20 sorteios × HP 5, Atk 1,4, Def 1,3, XP 8, DEX +5, Spd ≥ 5 |
| **Dano** | `entity.go:damage` | `atk·U(0,8; 1,2) − 0,5·def`, mínimo **1** (defesa **subtrativa**) |
| **Acerto/crítico** | `balance.go:MissChance/CritChance` | mesma fórmula para herói e monstro, depende de DEX |
| **Ataques e iniciativa** | `game.go:monstersAct`, `heroAttack` | sistema de energia por `Spd`; monstros rápidos agem antes do herói |
| **Habilidades/magias** | `ability.go`, `spell.go`, `AbilityInfo` | liberam por nível; mana em inteiros (`ManaAcc`) |
| **Poção e cura** | `game.go:needsPotion/incoming`, `PotionHealPct 35` | bebe em HP baixo ou se o golpe máximo dos vizinhos pode matar |
| **Retirada** | `game.go:checkRetreat` | mochila cheia, HP baixo sem poção, último andar |
| **Poder do herói** | `entity.go:Atk/Def/MaxHP/Spd/Attr` | Atk = 5 + arma + atributo de dano; Def = 1 + (nível−1)/2 + armadura + capacete + escudo |
| **XP e nível** | `balance.go:XPToNext = 20·1,5^(nível−1)` | exponencial |
| **Itens** | `items.go:newItem/newItemOf`, `balance.go:ItemBase/ItemRoll` | bônus **linear no andar** (arma `2 + 1,2·andar`) × raridade (1 a 2,9) |
| **Ferreiro** | `town.go:Upgrade`, `items.go:Bonus`, `balance.go:UpgradeCost` | +`max(1, Base/10)` por nível; custo × `1,6^nível` |
| **Treino** | `town.go:Train`, `TrainCost` | +1 atributo; custo × `1,5^nível` |
| **Ouro** | `items.go:placeLoot`, `GoldPerPile = 5·1,12^(andar−1)`, `GoldPerKill = XP/2`, `SellPrice` | 1–3 pilhas por andar, venda a 60% |
| **Mochila** | `town.go:BagCap/UpgradeBag`, `BagCost = 100·1,8^nível` | 8 a 24 vagas; mochila cheia força a volta |
| **Atalhos** | `game.go:killBoss/ShortcutList/startFloor` | matar o chefe do andar 10k libera começar em 10k+1 |
| **Morte** | `game.go:die` | perde mochila e 50% do ouro da expedição; mantém XP e poções |

## Interdependências que importam para o balanceamento

1. **Monstro × herói em escalas diferentes.** Os monstros crescem por **idade do tipo** (exponencial no A-exp, quadrático no A-quad) **mais** o salto de base quando um tipo novo libera. O equipamento do herói cresce **linearmente** com o andar onde foi achado. Os níveis crescem por XP exponencial.
2. **Ameaça é produto.** HP e Atk do monstro sobem juntos pelo fator `s`, então "HP × dano" sobe com **s²**. No A-quad, `s` já é quadrático em `n`, então a ameaça cresce perto de **n⁴** dentro da vida de um tipo.
3. **Defesa subtrativa com piso 1.** Quando `0,5·Def` do alvo se aproxima do `Atk`, o dano cai para o piso. Pequenas diferenças de Atk/Def viram grandes diferenças de TTK.
4. **A janela de 4 tipos** faz os tipos novos dominarem o sorteio, e o `MonsterMaxAge` (A-quad) limita `n ≤ 20` para os tipos antigos.
5. **Economia ligada à profundidade.** Ouro por pilha cresce `1,12^andar`; custos de ferreiro/treino/mochila crescem exponencialmente por nível comprado. O que o herói compra depende de **até onde ele chega**.
6. **IA lê `Config` ao vivo.** Retirada, filtro de raridade e poções mudam o resultado das simulações tanto quanto os números.

Detalhes de cada fórmula: [02a — Combate e monstros](02a-combat-monsters.md) e [02b — Progressão e economia](02b-progression-economy.md).
