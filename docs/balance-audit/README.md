# Auditoria de balanceamento — CLIdle

Base: commit `2d21a25` (v0.2.1) e o experimento não commitado do dono (`MonsterScale` quadrático + `MonsterMaxAge`, guardado em `models/A-quad.patch`). A auditoria em si não alterou código de produção.

**Depois da auditoria (decisões do dono):**
- R0 feito: congelamento em `RandomFloor` corrigido (`1ea41fd`).
- Curva **B45** aplicada (`b9415f0`). O dono não quer metas de tempo e pediu algo parecido com o B35, mas sem ficar fácil. Calibração em [`data/calibration-B35-B45.md`](data/calibration-B35-B45.md).
- R5 aplicada: chefes mais fortes a cada chefe (HP ×(5+N−1), Atk ×(1,4+0,1·(N−1))). Mortes em chefe passaram de 4% para 22%. Dados em [`data/calibration-bosses-R5.md`](data/calibration-bosses-R5.md).
- Próximas da fila: R6–R11.

## Resumo executivo

1. **Diagnóstico.**
   - A parede não vem de "monstros quadráticos".
   - Ela vem de **monstros que crescem de forma exponencial por andar** (1,25×–1,37× por andar, nas duas curvas, por causa da janela de 4 tipos) contra **um herói que cresce de forma linear por andar e logarítmica no tempo**: os itens são lineares e as compras ficam cada vez mais caras.
2. **Problemas principais.**
   - **80% do tempo** de jogo fica nos andares 31–40.
   - As mortes por hora sobem **6×** e depois **4×** entre faixas, enquanto o XP sobe só 1,3× e 1,5×.
   - A morte quase não pesa: o ladino morre em **97%** das expedições.
   - **Bug real:** o jogo pode **congelar** em `RandomFloor` (sala cheia).
3. **Gargalos.**
   - Andares **36–45**.
   - **Chefe do 40:** mago e clérigo ficam parados nele de 32 a 41h.
   - Primeiro pico no **chefe do 30**.
4. **Comparação de modelos.**
   - O **experimento quadrático piora** o meio do jogo: andar 30 em **3,2h**, contra 1,7h no commit.
   - **Linear fraco** (g ≤ 0,25) acaba com o desafio: andar 80 em 2–5h.
   - **Linear g = 0,35** dá progresso contínuo: andar 50 em 8,8h.
   - **Degrau global no chefe** (E) volta a criar parede no corredor.
5. **Abordagem recomendada.**
   - Escrever um **cronograma alvo**.
   - Monstros comuns com **crescimento linear por tipo** (g ≈ 0,30–0,35).
   - Paredes **só nos chefes**.
   - Variedade por **novos tipos**, não por inflar números.
6. **Prioridades.**
   - R0: corrigir o congelamento.
   - R1: não publicar o A-quad.
   - R2: escrever o cronograma.
   - R3: criar a métrica de golpes para matar.
   - R4: curva linear.
   - R5: chefes como marco.
   - R6–R9: lado do herói (FOR, HP, morte, crítico no `incoming`).
   - R10–R11: Def e variedade.
7. **Riscos.**
   - A renda cai junto com o `s` (XP e ouro usam o mesmo fator).
   - Saves no andar 40+ ficam fortes demais de repente.
   - O ritmo do offline muda.
   - As mudanças no herói ainda **não foram simuladas**.
8. **Próximos passos.** Etapa 0 (bug) e etapa 1 (decisões do dono). Ver o [07](07-implementation-plan.md).

## Documentos

| # | Arquivo | Autor |
|---|---|---|
| 01 | [Arquitetura e sistemas](01-architecture-systems.md) | lead |
| 02a | [Combate e monstros](02a-combat-monsters.md) | combat |
| 02b | [Progressão e economia](02b-progression-economy.md) | progression |
| 03 | [Pesquisa](03-research.md) | research |
| 04 | [Resultados da simulação](04-simulation-results.md) | lead, com os dados do sim |
| 05 | [Modelos alternativos](05-alternative-models.md) | lead |
| 06 | [Recomendações](06-recommendations.md) | lead |
| 07 | [Plano de implementação](07-implementation-plan.md) | lead |

## Ferramentas

- **Simulador:** `internal/game/balance_audit_test.go` (pulado sem `BALANCE_AUDIT=1`).
- **Modelos:** `models/*.patch`, aplicados sobre `HEAD`.
- **Rodar um modelo:** `docs/balance-audit/tools/run-model.sh <modelo> [horas] [seeds]`. Cria um worktree em `/tmp/clidle-audit` e roda as 4 classes em paralelo.
- **Comparar:** `python3 docs/balance-audit/tools/compare.py /tmp/clidle-audit/sim-out A-exp B35 ...` (tabela), `analyze.py` (por faixa de andar).
- **Dados desta auditoria:** [`data/`](data/).

## Limitações

- Uma política de jogador só (cópia da `spendClass` do dono; não compra na loja).
- 3 seeds por modelo.
- Ladino incompleto em B35, D25 e E15 (congelamento).
- Taxa de vitória contra chefes no duelo isolado contestada pela revisão cruzada; não foi usada nas conclusões.
- O ADOM não foi pesquisado; três fontes deram 404/402 (ver `03` §6).
- **Contradição resolvida:** o `02b` dizia que o ouro "sobra sem uso" no fundo. A simulação mostra que o ouro fica **guardado para compras cada vez mais caras** (`04` §3). O efeito no poder é o mesmo; a causa é a escala dos custos.
