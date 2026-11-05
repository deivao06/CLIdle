# 02a — Combate e escala de monstros

Auditoria só de leitura. **A-exp** = curva publicada (v0.2.1, `git show HEAD:internal/game/balance.go`). **A-quad** = experimento do dono no working tree (não commitado).

Legenda: **Observação** = lido no código · **Hipótese** = interpretação · **Recomendação** = sugestão.

## 1. Fórmulas (Observação)

### 1.1 Herói (`entity.go`)

| Valor | Fórmula | Função |
|---|---|---|
| Atributo | `Start[t] + Growth[t]*(L-1)/2 + Training[t]` (divisão inteira de `Growth*(L-1)`) | `Hero.Attr` |
| Ataque | `5 + bônus da arma + Attr(DamageAttr)`; DamageAttr = FOR (Guerreiro, Mago, Clérigo), DES (Ladino) | `Hero.Atk`, `Class.DamageAttr` |
| Defesa | `1 + (L-1)/2 + armadura + capacete + escudo` | `Hero.Def` |
| Velocidade | `4 + botas + DES` | `Hero.Spd` |
| HP máx. | `(Base + (L-1)*PerLevel + CON*5) * (100+Vigor)/100` (inteiro) | `Hero.MaxHP` |
| Bônus de item | `Base + Upgrade*max(1, Base/10)` | `Item.Bonus` |
| Erro | `max(0.05, 0.20 - 0.005*DES)`; bênção divide por 2 | `MissChance` |
| Crítico | `min(0.25, 0.005*DES) + Crit%/100` (o efeito passa do teto de 25%) | `Hero.CritChance` |
| Esquiva (Ladino nv5+) | `min(0.35, 0.03*DES)` | `Hero.DodgeChance` |
| Mana máx. | `ClassMana + INT*2` (Mago 10, Clérigo 8; outros 0) | `Hero.MaxMP` |
| Regen. de mana | `ManaAcc += 25 + 4*INT` por passo; cada 100 → +1 MP | `Game.regenMana` |

Classes (`balance.go`): HP base/nível — Guerreiro 30/8, Mago 26/5, Ladino 25/6, Clérigo 28/7. Ganho a cada 2 níveis — Guerreiro FOR+3; Mago INT+3 (começa INT 2); Ladino DES+1 (começa DES 1); Clérigo FOR+2, INT+1.

### 1.2 Dano (`entity.go: damage`)

```
d = atk * (0.8 + 0.4*u) - 0.5*def        u = RNG.Float() em [0,1)
dano = max(1, int(d + 0.5))
```

- Variação ±20% no ataque; defesa reduz **metade** do seu valor, subtrativa.
- **Mínimo 1** por golpe acertado.
- Mesma função para herói, monstros, magias e chefes.

### 1.3 Ataque do herói (`game.go: heroAttack`) — ordem dos sorteios

1. `Chance(erro)` (sempre sorteia). Errou → fim.
2. `Attacks++`. Guerreiro com golpe forte e `Attacks % 4 == 0` → `2 × damage(Atk, Def)` (sem crítico/furtivo); nv10 com ≥2 vizinhos → em todos (giratório). Fim.
3. `d = damage(Atk, m.Def)`.
4. Se `CritChance > 0`: `Chance(crit)` → `d *= 2`. (Sem DES nem efeito, **não sorteia**.)
5. Furtivo (Ladino, monstro dormindo ou com HP cheio) → `d *= 3` (acumula com crítico).
6. Morto-vivo e clérigo com mana para expulsar (3 MP) → `d *= 3`.
7. `hurt`: `m.HP -= d`; roubo de vida cura `d*LifeSteal/100` (inteiro); acorda o monstro (`Asleep = 0`).

### 1.4 Ataque do monstro (`game.go: monsterAttack`)

1. `Chance(MissChance(m.Dex))` → erra; senão, se Ladino com esquiva: `Chance(esquiva)` → erra.
2. `d = damage(m.Atk, heroDef + 3 se abençoado)`.
3. Se `CritChance(m.Dex) > 0`: `Chance` → `d *= 2`.

### 1.5 Velocidade, energia e ritmo

