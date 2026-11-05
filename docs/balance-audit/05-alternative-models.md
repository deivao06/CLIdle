# 05 — Modelos alternativos

Comparação com as mesmas métricas (dados no [04](04-simulation-results.md)). Patches em [`models/`](models/), feitos sobre o commit `2d21a25`. Em todos, `n = andar − MinDepth do tipo`.

## Resumo

| Modelo | Fórmula | Resultado em 48h | Veredito |
|---|---|---|---|
| **A-exp** (atual) | `1,15ⁿ` (`1,12ⁿ` após o 20) | andar 40 em 12h; **parede no 36–45**; chefe do 40 trava mago e clérigo | parede **acidental**: desgaste no corredor, não no chefe |
| **A-quad** (experimento) | `1 + 0,026n + 0,035n²`, idade máx. 20 | andar 30 em 3,2h (**o dobro do A-exp**) | **piora** o que pretendia melhorar |
| **B** linear | `1 + g·n`, g = 0,15 / 0,25 / 0,35 | g ≤ 0,25: andar 80 em 2–5h. **g = 0,35: andar 50 em 8,8h, sem parede dura** | B35 é o mais **regular** |
| **C** TTK alvo | HP e Atk seguem um herói de referência linear | andar 80 em 1,7h | conceito bom, **calibração ruim**. Útil como **métrica**, não como fórmula |
| **D** híbrido | linear + idade máx. 10 (mais rodízio de tipos) | D15 trivial; D25: andar 50 em 5,1h | idade menor **suaviza** e **diversifica** os tipos, mas precisa de g maior |
| **E** marcos | `1,07ⁿ × B^k`, k = chefes passados desde o tipo | E13: andar 40 em 2,6h, **parede no 46–55**; E15: **parede no 40** | concentra a dificuldade nos chefes, mas o efeito composto vira parede |

## Detalhes

### A — Atual (commit e experimento)
- **Vantagem:** já está publicado, e os jogadores conhecem o ritmo.
- **Problema:**
  - a ameaça por andar cresce de **1,25× a 1,37×** (exponencial) nas duas curvas;
  - o poder do herói cresce só **~20% em 18h** depois da 6ª hora;
  - resultado: **80% do tempo** fica nos andares 31–40.
- **A-quad:**
  - **não resolve**, porque a janela de 4 tipos deixa os tipos com idade n≈6–18 dominarem o andar;
  - é justamente nessa faixa que a quadrática é **mais forte** que a exponencial;
  - só fica mais suave a partir do andar 39, quando sobra só o dragão.

### B — Linear
- **O que mostra:** o "crescimento quadrático/exponencial" não é o único problema. Um linear fraco (0,15–0,25) faz o **herói ultrapassar os monstros**: os itens também são lineares, e o nível e o treino se somam a eles.
- **B35:**
  - progresso contínuo: andar 30 em 2h, 40 em 5,5h, 50 em 8,8h, 60+ em 48h;
  - mortes em ondas de 1 a 9 por hora, com picos nos andares de chefe;
  - 16% das mortes em chefe.
- **Risco:**
  - sem teto natural, o herói pode chegar a andares onde os números estouram (o simulador limita em 100 por isso);
  - **endgame indefinido**: a partir do andar 24, a janela fica fixa nos mesmos 4 tipos (orc, esqueleto, lobisomem, dragão), e só os números crescem.

### C — Orientado por TTK
- **Fato:** com a referência ajustada aos dados (Atk ≈ 4 + 4·andar, HP ≈ 20 + 9·andar), os monstros ficaram fracos demais. **Hipótese:** o herói real **supera a referência** porque a política de gasto e os níveis somam mais que a regressão linear previu.
- **Uso recomendado:** como **métrica de controle** ("golpes para matar" e "rodadas para morrer" por andar, contra um herói de referência), e não como fórmula de produção. Ver R3 no [06](06-recommendations.md).

### D — Híbrido (idade máxima menor)
- `MonsterMaxAge = 10` tira tipos velhos do sorteio mais cedo. O andar passa a ter **monstros mais novos e variados**, e o tipo mais velho deixa de dominar.
- D25 (andar 50 em 5,1h, 9% das mortes em chefe) é mais fácil que B35 com o mesmo g. **A idade máxima é uma alavanca de dificuldade**, como o combat previu (R2 do `02a`).
- **Limitação:** o jogo tem só 10 tipos. Com idade 10, o fim do jogo vira "só dragão" a partir do **andar 29** (o lobisomem, `MinDepth 18`, sai depois do 28). A variedade de ameaças precisaria de **mais tipos** (conteúdo, não números).

### E — Marcos (degrau no chefe)
- **Ideia (do `03`, ONI/idle):** a parede deve estar no **chefe**, com saída clara pelo farm.
- **E13:**
  - corredor fácil até o andar 40;
  - depois, o degrau acumulado (`1,3^k`) vira parede no 46–55, com 83 mortes por hora no 51–55;
  - as mortes em chefe são só 11%, ou seja, a parede **voltou para o corredor**.
- **E15:** parede dura no chefe do 40 (só 1 de 9 rodadas passou do 50 em 48h).
- **Conclusão:**
  - **degrau multiplicativo composto** repete o problema exponencial em outra escala;
  - um marco que funcione precisa ser **por chefe** (HP/Atk do chefe), e não um multiplicador que se arrasta para todos os monstros dos andares seguintes.

## Recomendação de abordagem

1. **Não publicar o A-quad como está.** Ele deixa o meio do jogo **2× mais lento** (andar 30).
2. **Base da curva:** linear por tipo, com g ≈ **0,30–0,35**, calibrado contra um **cronograma alvo** decidido pelo dono (ex.: andar 30 em ~2h, 50 em ~10h, 70+ em uma semana).
3. **Paredes intencionais nos chefes:** ajustar só `spawnBoss` (`BossHPMult`, `BossAtkMult`), sem multiplicador global.
4. **Variedade, não inflação:** nos andares 30+, a diversidade depende de **novos tipos** de monstro. Hoje o dragão é o último. É conteúdo, e a decisão de criar é do dono.
5. **O lado do herói** (FOR proporcional, fonte de HP, penalidade de morte) precisa de uma **segunda rodada de simulação** antes de qualquer número final.
