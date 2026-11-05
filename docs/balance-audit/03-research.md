# 03 — Pesquisa: roguelikes e idle games

Auditoria de balanceamento do CLIdle. Este arquivo só traz **o que outros jogos e autores fazem** e se isso serve para o CLIdle. Não lê o código a fundo (isso é do **combat**, **progression** e **sim**).

Legenda:
- **Fato documentado:** está na fonte citada, que foi aberta.
- **Interpretação:** leitura minha para o CLIdle. Pode estar errada.

---

## 1. Roguelikes tradicionais

### 1.1 NetHack — dificuldade = média de andar e nível do herói

**Fato documentado** ([NetHack Wiki — Monster difficulty](https://nethackwiki.com/wiki/Monster_difficulty)):
- Monstro é "forte demais" para ser gerado se a dificuldade dele for **maior que a média do nível de experiência e do andar**: máx = `(XL + andar) / 2`.
- "Fraco demais" se for **menor que `andar / 6`** (arredondado para baixo).
- A regra é de **seleção de tipo**, não de força do monstro.

**Interpretação para o CLIdle:**
- **Aplica parcialmente.** O CLIdle hoje escolhe tipo só por andar e **infla stats** por andar. O NetHack faz o contrário: o tipo varia, o stat não.
- Ligar a seleção ao nível do herói pune quem sobe de nível (rubber band). Num idle, onde o jogador **quer ver número subir**, isso pode tirar a sensação de progresso. Usar só como teste (modelo M4).

### 1.2 Dungeon Crawl Stone Soup — janela de profundidade por tipo

**Fato documentado** ([mon-pick-data.h](https://github.com/crawl/crawl/blob/master/crawl-ref/source/mon-pick-data.h), [random-pick.h](https://raw.githubusercontent.com/crawl/crawl/master/crawl-ref/source/random-pick.h)):
- Cada monstro tem **andar mínimo, máximo, peso e forma**. Exemplos do Dungeon: rato 1–3, goblin 1–4, orc 3–8, ogro 4–7 (PEAK), troll 9–14 (PEAK), hidra 13–23 (SEMI).
- Fora da janela o monstro **não aparece**.
- Formas do peso (`rarity_at`):
  - **FLAT:** 100% na janela toda.
  - **SEMI:** 100% no meio, 50% nas pontas.
  - **PEAK:** `peso × (len+2 − |min+max−2·andar|) / (len+2)`, quase zero nas pontas.
  - **RISE / FALL:** sobe ou desce linear na janela.

**Fato documentado** ([crawl docs — levels/introduction.txt](https://github.com/crawl/crawl/blob/master/crawl-ref/docs/develop/levels/introduction.txt)):
- "Crawl é difícil, mas tente equilibrar seus monstros."
- "Não use monstros OOD (fora de profundidade) a menos que saiba bem o que quer."
- Um monstro que era ameaça média **dez ou mais andares antes** provavelmente não ameaça mais.

**Fato documentado** ([CrawlWiki — Experience](http://crawl.chaosforge.org/Experience), página marcada como possivelmente desatualizada):
- XP total por nível: 10, 30, 70, 140 … 3.910 (nv 10) … 74.000 (nv 15). O intervalo **quase dobra** no começo; do nível 16 em diante cresce **6.000 fixos** por nível.

**Interpretação para o CLIdle:**
- **É o modelo que mais combina com o "estilo Rogue/NetHack" do design.** Dificuldade vem de **tipos novos**, não de "rato do andar 30 com 20× HP".
- O CLIdle tem andares **infinitos**; o DCSS não. Para infinito, a janela precisa de uma regra para depois do último tipo (ex.: stat cresce devagar só depois do último tipo da tabela).
- Não verifiquei se o DCSS escala stats por andar. Não afirmo isso.

### 1.3 Brogue — hordas e fora de profundidade

**Fato documentado** ([Brogue Wiki — Monster generation](https://brogue.wiki/wiki/Monster_generation); a página tem aviso "[verify]"):
- "A partir do nível 2, **10% das hordas** são geradas fora de profundidade."
- São escolhidas "como se tivessem nascido **1dK níveis mais abaixo**, com K = min(andar/2, 5)". Exemplo da página: ogros (mín. 7) aparecem no 5.
- Andares mais fundos têm **monstros novos e hordas maiores**.
- A cada 125–175 turnos nasce uma horda nova, longe do herói.

**Interpretação para o CLIdle:**
- **Aplica.** Um pouco de surpresa (10%) dá variância sem inflar a média. Bom para o "momento" de risco que o idle precisa ver acontecer.
- O spawn periódico **não** aplica: o herói do CLIdle explora tudo e sai; spawn contínuo só alongaria o andar (já lento, ~110 s no andar 1).

### 1.4 Tales of Maj'Eyal — zona escala com o jogador

**Fato documentado** ([TE4 — Dev-Zones](https://te4.org/wiki/Dev-Zones), seção marcada "fact check me"):
- `level_scheme = "player"`: zona e NPCs sobem ao nível do jogador, limitados por `level_range`.
- Exemplo: zona com `level_range {10,20}`, jogador nível 12 → andar 1 nível 12, **+1 por andar**.
- Nível do NPC: `base_level + ajuste_de_rank + (andar − 1) + rng(−1, 2)`.

**Interpretação para o CLIdle:**
- **Não recomendo para o eixo principal.** Escalar o mundo ao herói apaga a sensação de ficar mais forte, que é o motor de um idle. Só faz sentido com **teto** (`level_range`), como o próprio ToME faz.

### 1.5 Tutorial de roguelike (libtcod, parte 12)

**Fato documentado** ([rogueliketutorials — part 12](https://www.rogueliketutorials.com/tutorials/tcod/v2/part-12/)):
- Máx. monstros por sala: 2 (andar 1–3), 3 (4–5), 5 (6+). Itens: 1 → 2.
- Peso do troll **substituído** por andar: 15 (3–4), 30 (5–6), 60 (7+); orc fixo em 80.

**Interpretação:** é a versão mínima do modelo DCSS — **dificuldade por mistura de tipos e quantidade**, sem multiplicar stats.

---

## 2. Idle / incremental

### 2.1 Custo exponencial vs produção polinomial = parede

**Fato documentado** ([Pecorella — The Math of Idle Games, Part I](https://www.gamedeveloper.com/design/the-math-of-idle-games-part-i)):
- `custo_próximo = base × taxa^possuídos`; `produção = base × possuídos × multiplicadores`.
- Exemplo AdVenture Capitalist: taxa **1,07**.
- Custo cresce exponencial, produção linear/polinomial: no começo sobra; depois o exponencial **sempre alcança** e a espera fica proibitiva.
- **Multiplicadores por marco** (×2 aos 25 e 50) puxam a produção de volta para perto do custo e geram picos de compra.
- Prestígio deixa o jogador "ir mais longe na curva de custo antes de ficar para trás".

**Interpretação para o CLIdle:**
- **Aplica direto.** No CLIdle o "custo" é a força do monstro por andar (exponencial ou quadrática) e a "produção" é o poder do herói (nível, ferreiro, treino, loot — tudo mais perto de linear).
- **A parede é inevitável; a pergunta é onde ela fica.** Sem prestígio (ainda "para depois"), a parede é o fim do jogo.
- **FOR (pendente do dono):** +1 FOR = +1 Atq é produção linear pequena contra uma arma que cresce com o andar. É o padrão "gerador sem multiplicador". Saída clássica: **marcos** (ex.: bônus a cada N pontos) ou FOR em **%** do dano da arma.

### 2.2 Paredes, marcos e chefes (ONI Games)

**Fato documentado** ([ONI — How to design an idle game](https://onirealms.com/guides/how-to-design-an-idle-game/)):
- Idle Ronin: HP do inimigo **+40–57% por fase no início**, depois **14,5%** (após fase 140), 15,5% (500), 15% (1000).
- Chefes de fim de fase: **6× HP**, **10×** nas fases terminadas em 4 ou 9 → **parede a cada 5 fases**.
- "Ponha paredes onde existe uma resposta clara." Perder para o chefe volta o jogador ao farm, **não ao começo**.
- Idle AI Factory: produção **dobra** nos níveis 10, 25, 50, 75, 100.
- Escolha **primeiro o cronograma** (ex.: nível 5 no dia 1, 15 em ~2 semanas), depois ajuste a curva a ele. XP do Block Brain: `round(19,7 × nível^2,3)`.
- Offline: tetos de 2–8 h; ignorar pausas < 3 min.
- Prestígio: **o primeiro deve vir cedo e valer claramente a pena**.

**Interpretação para o CLIdle:**
- **Aplica muito.** O CLIdle já tem chefe a cada 10 andares e atalho; falta o chefe ser **a** parede (com a resposta clara: "volte, compre, treine"), em vez de a parede vir espalhada por todo andar.
- A morte no CLIdle já segue "volta ao farm, não ao começo" (mantém XP, poções, equipamento inicial). **Coerente.**
- Taxa por andar de 12–15% do design está **na mesma faixa** do Idle Ronin pós-fase 140 (14,5–15,5%). Mas o Idle Ronin compensa com produção multiplicativa; o CLIdle não tem multiplicador.
- **"Cronograma primeiro"** é a recomendação mais útil: o dono deve dizer onde quer o herói em 1 h, 12 h, 1 semana. Hoje o "ritmo definido jogando" (design) não tem alvo escrito.

### 2.3 Prestígio

**Fato documentado** ([Pecorella — Part III](https://www.gamedeveloper.com/design/the-math-of-idle-games-part-iii)):
- Fórmulas de moeda de prestígio usam **raiz** do ganho (AdCap `150·√(c/10¹⁵)`, Cookie Clicker `∛(c/10¹²)`, Egg Inc. expoente ~0,14).
- Para dobrar a moeda é preciso ganhar ~3–4× (AdCap, Realm Grinder) até 128× (Egg Inc.).
- Variar o ponto de reset; marcos alongam corridas.

**Interpretação:** **não aplica agora** (prestígio está em "Para depois"). Quando vier: medir em **andar máximo** (não ouro), com raiz, e o 1º reset perto do 1º chefe difícil.

---

## 3. Combate: TTK / hits-to-kill

**Fato documentado** ([Thumbsticks — Outer Worlds 2](https://www.thumbsticks.com/how-obsidian-fixed-combat-in-the-outer-worlds-2/)):
- Obsidian recalibrou os tiers por **número de golpes para matar**, porque os jogadores chegavam a ~36 de dano contra tiers feitos para 17.
- "Hits to kill é um jeito do designer **abstrair todas as entradas**."
- "Sempre erramos começando baixo e subindo." O chefe final foi endurecido de propósito.

**Fato documentado** ([Game Developer — Gameplay progression](https://www.gamedeveloper.com/design/gameplay-design-fundamentals-gameplay-progression)):
- Dificuldade em **curva**, não linear: "cada passo maior que o anterior", começo fácil.
- Recompensas rápidas no início e mais lentas depois.
- Estimar taxas de falha (mortes) numa **tabela de progressão** e validar testando.

**Interpretação para o CLIdle:**
- **Aplica muito.** O CLIdle tem muitas entradas (FOR, arma, ferreiro, DES → erro/crítico, defesa, efeitos). Medir **golpes para matar** e **rodadas para morrer** por andar deixa o balanceamento legível.
- Combate é 1 rodada/s: golpes-para-matar = segundos de luta. Isso conecta direto com o ritmo do idle.

---

## 4. Princípios e se valem para o CLIdle

| Princípio | Fonte | Vale? | Por quê |
|---|---|---|---|
| Dificuldade por **tipo novo**, não por inflação | DCSS, Brogue, tutorial | **Sim** | Combina com o estilo Rogue; mantém TTK estável |
| Parede vem de exponencial vs polinomial | Pecorella I | **Sim** | Explica a parede do fim do jogo e a FOR fraca |
| Parede no **chefe**, com resposta clara | ONI | **Sim** | Chefe a cada 10 + atalho já existem |
| **Cronograma alvo** antes da curva | ONI, Game Developer | **Sim** | "Ritmo jogando" não tem alvo escrito |
| Hits-to-kill como métrica | Obsidian | **Sim** | Abstrai muitas entradas |
| 10% fora de profundidade | Brogue | **Talvez** | Variância boa; pode aumentar mortes |
| Dificuldade pelo nível do herói | NetHack, ToME | **Cuidado** | Rubber band tira a sensação de progresso do idle |
| Spawn periódico | Brogue | **Não** | Alongaria andares já lentos |
| Prestígio por raiz | Pecorella III | **Depois** | Prestígio ainda não existe |

**Diferença importante:** roguelike tradicional é **uma corrida curta e finita** com morte permanente; o CLIdle é **progressão infinita e persistente**, com a morte custando só a expedição. Do roguelike vem a **forma** da dificuldade (tipos, janelas, OOD); do idle vem o **ritmo** (curvas, paredes, marcos, cronograma).

---

## 5. Modelos testáveis (enviados ao **sim**)

`n` = andares abaixo do 1º andar do monstro · `d` = andar · `L` = nível do herói.

1. **M1 — Janela por tipo (DCSS):** cada tipo tem `[minD, maxD]` e peso PEAK; stats `1 + g·min(n, W)`, `W ≈ 5–6`. Dificuldade vem de tipos novos.
2. **M2 — Escada por marcos (Idle Ronin/AdCap):** `r^n × B^⌊d/10⌋`, `r ≈ 1,06–1,08`, `B ≈ 1,3–1,5`. Parede no chefe.
3. **M3 — Faixa de TTK:** `HP_m = H × dano_herói_ref(d)`, `Dano_m = HP_herói_ref(d) / K`; `H = 3–5` (normal), `15–25` (chefe), `K = 8–12` rodadas sem poção. **Primeiro como métrica** do modelo atual.
4. **M4 — Seleção NetHack:** tipo permitido se `d/6 ≤ minD ≤ (d + L)/2`. Só seleção, stats sem mudar.
5. **M5 — Fora de profundidade (Brogue):** do andar 2 em diante, 10% dos spawns escolhem tipo como no andar `d + 1dK`, `K = min(d/2, 5)`.

**Métrica comum:** razão `poder do monstro(d) / poder do herói` no andar máximo, hora a hora em 12 h.

---

## 6. Fontes que não consegui acessar ou não verifiquei

- [Kongregate blog — The Math of Idle Games I](https://blog.kongregate.com/the-math-of-idle-games-part-i/): **404**. Usei a cópia no Game Developer.
- [Brogue fandom — Monster Hordes](https://brogue.fandom.com/wiki/Monster_Hordes): **402**. Usei o brogue.wiki (com aviso "[verify]").
- [TE4 — Zone Level](https://te4.org/wiki/Zone_Level): **404**. Usei Dev-Zones.
- **ADOM:** não pesquisado; nada dele é citado aqui.
- **NetHack, fórmula de XP** e **DCSS, escala de stats por andar:** não verificados; não afirmo nada sobre eles.
- Fonte genérica sobre TTK em RPG (fórum/itch.io) apareceu na busca, mas **não abri**; só cito o artigo da Obsidian.
