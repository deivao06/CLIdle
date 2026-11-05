# 02b — Progressão do jogador e economia

Auditoria de balanceamento, papel **progression**. Base: working tree com o experimento **não commitado** do dono (`MonsterScale` quadrático `1 + 0,026n + 0,035n²`, `MonsterMaxAge = 20`). Nada de código foi alterado.

**Método:** cálculo analítico com as fórmulas do código (script Python fora do repo). Valores de item usam o roll médio (0,5). Números de "uma passada" supõem que o herói mata todos os monstros do andar uma vez (modo explorar tudo). **Não é simulação**: confirmar com o doc do **sim**.

---

## 1. Fórmulas extraídas

### Nível e XP

| O quê | Fórmula | Onde |
|---|---|---|
| XP para o próximo nível | `20 · 1,5^(L−1)` | `balance.go: XPToNext` |
| XP do monstro | `scaled(k.XP, MonsterScale)` | `entity.go: spawnMonster` |
| XP do chefe | `base.XP · 8` | `entity.go: spawnBoss` (`BossXPMult`) |
| Subir de nível | HP sobe junto (`h.HP += MaxHP − old`) | `entity.go: gainXP` |

### O que cada nível dá

| Classe | HP/nível | Atributo/nível | Defesa/nível |
|---|---|---|---|
| Guerreiro | +8 | FOR +1,5 | +0,5 (todas) |
| Mago | +5 | INT +1,5 | +0,5 |
| Ladino | +6 | DES +0,5 | +0,5 |
| Clérigo | +7 | FOR +1, INT +0,5 | +0,5 |

- `Attr(t) = Start + Growth·(L−1)/2 + Training` (`entity.go: Attr`; `ClassStats` em `balance.go`).
- `Def = 1 + (L−1)/2 + armadura + capacete + escudo` (`entity.go: Def`).

### Atributos: efeito real por ponto

| Atributo | Efeito | Onde |
|---|---|---|
| **FOR** | +1 Ataque corpo a corpo (Guerreiro, **Mago**, Clérigo). **Nada mais.** | `class.go: DamageAttr`, `entity.go: Atk` |
| **DES** | +1 Vel; −0,5% erro (mín. 5%); +0,5% crítico (máx. 25%); Ladino: +1 Ataque e +3% esquiva (máx. 35%) | `entity.go: Spd/MissChance/CritChance/DodgeChance` |
| **INT** | Mago/Clérigo: +2 MP, +0,04 MP/passo; +1 dano de magia; Clérigo: +1 de cura | `entity.go: MaxMP`, `ability.go: regenMana`, `spell.go: spellAtk/healSpell` |
| **CON** | +5 HP (× Vigor) | `entity.go: MaxHP` (`TrainHP`) |

- **Observação:** o Mago tem `DamageAttr = Str` (golpe de cajado). FOR para mago só ajuda sem mana.
- **Observação:** dano = `atk·U(0,8–1,2) − def·0,5` (`entity.go: damage`). **1 Defesa vale ~0,5 Ataque.**

### Treino (cidade)

`TrainCost = trainBase · 1,5^n` (`balance.go: TrainCost`), base FOR 40, DES 120, INT 40, CON 30.

| Pontos treinados | FOR total | DES total | CON total |
|---|---|---|---|
| 5 | 527 | 1.582 | 394 |
| 10 | 4.530 | 13.597 | 3.395 |
| 15 | 34.945 | 104.849 | 26.205 |
| 20 | 265.931 | 797.814 | 199.444 |

### Itens

| O quê | Fórmula | Onde |
|---|---|---|
| Arma | `(2 + 1,2·f) · rarityMult` | `balance.go: ItemBase` |
| Armadura | `(1 + 0,8·f) · rarityMult · {0,8; 1; 1,25}` (leve/média/pesada) | idem |
| Capacete/escudo | `(1 + 0,8·f) · rarityMult / 2` | idem |
| Botas | `1 + f/6 + raridade` (sem variação) | idem |
| Variação | ±10% (`ItemSpread`) | `ItemRoll` |
| `rarityMult` | comum 1 · mágico 1,5 · raro 1,85 · épico 2,3 · lendário 2,9 | `balance.go` |
| Chance de raridade | mágico 20% · raro 8% · épico 3% · lendário 1% (comum 68%) | `items.go: newItem` |
| Itens por andar | 0–2 (média 1) + 50% de +1 com Faro (Ladino) | `items.go: placeLoot` |
| Chefe | 1 épico de **slot aleatório** | `game.go: killBoss` |
| Efeitos | épico 1, lendário 2; roubo de vida/crit/vigor/XP 5–15, ouro/poção 10–30 | `EffectRange` |

