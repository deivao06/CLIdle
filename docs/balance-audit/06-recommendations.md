# 06 — Recomendações

Em ordem de prioridade. **Todas são decisões do dono.** Nada aqui foi implementado.

Legenda: **Impacto** = efeito esperado no jogo · **Evidência** = **S** (simulado), **C** (cálculo de fórmula), **H** (hipótese, ainda sem simulação).

| # | Recomendação | Impacto | Complexidade | Risco | Depende de | Evidência |
|---|---|---|---|---|---|---|
| R0 | Corrigir o congelamento em `RandomFloor` | Alto (bug) | Baixa | Baixo | — | **S** |
| R1 | Não publicar o A-quad como está | Alto | Nenhuma | Nenhum | — | **S** |
| R2 | Escrever um cronograma alvo | Alto | Baixa | Baixo | — | pesquisa (`03`) |
| R3 | Golpes para matar / rodadas para morrer como métrica fixa | Médio | Baixa | Nenhum | R2 | **C**, **S** |
| R4 | Crescimento linear por tipo, g ≈ 0,30–0,35 | Alto | Baixa | Médio | R2 | **S** (B25/B35) |
| R5 | Paredes nos chefes, só via `spawnBoss` | Médio | Baixa | Médio | R4 | **S** (E13/E15 mostram o que evitar) |
| R6 | Herói: FOR proporcional à arma | Médio | Baixa | Médio | R4 | **C** + **H** |
| R7 | Herói: fonte de HP que acompanhe o andar | Médio | Média | Médio | R4 | **C** + **H** |
| R8 | Rever a penalidade de morte | Médio | Baixa | Médio | R4 | **S** (ladino morre em 97%) + **H** |
| R9 | `incoming()` considerar crítico | Baixo | Baixa | Baixo | — | **C** |
| R10 | Def dos monstros escalar menos que o HP | Baixo agora / alto no 45+ | Baixa | Baixo | R4 | **C** |
| R11 | Mais tipos de monstro nos andares 30+ | Alto (variedade) | Alta (conteúdo) | Baixo | R4 | pesquisa + **S** (D) |

## R0 — Congelamento em `RandomFloor` (bug, não balanceamento)
- **Sistema:** `dungeon.go:Level.RandomFloor`, chamado por `enterLevel` e `placeLoot`.
- **Por quê:** o laço `for {}` sorteia uma casa até achar uma livre. **Não há limite.** Uma sala pequena (4×2 casas internas) pode encher de monstros e itens.
- **Quando:** andares fundos (12 monstros), ladino com Faro do tesouro. Reproduzido em **2 rodadas** (D25 e E15, ladino, seed 3, andar 42).
- **Correção proposta:**
  - devolver `ok=false` depois de N tentativas, ou varrer a sala em busca de uma casa livre;
  - quem chamou escolhe outra sala.
  - **Atenção ao determinismo:** a mudança altera a sequência do RNG só nos casos que hoje travam.
- **Risco:** baixo. Precisa de um teste que monte uma sala cheia.

## R1 — Não publicar o A-quad
- **Sistema:** `MonsterScale` + `MonsterMaxAge` (árvore de trabalho).
- **Por quê:**
  - com `MonsterWindow = 4`, o andar é dominado pelo tipo mais velho (n≈6–18);
  - nessa faixa, `1 + 0,026n + 0,035n²` **é maior** que `1,15ⁿ`;
  - medido: andar 30 em **3,16h** contra 1,68h, e andar 40 em 17,6h contra 12,1h.
- **Consequência:** se o objetivo era suavizar, ele faz o contrário até o andar 38.

## R2 — Cronograma alvo antes de mexer em números
- **Por quê:** "o ritmo ainda não foi ajustado jogando" (AGENTS.md). Sem alvo, não dá para dizer se B35 ou E13 é "melhor".
- **Proposta:** uma tabela no `design.md`:

  | Marco | Tempo de jogo (alvo) | Comentário |
  |---|---|---|
  | Andar 10 (1º chefe) | ? | |
  | Andar 30 | ? | |
  | Andar 50 | ? | |
  | Andar 70+ | ? | uma semana de offline? |

- **Para comparar:** A-exp dá 0,3h / 1,7h / nunca (48h). B35 dá 0,4h / 2,1h / 8,8h.

## R3 — Métrica fixa de golpes para matar
- **Por quê:**
  - "ameaça = HP × Atk" esconde a defesa subtrativa e o piso de dano 1;
  - "golpes para matar" e "rodadas para morrer" contra um herói de referência por andar resumem tudo num número (ver `03`, §3).
- **Como:** usar a camada B do simulador como **teste de regressão de balanceamento** (pulado por padrão), com faixas alvo por categoria: comum, mais velho do andar, chefe.