- **Herói: 1 ação por passo**, sempre. Velocidade não dá ataques extras ao herói.
- Monstro acordado ganha `m.Spd / heroSpd` de energia por passo (metade se lento) e age 1 vez por unidade (`monstersAct`).
- **Iniciativa:** `m.Spd > heroSpd` age **antes** do herói no passo; os outros, depois.
- A energia acumula também explorando (monstro perseguindo).
- **Duração do passo** (`StepMs`): `CombatRoundMs = 1000` se está em combate e o passo anterior também estava; senão `1000 / (1 + 0.5*(Spd - DES/2))` ms. Em combate, a velocidade só muda a razão de ataques, não o tempo.
- "Em combate" = monstro adjacente, ou mago com mana e alvo a ≤5 casas visível.

### 1.6 Ordem de ações do herói no passo (`game.go: step`)

regen. de mana → bênção decrementa → monstros rápidos agem → checa retirada → **uma** ação: segundo fôlego › cura (se HP < `PotionAtPct`) › poção › magia (sem recuar) › bênção (com vizinho) › ataque › andar → monstros lentos agem.

### 1.7 Monstros (`entity.go: spawnMonster`)

- Elegíveis: `MinDepth ≤ andar`; **A-quad** também exige `andar - MinDepth ≤ 20` (menos o último tipo, dragão). Dos elegíveis, ficam os **4 últimos** (`MonsterWindow`); sorteio uniforme.
- `s = MonsterScale(andar, MinDepth)`, `n = andar - MinDepth`:
  - **A-exp:** `1.15^min(n, até o 20) × 1.12^(andares além do 20)` (com `MonsterDeepFrom = 20`).
  - **A-quad:** `1 + 0.026n + 0.035n²`.
- `HP = round(HP*s)`, `Atk = round(Atk*s*1.3)`, `Def = round(Def*s)`, `XP = round(XP*s)`, `Spd` fixo, `Dex = Dex + n/2`.
- Por andar: `min(max(5, 3 + andar/2), 12)` monstros (`MonstersPerLevel`).
- Ouro por morte: `max(1, XP/2)`.

### 1.8 Chefe (`entity.go: spawnBoss`)

- Gera 20 monstros do andar e usa o de **maior HP**.
- `HP = int(HP*5)`, `Atk = int(Atk*1.4)`, `Def = int(Def*1.3)`, `Spd = max(Spd, 5)`, `Dex + 5`, `XP*8`. A cada 10 andares. Dorme até distância 1.

### 1.9 Magias e habilidades (`spell.go`, `ability.go`)

| Habilidade | Regra |
|---|---|
| Magias de dano | `damage(SpellPower + INT + bônus da arma, m.Def)`; **não erram, sem crítico**. Míssil 4 (2 MP), Bola de fogo 4 (5 MP, alvo + vizinhos a 1 casa, ≥2 alvos, ignora parede), Raio gélido 6 (4 MP, só se `m.Spd > heroSpd`, lento 5 passos) |
| Sono | 3 MP; alvo acordado, HP > 50%, ≥2 acordados no alcance; 6 passos (chefe 2) |
| Piscar | 4 MP; ≥2 vizinhos e HP < 35% |
| Cura | 4 MP; `min(MaxHP*35/100 + INT, falta)`, só com HP < `PotionAtPct` |
| Bênção | 3 MP; 8 passos; Def +3 e erro ÷2; gasta a ação |
| Expulsar mortos-vivos | 3 MP por golpe, ×3 |
| Golpe forte | a cada 4 acertos, ×2 |
| Segundo fôlego | 1×/andar, HP < 30%, cura 40% |
| Furtivo | ×3 |

### 1.10 Poção e retirada (`game.go`)

- Bebe se HP < `PotionAtPct` (40%), **ou** se: falta ≥ metade da cura, `worst ≥ HP` e `avg < cura`.
- `incoming`: por vizinho acordado, `rate = m.Spd/heroSpd`; `worst += max(1, Atk*1.2 - Def*0.5) * ceil(rate)`; `avg += max(1, Atk - Def*0.5) * rate`.
- Cura da poção: `MaxHP * (35 + PotionBoost)/100`.
- Retirada: mochila cheia, ou **sem poções** e HP < 30%. Recuando, não luta (só se cura e ataca quem bloqueia o caminho).