**Bônus de item por andar (roll médio):**

| Andar | Arma C/M/R/E/L | Armadura média C/M/R/E/L | Capacete/escudo C/R/L |
|---|---|---|---|
| 5 | 8 / 12 / 15 / 18 / 23 | 5 / 8 / 9 / 12 / 14 | 2 / 5 / 7 |
| 10 | 14 / 21 / 26 / 32 / 41 | 9 / 14 / 17 / 21 / 26 | 4 / 8 / 13 |
| 20 | 26 / 39 / 48 / 60 / 75 | 17 / 26 / 31 / 39 / 49 | 8 / 16 / 25 |
| 30 | 38 / 57 / 70 / 87 / 110 | 25 / 38 / 46 / 57 / 72 | 12 / 23 / 36 |
| 40 | 50 / 75 / 92 / 115 / 145 | 33 / 50 / 61 / 76 / 96 | 16 / 31 / 48 |
| 50 | 62 / 93 / 115 / 143 / 180 | 41 / 62 / 76 / 94 / 119 | 20 / 38 / 59 |

- **Observação:** o poder de item cresce **linear** no andar (+1,2 Atq/andar na arma comum).

**Chance de um item aleatório servir para a classe** (slot uniforme 1/5, tipo uniforme):

| Classe | Arma | Armadura | Total útil |
|---|---|---|---|
| Guerreiro | 10% | 20% | **90%** |
| Clérigo | 10% | 13% | 83% |
| Ladino | 5% | 13% | 58% |
| Mago | 5% | 7% | **52%** |

- **Observação:** o guerreiro acha arma útil só em ~1 de cada 10 itens; o mago em 1 de 20. O épico do chefe é arma útil só 10% (guerreiro) / 5% (mago) das vezes.
- **Observação:** a mochila (8) enche com itens úteis, então o guerreiro volta mais cedo (~8–9 andares) que o mago (~15). Consistente com o problema conhecido do `AGENTS.md`.

### Ferreiro, venda, loja

| O quê | Fórmula | Onde |
|---|---|---|
| Bônus com melhoria | `Base + Upgrade · max(1, Base/10)` (~+10% da base por nível) | `items.go: Bonus` |
| Valor | `Bonus · 4 · (raridade+1)` | `items.go: Value` |
| Custo do ferreiro | `(Value + 10) · 1,6^u` | `balance.go: UpgradeCost` |
| Venda | `60% do Valor + 20% do gasto no ferreiro` | `items.go: SellPrice` |
| Loja | 3 itens do `MaxDepth`, preço `Valor · 3`; renova a cada chegada na cidade | `town.go: refreshShop/ShopPrice`, `arriveInTown` |
| Mochila | `100 · 1,8^n`, +2 vagas, 8 níveis (8→24), total **13.648** | `BagCost` |
| Poção | compra 10, vende 5, cura `35% MaxHP` (+PotionBoost), máx. 10 | `balance.go`, `entity.go: PotionHeal` |

### Ouro

| Fonte | Fórmula | Onde |
|---|---|---|
| Pilha | `5 · 1,12^(f−1) · U(0,5–1,5)`, 1–3 pilhas | `GoldPerPile`, `placeLoot` |
| Abate | `max(1, XP/2)` | `GoldPerKill`, `game.go: hurt` |
| Morte | fica 50% do ouro da expedição; perde mochila e itens achados; volta ao `StartEquip` | `game.go: die`, `deathKeep` |
| Atalho | matar chefe do andar `f` libera início em `f+1` | `game.go: killBoss`, `startFloor` |
| Offline | teto 12h, mesma simulação | `OfflineCap`, `offline.go` |

- **Observação:** a melhoria do ferreiro **sobrevive à morte** (o `StartEquip` guarda o ponteiro do mesmo item). Itens achados na expedição, não.

---

## 2. Curva de poder esperado (Guerreiro)

### Nível por andar (uma passada, todos os monstros)