## R4 — Crescimento linear por tipo (g ≈ 0,30–0,35)
- **Sistema:** `balance.go:MonsterScale`.
- **Por quê:**
  - o herói cresce de forma **linear** com o andar (itens: `2 + 1,2·andar`) e **logarítmica** com o tempo (compras com custo `1,5^k` e `1,6^k`);
  - um monstro **exponencial** por idade cria a parede do 36–45, onde fica **80% do tempo**;
  - um monstro **linear** com g ≈ 0,35 deu progresso contínuo e mortes em ondas.
- **Quando o problema aparece:** a partir do andar ~30, quando o tipo mais velho da janela passa de n≈12.
- **Por que não menos:** g ≤ 0,25 chega ao andar 80 em 2–5h, **sem desafio**.
- **Consequências:**
  - XP e ouro por kill usam o mesmo `s`, então **a renda cai** no fundo. Isso precisa ser revisto junto, senão a economia fica mais lenta;
  - a partir do andar 24, a janela fica fixa em orc, esqueleto, lobisomem e dragão, e só os números crescem (ver R11).
- **Antes de fechar o número:** rodar g = 0,30 / 0,35 / 0,40 contra o cronograma do R2.

## R5 — Paredes nos chefes, só no chefe
- **Por quê:**
  - hoje só 4–12% das mortes são em chefe; a dificuldade está espalhada no corredor;
  - E13/E15 mostraram que um **degrau global** (`× B^k` para todos) vira parede de corredor de novo.
- **Como:** aumentar `BossHPMult`/`BossAtkMult` com o número do chefe (ex.: chefe do 40 mais forte que o do 20), **sem mexer nos monstros comuns**.
- **Risco:** mago e clérigo já travam no chefe do 40 hoje. Precisa de simulação por classe.

## R6 — FOR proporcional à arma
- **Sistema:** `entity.go:Hero.Atk` (`HeroBaseAtk + bonus(Weapon) + Attr(DamageAttr)`).
- **Por quê:**
  - no andar 30, +1 FOR ≈ **+1,9%** de dano, enquanto uma arma rara do mesmo andar dá +32 Atk (`02b` §3);
  - o custo do treino cresce `1,5^k` por ponto. É o pedido que já estava na fila do dono.
- **Opção mínima:** `Atk = 5 + arma × (100 + c·FOR)/100 + FOR`, ou seja, FOR também multiplica a arma.
- **Status:** **hipótese**. Precisa de simulação com a política de treino (`auditSpend`).

## R7 — Fonte de HP que acompanhe o andar
- **Sistema:** `entity.go:MaxHP`. Nenhum item dá HP (só o efeito Vigor, em %).
- **Por quê:**
  - Def é subtrativa e cresce com os itens, mas o HP só cresce por nível (de forma logarítmica no tempo) e por CON (cara);
  - poção cura 35% de um HP que fica pequeno: ~70 de cura contra golpes de 100+ no andar 30 (`02b`).
- **Opções:** armadura dá um pouco de HP **ou** `PotionHealPct` fixo + parte do andar. **Hipótese**, simular junto com R4.

## R8 — Penalidade de morte
- **Fato:** o ladino morre em **97%** das expedições e mesmo assim é quem vai mais fundo. Mantém XP, upgrades, poções e 50% do ouro.
- **Por quê importa:** num roguelike, a morte é o sinal de "não está pronto". Hoje morrer é a forma **normal** de farmar, então a parede **não comunica** nada ao jogador.
- **Opções:** perder parte do XP do nível atual, ou `DeathKeepGoldPct` menor no fundo. **Hipótese**. Cuidado: o idle precisa de progresso mesmo offline.

## R9 — `incoming()` considerar crítico
- `game.go:incoming` calcula o pior golpe sem ×2 de crítico (até 25%). A poção preventiva falha nos golpes que mais importam. Correção pequena.

## R10 — Def dos monstros escalar menos que o HP
- `spawnMonster` usa o mesmo `s` para HP e Def. O herói passa a tirar só **1** de dano quando `Def ≥ 2,4 × Atk do herói` (andares 45–56 no cenário do `02a`).
- Com R4 isso fica mais longe, mas continua sendo uma parede futura. Opção: `Def` com `√s`, ou um piso de dano percentual.

## R11 — Variedade de ameaças no fundo
- A partir do andar 24, o dragão é o último tipo. Toda a dificuldade depois vem de **inflar números**.
- **Pesquisa (`03`):** DCSS e Brogue fazem a dificuldade vir de **tipos novos com comportamento novo**.
- O CLIdle já tem as alavancas: `Undead`, `Spd`, sono, lentidão.
- **Conteúdo e design: decisão do dono.**