## 2. As duas curvas (Observação + cálculo)

`n` = andares abaixo do `MinDepth` do monstro ("idade" do tipo).

| n | A-exp (`1.15^n`, tipo com MinDepth < 20, até o andar 20) | A-exp além do 20 (`1.12^n`) | A-quad (`1 + 0.026n + 0.035n²`) |
|---|---|---|---|
| 1 | 1.15 | 1.12 | 1.06 |
| 5 | 2.01 | 1.76 | 2.00 |
| 10 | 4.05 | 3.11 | **4.76** |
| 15 | 8.14 | 5.47 | **9.27** |
| 19 | 14.23 | 8.61 | 14.13 |
| 20 | 16.37 | 9.65 | 15.52 |
| 30 | 66.21 | 29.96 | 33.28 |
| 40 | 267.86 | 93.05 | 58.04 |

- **A-quad é mais forte que A-exp entre n ≈ 6 e n ≈ 18.** Só fica mais fraco depois de n ≈ 19. Como o `MonsterMaxAge = 20` corta os tipos com n > 20, **a curva quadrática quase nunca chega na parte em que é mais suave**, exceto no dragão.
- A-exp é **híbrida**: exponencial 15% até o andar 20, exponencial 12% depois. A-quad é **polinomial por tipo**, mas o conjunto de monstros do andar não é (ver §4).

### Tipos disponíveis por andar (`MonsterWindow = 4`)

| Andar | A-exp | A-quad |
|---|---|---|
| 1–3 | rato, morcego, kobold | igual |
| 4–5 | + goblin | igual |
| 6–7 | morcego, kobold, goblin, lobo | igual |
| 8–10 | kobold, goblin, lobo, zumbi | igual |
| 11–13 | goblin, lobo, zumbi, orc | igual |
| 14–17 | lobo, zumbi, orc, esqueleto | igual |
| 18–23 | zumbi, orc, esqueleto, lobisomem | igual |
| 24–31 | orc, esqueleto, lobisomem, dragão | igual |
| 32–34 | igual (para sempre) | esqueleto, lobisomem, dragão |
| 35–38 | — | lobisomem, dragão |
| 39+ | — | **só dragão** |

## 3. Monstros por andar

Formato: **HP / Atk / Def / Spd / Dex / XP**, como gerados (`spawnMonster`). Script de cálculo replica `MonsterScale` e `scaled`.

### A-exp (v0.2.1)

| Andar | Monstros |
|---|---|
| 1 | rato 6/4/0/4/3/3 · morcego 5/4/0/7/8/4 · kobold 10/5/1/4/2/6 |
| 5 | rato 10/7/0/4/5/5 · morcego 9/7/0/7/10/7 · kobold 17/9/2/4/4/10 · goblin 16/9/2/4/4/10 |
| 10 | kobold 35/18/4/4/6/21 · goblin 32/18/5/4/7/21 · lobo 28/16/2/6/8/21 · zumbi 37/14/4/2/1/20 |
| 15 | lobo 56/32/4/6/10/42 · zumbi 74/28/8/2/3/40 · orc 52/25/9/4/4/38 · esqueleto 30/19/7/4/4/32 |
| 20 | zumbi 150/56/16/2/6/80 · orc 106/50/18/4/6/77 · esqueleto 60/39/14/4/7/65 · lobisomem 60/29/9/6/9/53 |
| 25 | orc 186/89/31/4/9/136 · esqueleto 106/69/24/4/9/114 · lobisomem 105/52/16/6/11/93 · dragão 78/32/11/5/6/67 |
| 30 | orc 328/156/55/4/11/240 · esqueleto 187/121/43/4/12/201 · lobisomem 185/91/29/6/14/164 · dragão 138/56/20/5/9/118 |
| 40 | orc 1018/485/170/4/16/747 · esqueleto 580/377/134/4/17/625 · lobisomem 574/282/89/6/19/510 · dragão 429/175/61/5/14/368 |
| 50 | orc 3162/1507/527/4/21/2319 · esqueleto 1802/1171/416/4/22/1940 · lobisomem 1783/876/277/6/24/1585 · dragão 1333/545/190/5/19/1142 |