| Andar | XP acumulado | Nível | XP de 1 andar ali | XP p/ próximo nível |
|---|---|---|---|---|
| 5 | 144 | 4 | 40 | 67 |
| 10 | 876 | 8 | 168 | 341 |
| 20 | 6.544 | 13 | 903 | 2.594 |
| 30 | 27.407 | 17 | 3.003 | 13.136 |
| 40 | 84.894 | 19 | 7.476 | 29.557 |
| 50 | 226.846 | 22 | 18.240 | 99.757 |

- **Observação:** com a escala antiga (v0.2.1) a curva de nível é quase igual (16 no andar 30, 22 no 50). O experimento não muda o ritmo de nível.
- **Observação:** passado o andar 30, **um nível a cada ~4–5 andares**. Cada nível dá ao guerreiro +1,5 Atq, +0,5 Def, +8 HP.
- **Hipótese:** jogando de verdade (repetindo andares a partir de atalhos), o nível fica ~2–4 acima. O **sim** deve medir.

### Perfis (enviados ao sim)

Só itens/atributos que existem. Nível = tabela acima.

| Perfil | Definição |
|---|---|
| **Sub-equipado** | itens comuns do andar f−10 nos 5 slots (armadura média), sem ferreiro, sem treino |
| **Adequado** | itens comuns do andar f, ferreiro +2 em arma e armadura, FOR = CON = min(8, f/5) |
| **Sobre-equipado** | itens raros do andar f (armadura pesada), ferreiro +4 arma/armadura, FOR 10, CON 10 |
| **Ofensivo** | arma épica do andar f +2 (efeito crit ou roubo de vida, **hipotético** no valor), armadura leve comum, FOR 10, CON 0 |
| **Defensivo** | arma comum, armadura pesada rara +2, capacete e escudo raros, CON 10, Vigor 15 (efeito épico), FOR 0 |

**Atq / Def / HP** e luta contra o monstro médio do andar (golpes para matar / golpes para morrer, sem erro, poção ou grupo):

| Andar (nv) | Monstro médio HP/Atq/Def | Sub | Adequado | Sobre | Ofensivo | Defensivo |
|---|---|---|---|---|---|---|
| 5 (4) | 12 / 7 / 1 | 12/6/54 | 20/15/59 | 38/28/104 | 39/12/54 | 17/26/119 |
| 10 (8) | 34 / 17 / 4 | 18/8/86 · 2,1 / 6,7 | 33/25/96 · 1,1 / 23 | 59/49/136 | 63/21/86 | 29/45/156 |
| 20 (13) | 104 / 48 / 15 | 37/26/126 · 3,5 / 3,6 | 57/44/146 · 2,1 / 5,7 | 97/90/176 · 1,2 / 64 | 105/39/126 · 1,1 / 4,5 | 49/84/202 · 2,5 / 35 |
| 30 (17) | 287 / 146 / 50 | 55/44/158 · 9,6 / 1,3 | 79/64/188 · 5,3 / **1,6** | 137/133/208 · 2,6 / 2,6 | 142/55/158 · 2,5 / 1,3 | 67/123/239 · 6,9 / 2,8 |
| 40 (19) | 726 / 297 / 104 | 70/61/174 · 40 / 0,7 | 100/83/214 · 15 / **0,8** | 171/176/224 · 6,1 / 1,1 | 179/70/174 · 5,7 / 0,7 | 82/162/257 · 24 / 1,2 |
| 50 (22) | 1774 / 725 / 253 | 86/78/198 · ∞ / 0,3 | 118/102/238 · ∞ / 0,4 | 205/218/248 · 23 / 0,4 | 217/86/198 · 20 / 0,3 | 98/200/285 · ∞ / 0,5 |

Chefes (aprox., monstro mais forte ×5 HP, ×1,4 Atq): andar 10 HP 205 / Atq 29 · 20: 890 / 92 · 30: 2.120 / 282 · 40: 3.630 / 415 · 50: 8.870 / 1.014.

