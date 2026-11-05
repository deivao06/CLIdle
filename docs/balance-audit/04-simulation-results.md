# 04 — Resultados da simulação

Todos os números deste documento vêm de rodadas reais do simulador. Tabelas completas em [`data/`](data/).

## Metodologia

- **Simulador:** `internal/game/balance_audit_test.go` (`TestBalanceAudit`). Ele é pulado sem `BALANCE_AUDIT=1`, então o `go test ./...` continua rápido.
- **Camada A — jogo completo** (`auditFullRun`):
  - usa `New(seed, classe)` e o `Advance` real, segundo a segundo, por **48h de jogo**;
  - na cidade, a política de gasto é **cópia da `spendClass`** do `sim50_test.go` do dono: compra o mais barato entre ferreiro, treino (FOR+CON, INT+CON, DES) e mochila, sempre guardando o dinheiro das poções;
  - o andar inicial é sempre o **atalho mais fundo**.
- **Camada B — duelo** (`auditFight`):
  - um herói (real ou sintético) contra um monstro ou chefe criado por `spawnMonster`/`spawnBoss`, numa sala vazia, usando o `step()` real;
  - **viés otimista conhecido:** cada luta começa com HP cheio e com o segundo fôlego disponível;
  - **ordem do primeiro golpe contra o chefe:** o combat contestou essa ordem. Por isso, as conclusões sobre chefes usam **mortes reais da camada A**, e não a taxa de vitória dos duelos.
- **Amostra:** seeds **1, 2 e 3** × 4 classes × 48h por modelo. Cada classe roda num processo separado, porque o `path.go` usa um buffer global.
- **Modelos:**
  - cada modelo é um patch em [`models/`](models/), aplicado num `git worktree` fora da árvore principal;
  - **A-exp** = commit `2d21a25`;
  - **A-quad** = o experimento não commitado do dono.
- **Como reproduzir:** `docs/balance-audit/tools/run-model.sh <modelo> 48 1,2,3`, depois `python3 docs/balance-audit/tools/compare.py /tmp/clidle-audit/sim-out A-exp A-quad ...`.
- **Validação:**
  - `go test -short ./...` passa com o simulador presente;
  - o combat revisou o `auditFight` contra as fórmulas e confirmou a montagem, com a ressalva do chefe acima;
  - os números de monstro por andar batem com o `02a` e o `02b`.

## Achado colateral: congelamento real do jogo

- O ladino travou para sempre em **D25 e E15** (seed 3, andar 42).
- **Stack:** `Advance → step → enterLevel → placeLoot → Level.RandomFloor` (`dungeon.go`).
- `RandomFloor` sorteia posições **sem limite de tentativas**. Se a sala sorteada já está cheia de monstros e itens, o laço nunca termina.
- **No jogo de verdade, isso congela a UI e o catch-up offline.**
- É mais provável:
  - em andares fundos (12 monstros);
  - com o **Faro do tesouro** do ladino (item extra);
  - em salas pequenas (o mínimo é 6×4, ou seja, 4×2 casas internas).
- **Consequência para a auditoria:** faltam dados do ladino em B35, D25 e E15. Nesses modelos, o "spread entre classes" ficou vazio.

## Resultado principal: todos os modelos (48h, mediana das seeds)

| Modelo | O que muda | h andar 20 | h andar 30 | h andar 40 | h andar 50 | Andar máx 48h | Mortes/h | % mortes em chefe |
|---|---|---|---|---|---|---|---|---|
| **A-exp** | nada (v0.2.1) | 0,64 | 1,68 | 12,1 | — | 40 | 16,6 | 12% |
| **A-quad** | quadrático + `MonsterMaxAge 20` | 0,76 | **3,16** | **17,6** | — | 44 | 18,2 | 4% |
| B15 | linear `1 + 0,15n` | 0,58 | 0,82 | 1,05 | 1,63 | ≥80 | 4,3 | 48% |
| B25 | linear `1 + 0,25n` | 0,72 | 1,18 | 1,97 | 3,09 | ≥80 | 7,9 | 24% |
| **B35** | linear `1 + 0,35n` | 0,95 | 2,11 | 5,45 | 8,75 | 77 | 6,9 | 16% |
| C | HP pelo herói de referência | 1,03 | 1,25 | 1,43 | 1,73 | ≥80 | 2,9 | 25% |
| D15 | linear 0,15 + idade máx 10 | 0,58 | 0,78 | 1,06 | 1,76 | ≥80 | 4,3 | 27% |
| D25 | linear 0,25 + idade máx 10 | 0,77 | 1,22 | 2,18 | 5,12 | ≥80 | 5,2 | 9% |
| **E13** | `1,07ⁿ × 1,3^(chefes passados)` | 0,62 | 1,00 | 2,58 | 12,4 | 50 | 17,1 | 11% |
| E15 | `1,07ⁿ × 1,5^(chefes passados)` | 0,73 | 1,25 | 5,66 | 46,6 (1/9) | 40 | 8,2 | 48% |

"≥80": chegou ao fim da tabela de medição.

## Leitura

### 1. A-quad é mais difícil que A-exp no meio do jogo
- **Andar 30:** 3,16h contra 1,68h (**quase o dobro**).
- **Andar 40:** 17,6h contra 12,1h.
- Isso confirma o combat: com a janela de 4 tipos, a curva quadrática é mais forte entre n≈6 e n≈18, que é exatamente a idade dos tipos que dominam os andares 15–38.
- **A suspeita "o problema é crescimento quadrático" não vale para o commit.** O commit é exponencial. O experimento quadrático **piora** o meio do jogo.