### A-quad (working tree)

| Andar | Monstros |
|---|---|
| 1 | igual ao A-exp |
| 5 | rato 10/6/0/4/5/5 · morcego 8/6/0/7/10/7 · kobold 17/9/2/4/4/10 · goblin 15/8/2/4/4/10 |
| 10 | kobold 41/21/4/4/6/24 · goblin 34/19/5/4/7/22 · lobo 27/15/2/6/8/20 · zumbi 33/12/4/2/1/18 |
| 15 | lobo 65/37/4/6/10/49 · zumbi 81/30/9/2/3/43 · orc 50/24/8/4/4/37 · esqueleto 28/18/6/4/4/30 |
| 20 | zumbi 178/66/19/2/6/95 · orc 122/58/20/4/6/90 · esqueleto 63/41/14/4/7/68 · lobisomem 54/26/8/6/9/48 |
| 25 | orc 247/118/41/4/9/181 · esqueleto 144/93/33/4/9/155 · lobisomem 130/64/20/6/11/116 · dragão 74/30/11/5/6/64 |
| 30 | orc 424/202/71/4/11/311 · esqueleto 270/175/62/4/12/291 · lobisomem 286/140/44/6/14/254 · dragão 169/69/24/5/9/145 |
| 40 | dragão 726/297/104/5/14/623 |
| 50 | dragão 1774/725/253/5/19/1520 |

### Média do andar (sorteio uniforme entre os tipos)

| Andar | A-exp HP / Atk / Def | A-quad HP / Atk / Def | A-quad ÷ A-exp (HP×Atk) |
|---|---|---|---|
| 10 | 33.0 / 16.5 / 3.8 | 33.8 / 16.8 / 3.8 | 1.04 |
| 15 | 53.0 / 26.0 / 7.0 | 56.0 / 27.2 / 6.8 | 1.11 |
| 20 | 94.0 / 43.5 / 14.2 | 104.2 / 47.8 / 15.2 | 1.22 |
| 25 | 118.8 / 60.5 / 20.5 | 148.8 / 76.2 / 26.2 | 1.58 |
| 30 | 209.5 / 106.0 / 36.8 | 287.2 / 146.5 / 50.2 | **1.89** |
| 40 | 650.2 / 329.8 / 113.5 | 726.0 / 297.0 / 104.0 | 1.01 |
| 50 | 2020.0 / 1024.8 / 352.5 | 1774.0 / 725.0 / 253.0 | 0.62 |

- **Observação:** entre os andares 15 e 38, **A-quad é mais difícil que A-exp**, até ~1.9× no andar 30. Só fica mais fácil a partir do 40.

### Chefes (`spawnBoss`)

O chefe vem do monstro de maior HP entre 20 sorteios: com 4 tipos, a chance de não sair o tipo de maior HP é `(3/4)^20 ≈ 0.3%`. Na prática, **o chefe é sempre o tipo mais velho/forte do andar**.

| Andar | A-exp (base) HP/Atk/Def/Spd/Dex/XP | A-quad (base) HP/Atk/Def/Spd/Dex/XP |
|---|---|---|
| 10 | zumbi 185/19/5/5/6/160 | **kobold** 205/29/5/5/11/192 |
| 20 | zumbi 750/78/20/5/11/640 | zumbi 890/92/24/5/11/760 |
| 30 | orc 1640/218/71/5/16/1920 | orc 2120/282/92/5/16/2488 |
| 40 | orc 5090/679/221/5/21/5976 | dragão 3630/415/135/5/19/4984 |
| 50 | orc 15810/2109/685/5/26/18552 | dragão 8870/1014/328/5/24/12160 |

- **Observação:** no andar 10, A-quad troca o chefe de zumbi (Atk 19) para kobold (Atk 29, +53%), porque o kobold com n = 9 passa o zumbi em HP (41 > 33). Spd do zumbi (2) sobe para 5 no chefe: o chefe-zumbi ataca 2,5× mais que o zumbi comum.

## 4. Classificação do crescimento