- **Observação:** com a escala quadrática, o herói **adequado morre em <2 golpes a partir do andar ~30** e até o sobre-equipado morre em ~1 golpe no 40. Parede em **~28–32** para quem joga normal.
- **Observação (alinhado com combat, `02a-combat-monsters.md` §3):** médias de monstro iguais às do combat. A tabela é otimista porque ignora o erro do herói, a velocidade do monstro (`mSpd/heroSpd`) e a dispersão (no andar 30 o orc 424/202/71 custa ~2× a média). O combat põe a parede em f25–30 com um herói de referência mais otimista (nível = andar, Def 72); a nossa (Def 64) cai um pouco antes, coerente.
- **Observação:** o HP do herói só vem de nível, CON e Vigor. **Item nenhum dá HP direto.** No andar 50 o HP é ~200–285 contra golpes de 600+.

### De onde vem o poder (Guerreiro adequado)

| Fonte | Atq andar 10 | Atq andar 30 | Atq andar 50 |
|---|---|---|---|
| Base | 5 (15%) | 5 (6%) | 5 (4%) |
| Arma (achada) | 14 (42%) | 38 (48%) | 62 (53%) |
| Ferreiro (+2) | 2 (6%) | 6 (8%) | 12 (10%) |
| FOR de nível | 10 (30%) | 24 (30%) | 31 (26%) |
| FOR treinada | 2 (6%) | 6 (8%) | 8 (7%) |

| Fonte | Def andar 30 | HP andar 30 |
|---|---|---|
| Itens + ferreiro | 55 (86%) | 0 |
| Nível | 8 (12%) | 128 (68%) |
| Base | 1 | 30 (16%) |
| Treino (CON) | — | 30 (16%) |

- **Observação:** equipamento é **~55–70% do Ataque** e **~85% da Defesa**. Nível é ~30% do Ataque (via FOR automática). **Treino é <10%** em qualquer fase.

---

## 3. O problema da FOR, em números

**Custo por +1 Ataque (guerreiro):**

| Fonte | Ouro por +1 Atq |
|---|---|
| Treino FOR, pontos 1–5 | 40 → 203 |
| Treino FOR, pontos 10–15 | 2.306 → 17.515 |
| Ferreiro, arma comum andar 30, níveis 0–3 | 54 → 270 (+3 Atq por nível) |
| Ferreiro, arma épica andar 30, níveis 0–3 | 175 → 914 (+8 por nível) |
| **Loja**: arma comum do andar 30 (456) trocando a do andar 20 | **~38** (+12 Atq, mais 60% de revenda da velha) |
| Drop no chão | 0 (+1,2 Atq por andar descido) |

- **Observação:** ponto a ponto, os **primeiros ~5 pontos de FOR custam o mesmo que o ferreiro**. O problema não é o preço inicial: é que **FOR é +1 fixo** e o custo cresce 1,5× por ponto, enquanto arma nova e ferreiro **crescem com o andar** (o ferreiro dá ~10% da base por nível).
- **Observação:** no andar 30 o adequado causa ~54 por golpe; +1 FOR = **+1,9% de dano**. No andar 50, ~0%. Um único upgrade de arma comum → rara do mesmo andar dá +32 Atq.
- **Observação:** FOR só faz uma coisa. DES, INT e CON têm 2–4 efeitos cada. Para o mago, FOR é quase inútil (INT custa igual e dá dano de magia + mana).
- **Observação:** o próprio nível já dá ao guerreiro 1,5 FOR por nível de graça, mais do que o treino rende depois do 10º ponto.

**Recomendação (opções para o dono escolher; nenhuma aplicada):**
1. **FOR em %**: cada ponto dá +X% do Ataque da arma (ex.: 3–5%), em vez de +1 fixo. Escala com o andar como o ferreiro. Mudança em `entity.go: Atk` + número em `balance.go`.
2. **FOR com segundo efeito**: ex. FOR soma à Defesa/2 ou a HP (como no D&D, carga/robustez). Dá motivo para treinar fora do dano.
3. **Custo de treino mais suave**: `1,5^n` → `1,3^n` ou linear+quadrático. Barato, mas não resolve o "+1 fixo" no fundo.
4. **Ganho de treino por andar**: o treino dá `max(1, MaxDepth/10)` por ponto (mesma lógica do ferreiro `Base/10`).

**Recomendação:** opção 1 (escala com a arma, fala com o problema raiz). A 4 é a de menor diff.

---

## 4. Economia

### Renda por expedição (8 andares a partir do atalho, todos os monstros, sem venda de loot)

