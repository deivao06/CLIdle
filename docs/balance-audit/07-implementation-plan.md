# 07 — Plano de implementação proposto

**Não executado.** Cada etapa termina com o dono testando antes da próxima. Todos os números novos vão em `balance.go`. Toda mudança que afete saves precisa respeitar a regra de compatibilidade (`saveVersion`, `migrate.go`).

## Etapa 0 — Bug do congelamento (R0)
- **Mudança:** `RandomFloor` com limite de tentativas ou varredura da sala; `enterLevel` e `placeLoot` tratam "sala cheia".
- **Teste:** sala mínima cheia de monstros + `placeLoot` termina; `TestAdvanceChunkingIsDeterministic` continua passando.
- **Release:** `fix:` independente do balanceamento. Pode sair antes de tudo.

## Etapa 1 — Decisões do dono (R1, R2)
- Descartar ou guardar o experimento A-quad (não publicar como está).
- Preencher o cronograma alvo no `design.md` (andar 10 / 30 / 50 / 70+).
- **Saída:** tabela de alvos. Sem ela, as etapas 3 e 4 não têm critério.

## Etapa 2 — Métrica de regressão (R3)
- Transformar a camada B do simulador num teste pulado por padrão que imprime golpes para matar e rodadas para morrer por andar, contra um herói de referência.
- Resolver antes a divergência do duelo de chefe (34% contra ≤7%): ordem do primeiro golpe, HP inicial e segundo fôlego.
- **Saída:** tabela de base A-exp para comparar as próximas etapas.

## Etapa 3 — Curva dos monstros (R4 + R10)
- `MonsterScale` linear, com g calibrado entre 0,30 e 0,40 contra o cronograma.
- Junto: revisar XP e ouro por kill, que hoje usam o mesmo `s`. Se `s` cai, a renda cai.
- Opcional: Def com escala menor que o HP.
- **Validação:** `tools/run-model.sh` com 3 seeds × 4 classes × 48h, mais `compare.py`. Critérios:
  - tempos dentro de ±25% do alvo;
  - nenhuma classe parada mais de 6h no mesmo andar;
  - spread entre classes no andar 30 abaixo de 1,6×.

## Etapa 4 — Chefes como marcos (R5)
- `spawnBoss`: multiplicadores que crescem com o número do chefe.
- **Validação:** mortes em chefe entre 25% e 50% do total, e nenhuma classe travada mais de X horas (X vem do cronograma).

## Etapa 5 — Lado do herói (R6, R7, R8, R9)
- FOR proporcional à arma, fonte de HP, penalidade de morte, crítico no `incoming()`.
- **Uma mudança por vez**, cada uma com a rodada completa do simulador, porque elas mexem no mesmo equilíbrio da etapa 3.
- R9 (`incoming`) pode ir antes: é correção de IA, não de números.

## Etapa 6 — Variedade (R11)
- Novos tipos de monstro para os andares 30+, com comportamento (velocidade, morto-vivo, sono, lentidão) em vez de só números.
- É conteúdo: o escopo é decidido pelo dono.

## Ordem sugerida e dependências

```
Etapa 0 ──────────────────────────────► release fix
Etapa 1 ─► Etapa 2 ─► Etapa 3 ─► Etapa 4 ─► Etapa 5 ─► Etapa 6
                         ▲            R9 pode ir a qualquer hora
```

## Riscos gerais
- **Saves existentes:** jogadores no andar 40+ com a curva nova podem ficar **fortes demais** de repente. Avisar nas notas da release.
- **Offline:** qualquer mudança de ritmo muda o que 12h offline entregam. O `TestTwelveHoursIsFastEnough` continua sendo o limite de desempenho.
- **Ladino:** já faz ~1,7× mais expedições e morre em 97% delas. Mudanças na morte (R8) o afetam mais que as outras classes.