- **Por tipo:** A-exp exponencial (15%/andar → 12%/andar após o 20). A-quad quadrático.
- **Por andar (o que o herói enfrenta):** o produto médio **HP × Atk do andar cresce ~1.25–1.37× por andar nas duas curvas** até o andar 38. Motivo: a janela de 4 tipos sempre contém tipos com n de 0 a ~13–20, e o mais velho domina.
  - A-exp: ~1.31×/andar até o 20, ~1.25×/andar depois (constante = exponencial puro).
  - A-quad: ~1.35×/andar até o 31; **depois cai** (1.24× no 41, 1.16× no 50, 1.12× no 60), porque só resta o dragão envelhecendo.
- **Conclusão (Observação):** com a janela de 4 tipos, **A-quad não é "suave" no meio do jogo**. A diferença real entre as curvas aparece só a partir do 39 (dragão sozinho).

### Rupturas (variação de HP×Atk médio contra o andar anterior)

| Andar | A-exp | A-quad | Causa |
|---|---|---|---|
| 4, 6, 8 | 1.52×, 1.45×, 1.57× | 1.51×, 1.56×, 1.55× | tipo novo forte (goblin, lobo, zumbi) entra e um fraco (rato, morcego) sai |
| 11 | 1.12× | 1.08× | kobold (n = 10) sai, orc entra |
| 14 | 0.99× | 0.95× | goblin sai |
| 18 | 0.96× | 0.93× | lobo (n = 12) sai |
| 24 | **0.71×** | **0.72×** | zumbi (n = 16) sai, dragão (n = 0) entra: **o andar 24 é mais fácil que o 23** |
| 32 | — | 0.98× | orc envelhece e sai (`MonsterMaxAge`) |
| 35 | — | 1.12× | esqueleto sai |
| 39 | — | 0.93× | lobisomem sai; só dragão |

- **Andares de chefe (10, 20, 30…)**: o chefe é `HP×5`, `Atk×1.4` do tipo mais velho. Ver §5 para o custo.

## 5. Custo de uma luta para um herói de referência (Hipótese / cenário)