### 2. A parede atual (A-exp) é o andar 36–45, e não um chefe só
| Andares (A-exp) | Min jogados | Mortes/h | XP/min | Ouro/min |
|---|---|---|---|---|
| 21–25 | 1.080 | 0,5 | 1.427 | 942 |
| 26–30 | 879 | 11,9 | 1.930 | 1.295 |
| 31–35 | **17.541** | 3,3 | 4.493 | 3.190 |
| 36–40 | 9.211 | **19,4** | 5.844 | 4.071 |
| 41–45 | 4.305 | **71,6** | 8.977 | 6.287 |

- **80% do tempo total de jogo** (26.752 de 33.635 min) fica nos andares 31–40.
- As mortes por hora sobem **6×** do 31–35 para o 36–40, e mais **4×** no 41–45.
- O XP/min só sobe **1,3×** e **1,5×** nesses mesmos passos. **A recompensa não acompanha o risco.**
- **Chefe do 40:** mago e clérigo ficam parados nele por **32 a 41h** (até o fim das 48h). No 41–45, quem passa morre em quase toda expedição.

### 3. O herói para de ficar mais forte (o motivo da parede)
Warrior, seed 1, A-exp (`data/` → hourly):

| Hora | Nível | Andar máx | HP | Atk | Def | Ouro | Próxima compra |
|---|---|---|---|---|---|---|---|
| 6 | 25 | 36 | 307 | 161 | 201 | 17k | 30k |
| 12 | 27 | 38 | 338 | 178 | 225 | 63k | 89k |
| 24 | 30 | 39 | 377 | 197 | 240 | 52k | 227k |
| 48 | 32 | 40 | 453 | 338 | 373 | 239k | 449k |

- **Das 6h às 24h (18h de jogo):** HP +23%, Atk +22%, Def +19%. Nesse tempo, cada andar fica de **1,25× a 1,37×** mais difícil.
- **O ouro não fica parado:** está sempre abaixo da próxima compra. O problema é que a **próxima compra fica cada vez mais cara**: de 30k para 449k. Assim, as compras ficam raras.
- **Isso refuta parte do `02b`:** o ouro não "sobra sem uso". Ele é guardado para compras exponenciais que quase não acontecem.
- **Padrão do idle** (Pecorella, ver `03`): o poder cresce de forma logarítmica no tempo, enquanto o custo do andar cresce de forma exponencial.

### 4. Morrer quase não pesa
- **Ladino em A-exp:** 1.971 mortes em 2.020 expedições (**97%**), e mesmo assim é a classe que vai mais fundo (andar 45).
- Nos modelos de "parede" (A-exp, A-quad, E13), a maior parte das mortes **não** acontece em chefe (de 4% a 12% em chefe). É desgaste no corredor.

### 5. Modelos lineares fracos zeram o desafio
- **B15, B25, D15 e C** chegam ao andar 80 em **2 a 5h**. Morre-se pouco, e quase metade das mortes é em chefe.
- **C** (HP pelo herói de referência) ficou **fácil demais** porque o herói de referência foi ajustado com folga. O conceito não foi descartado; foi a **calibração** que errou.
- Então **a curva não pode ser só "mais suave"**. O crescimento por tipo precisa ficar entre 0,25 e 0,35 por andar (linear), ou a dificuldade precisa vir de outra fonte.

### 6. Os dois candidatos com progressão contínua
| | **B35** | **E13** |
|---|---|---|
| Andar 30 / 40 / 50 | 2,1h / 5,5h / 8,8h | 1,0h / 2,6h / 12,4h |
| Formato das mortes | sobe em ondas suaves (até ~8,6/h no 56–60) | calmo até o 35, depois **parede** no 46–55 (20 a 83 mortes/h) |
| Mortes em chefe | 16% | 11% |
| Spread entre classes no andar 30 | — (ladino incompleto) | 1,82× |

- **B35:** ritmo **regular**, sem parede dura em 48h. O herói chega a 60–80.
- **E13:** **rápido** até o 40 e **parede no andar 46–55**. Os degraus nos chefes concentram a dificuldade como planejado, mas o efeito composto `1,07ⁿ × 1,3^k` ainda vira parede no 50.
- **E15:** degrau grande demais. **Parede dura no chefe do 40** (só 1 de 9 rodadas chegou ao 50).

## Limitações

- **Política de jogador única:** a `spendClass` do dono. Ela **nunca compra na loja** e treina só FOR+CON. Não houve perfis "abaixo / adequado / acima do esperado" no jogo completo. Os perfis sintéticos estão em `*_synth.csv`, mas usam a camada B.
- **Duelo de chefe contestado:** o combat estima ≤7% de vitória no chefe do 20, o duelo deu 34%. Não foi resolvido; nenhuma conclusão deste documento depende desse número.
- **Ladino incompleto** em B35, D25 e E15 (congelamento).
- **3 seeds:** suficiente para tendências e ordem de grandeza, não para intervalos de confiança finos. Os valores de cada seed estão em `data/`.
- **Config padrão** (filtro comum, sem "descer direto"). Jogadores que mexem na config progridem diferente.
- **Não simulado:** mudanças do lado do herói (FOR proporcional, fonte de HP, penalidade de morte). As recomendações sobre isso no `06` são **hipóteses**.