| Atalho | Pilhas | Abates | Total | 10 andares + chefe |
|---|---|---|---|---|
| 1 | 123 | 180 | **303** | 594 |
| 11 | 382 | 1.583 | **1.965** | 3.336 |
| 21 | 1.186 | 6.324 | **7.510** | 12.081 |
| 31 | 3.685 | 19.152 | **22.837** | 33.970 |
| 41 | 11.445 | 47.268 | **58.713** | 87.269 |

Loot vendido: ~1 item/andar, 60% do valor → ~20 (andar 5) a ~150 (andar 50) por item. **Pequeno.**

- **Observação:** a partir do andar ~10, **~80% da renda vem de abates** (`GoldPerKill = XP/2`, que cresce com `MonsterScale`). A pilha (`1,12^f`) é secundária.
- **Observação:** por construção, **o ouro acompanha a dificuldade** (cresce junto com o monstro). Recompensa em ouro ✔.

### Custos (sumidouros)

| Sumidouro | Forma | Escala com o andar? |
|---|---|---|
| Treino | `base · 1,5^n` | **não** (só com a contagem) |
| Ferreiro | `(Valor+10) · 1,6^u` por item | sim no ganho (Base/10), exponencial na contagem; volta ao zero ao trocar de item (só 20% devolvido) |
| Mochila | 13.648 no total | não, acaba |
| Poção | 10 fixo | **não** |
| Loja | `Valor · 3` | sim, **linear** no ganho |

- **Observação:** renda cresce ~×3 a cada 10 andares; poder comprável com ouro cresce como **log(ouro)** (treino e ferreiro exponenciais na contagem). **Ouro vira abundante e inútil no fundo.** Ex.: no atalho 41 uma expedição paga ~59k ≈ 1–2 pontos de FOR a mais (15º–16º ponto custam 17k–26k).
- **Observação:** a mochila inteira custa ~1 expedição do atalho 21. Some cedo.
- **Observação:** **poção a preço fixo e cura de 35% do MaxHP**: barata demais no fundo, e fraca, porque o MaxHP quase não cresce (≈70 de cura contra golpes de 100+ no andar 30).
- **Observação:** a **loja** é o único conversor linear de ouro em poder (arma lendária do andar 40 = 8.700 = 15% de uma expedição), mas depende de sorte (3 itens; arma útil em ~27% das visitas para o guerreiro) e **o auto/sim do dono nunca compra na loja**.
- **Observação:** morte custa 50% do ouro da expedição. Como os abates dominam, **morrer no fundo ainda dá lucro**. Farm automático é sustentável em ouro.

### Onde os sumidouros passam a renda

- **Andares 1–10:** renda 300–2.000/expedição. Mochila (100–580), treino inicial (40–200) e ferreiro (40–200) disputam o ouro. **Único trecho com escolha real.**
- **Andares 10–25:** ferreiro e treino nos níveis 5–10 (500–2.300 cada) ≈ 1 expedição. Equilíbrio razoável.
- **Andares 30+:** renda 20k–60k; nada útil para comprar além da loja. **Sumidouros ficam para trás.**

### Farm automático

- **Hipótese:** com a política do `sim50_test.go` (gasta no mais barato entre ferreiro, treino e mochila; sobe o atalho), o herói para no atalho mais fundo onde sobrevive (~21–31) e acumula ouro sem conseguir convertê-lo em poder. O avanço depois vem só de nível (log) e de sorte no drop.
- **Hipótese:** o guerreiro progride melhor em equipamento (90% dos drops úteis); mago/ladino ficam mais atrás em arma/armadura.

---

## 5. Resumo

1. **Poder do herói é linear no andar** (itens) e **logarítmico no tempo** (nível, treino). O monstro, no experimento, é **quadrático**. Parede em ~28–32.
2. **HP é o gargalo**: nenhum item dá HP; poção é % de um HP que mal cresce.
3. **FOR**: os primeiros pontos custam o mesmo que o ferreiro; o problema é o **+1 fixo** que some diante da arma. Recomendo FOR proporcional à arma.
4. **Ouro acompanha a dificuldade, mas o poder comprável não.** No fundo o ouro sobra.
5. **Loja** é a melhor fonte de poder por ouro e não é usada pelo auto.

**A confirmar com o sim:** nível real por andar, ouro real por expedição (mortes, retiradas), onde cada classe estaciona.