**Cenário (não é dado do jogo):** Guerreiro, **nível = andar**, só itens **comuns do próprio andar** sem ferreiro (arma, armadura pesada, capacete, escudo, botas), DES 0, golpe forte (média ×1,25 nos acertos). Serve para comparar curvas, não para prever o jogo; os números reais vêm do **sim**/**progression**.

- Herói no andar d: `Atk = 5 + round(2+1.2d) + 3(d-1)/2`; `Def = 1 + (d-1)/2 + round((1+0.8d)·1.25) + 2·round((1+0.8d)/2)`; `HP = 30 + 8(d-1)`; `Spd = 4 + int(1 + d/6)`.
- Métrica: **% do HP máximo perdido para matar um monstro** = passos para matar × dano esperado por passo ÷ HP. Dano esperado integrado sobre `u` com arredondamento e mínimo 1; acerto `1 − MissChance`; crítico do monstro `1 + CritChance`; ataques por passo `m.Spd / heroSpd`.

| Andar | Herói Atk/Def/HP | A-exp: média / pior tipo | A-quad: média / pior tipo |
|---|---|---|---|
| 10 | 32/26/102 | 2% / 3% | 3% / 6% |
| 15 | 46/38/142 | 4% / 9% (lobo) | 5% / 14% (lobo) |
| 20 | 59/49/182 | 7% / 13% (orc) | 11% / 20% (orc) |
| 25 | 73/61/222 | 14% / 33% (orc) | 30% / 73% (orc) |
| 30 | 86/72/262 | 41% / 93% (orc) | **100% / 192%** (orc) |
| 40 | 113/95/342 | 605% / 1607% | 367% (dragão) |
| 50 | 140/118/422 | ~125 000% | 6746% |

Chefes no mesmo cenário (% HP para matar): A-exp 10 = 26%, 20 = 254%, 30 = 1072%; A-quad 10 = **82%**, 20 = 396%, 30 = 2365%.

- **Hipótese:** o herói cresce **linearmente** (arma `+1.2/andar`, FOR `+1.5/nível`, armadura `+0.8/andar`), os monstros **superlinearmente**. Em qualquer das duas curvas há uma parede; no cenário ela aparece por volta do **andar 25–30**, mais cedo em A-quad.
- **Hipótese:** o **chefe do 20** é a primeira parede real (zumbi com Spd 5 e HP×5) nas duas curvas.

## 6. Picos e paredes (limiares)

**Observação (fórmula):** dano = `Atk·(0.8…1.2) − 0.5·Def`, mínimo 1.

- **Imunidade do monstro:** o herói tira só 1 quando `0.5·Def ≥ 1.2·Atk do herói`, ou seja, `Def ≥ 2.4·Atk`. Como Def do monstro escala igual ao HP, isso acontece cedo em A-exp. No cenário: tipo tira só ~1 de dano (≤ 1,5 em média) a partir de **orc 45, esqueleto 48, lobisomem 52, dragão 56** (A-exp); **dragão 55** (A-quad).
- **Imunidade do herói:** monstros fracos e velhos dão só 1 de dano no cenário: rato e morcego nos andares 3–5/6, zumbi no 8 (A-exp) e 8–10 (A-quad). Monstro novo com Atk ×1.3 compensa.
- **Monstro mais velho domina:** na janela de 4, a diferença entre o tipo mais velho e o mais novo chega a `s = 10.93` vs `1.97` (orc vs dragão, andar 30, A-exp) e `14.13` vs `2.42` (A-quad). O dragão recém-chegado no 24 tem custo ~1% de HP; o orc, 33–73%.
- **Velocidade fixa dos monstros:** `Spd` não escala. O herói ganha `+1 Spd` a cada 6 andares só de botas comuns, então monstros atacam **menos vezes por ação** com a profundidade (lobo Spd 6 vs herói Spd 10 no andar 30 = 0,6 ataque/ação). Morcego (7) age **antes** e 1,4×/ação nos primeiros andares (herói Spd 5).
- **DES dos monstros:** `+1 a cada 2 andares de idade`. Orc no andar 50 (A-exp): Dex 21 → erro 9,5%, crítico 10,5%. Chefe do 60: Dex 31 → erro 5% (mínimo), crítico 15,5%.
- **Previsão de golpe letal ignora crítico (Observação):** `incoming()` calcula `worst` com `Atk·1.2 − Def·0.5`, sem o ×2 do crítico e sem a bênção. Com crítico até 25%, o herói pode morrer num golpe que a poção preventiva não previu.
- **Bola de fogo:** `around(t)` usa só distância, então acerta através de parede (já anotado no AGENTS.md).
- **Chefe-zumbi:** `BossMinSpd = 5` faz o zumbi (Spd 2) virar o chefe mais rápido relativo à base (×2,5 ataques).
- **Andar 24 mais fácil que o 23** nas duas curvas (§4).

## 7. Respostas diretas

- **"Os monstros escalam quadraticamente?"** No working tree, **por tipo sim**. No andar inteiro, **não**: a janela de 4 tipos mantém o crescimento de HP×Atk em ~1,35×/andar (exponencial na prática) até o 31; só fica sub-exponencial no 39+ (só dragão).
- **A-quad deixa o jogo mais fácil?** Não até o andar ~38: é **mais difícil** do 15 ao 38 (até 1,9× no 30) e o chefe do 10 fica 1,5× mais forte em ataque. Fica mais fácil a partir do 40.

## 8. Recomendações (para o dono decidir)

- **R1 — Medir antes de mexer:** comparar A-exp e A-quad no sim nos andares 15–30 e no chefe do 10, onde A-quad endurece.
- **R2 — Se o objetivo é suavizar o meio do jogo**, a alavanca é a **idade máxima** dentro da janela, não a forma da curva: com `MonsterMaxAge` menor (ex.: 12), o tipo mais velho do andar seria menos escalado. Hipótese, precisa de sim.
- **R3 — Chefe do 10 em A-quad:** se a mudança de zumbi para kobold não for desejada, o chefe poderia vir do tipo mais novo ou de HP×Atk, não só de HP. Hipótese.
- **R4 — Parede de Def:** se o fim do jogo deve ser "lento" e não "impossível", o termo `−0.5·Def` com Def escalando como HP cria imunidade. Opções: Def dos monstros com escala menor que HP, ou dano mínimo percentual. Hipótese.
- **R5 — `incoming()`:** considerar o crítico (×2) no `worst`. Mudança pequena, mas muda comportamento; decisão do dono.
