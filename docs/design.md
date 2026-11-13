# clidle — Design

Idle game de terminal, fantasia medieval, dungeon explorer estilo roguelike clássico (NetHack/Rogue).
A ideia é deixar um terminal pequeno rodando e ter uma progressão satisfatória.

## Base

- **Objetivo:** uso pessoal, código limpo o bastante para publicar depois.
- **Stack:** Go + Bubble Tea.
- **Interação:** idle + decisões. Roda sozinho; o jogador entra a cada 10–30 min para decidir.
- **Offline:** ao reabrir, roda a simulação real (mesma lógica, sem desenhar) pelo tempo fora, teto de 12h, e mostra um resumo.
- **Save:** `~/.local/share/clidle/save.json`, a cada 30s e ao sair.

## Layout

- Adaptável ao tamanho do terminal, com breakpoints:
  - **Pequeno:** mapa + essencial (HP, andar, ouro, último evento).
  - **Médio:** + log.
  - **Grande:** + stats/inventário e legenda em painel lateral.
- Abaixo de **40×10**: aviso "janela muito pequena".
- Telas: `1` Principal · `2` Preparação · `3` Herói · `4` Config · `5` Nerd (só com a opção ligada) · `?` legenda/ajuda · `q` sair.
- **Principal** muda sozinha: mostra a dungeon durante a expedição e a cidade quando o herói volta.
- **Preparação** fica sempre disponível e só configura as expedições, em 4 grupos: Expedição · Poções · Loot · Táticas. Mudanças valem na hora, inclusive com o herói na dungeon; só poções e andar inicial esperam a próxima partida. Exceção: a regra de tática em edição só vale ao Salvar (2026-11-11: o dono temia perder um chefe com a regra pela metade); Cancelar, esc ou trocar de tela perguntam se descarta as mudanças. Apagar regra vale na hora (já pede confirmação).
- **Cidade (tela 1 com o herói na cidade):** à esquerda, o `@` descansa junto a uma fogueira animada numa salinha no estilo do mapa; à direita, a lista navegável: Partir agora, Ferreiro, Treino, Loja (com a mochila) e Baú. Em tela estreita (< 100 colunas), a sala da fogueira fica em cima e a lista embaixo, se sobrarem 10+ linhas para a lista; senão, só a lista. Ferreiro, treino, loja, mochila e baú **só funcionam na cidade**; não há fila. Para comprar no meio de uma expedição, o jogador força a volta (`r`). A auto-expedição não espera o jogador.
- **Cores na cidade:** ferreiro, loja e baú mostram o item na cor da raridade; o que custa mais que o ouro atual fica esmaecido, sem perder a cor. Treino e mochila ficam no branco dos itens comuns quando dá para pagar e cinza quando não dá.
- **Versão:** no canto direito do rodapé, em todas as telas, quando cabe.
- **Linhas longas nas listas:** no desktop e no celular, uma linha mais larga que a lista quebra (até 4 linhas) em vez de ser cortada; a continuação fica alinhada sob os atributos e quebra entre efeitos, nas vírgulas.
- **Linha de descrição:** nas listas (cidade, Preparação, Config), o rodapé explica o item selecionado. O treino mostra o efeito real do próximo nível (ex.: `Atq 8 → 9`); loja e baú comparam com o item equipado.
- **Herói (tela 3):** duas colunas quando cabem (≥ 84 colunas): à esquerda atributos, barras, **Combate** (ataque, defesa, chance de erro, crítico, esquiva do ladino, velocidade, mana por passo e quanto cada poção cura, das mesmas fórmulas do jogo), equipamento, expedição e recordes; à direita todas as habilidades da classe, liberadas (✓, branco) e bloqueadas (·, cinza), com nível, custo de MP e o que fazem, e a última morte: um desenho do mapa em volta do herói no momento da morte (recorte de 31×11 guardado no save, como o herói via; herói morto como `@` em vermelho, monstro que matou sublinhado; linhas nunca vistas somem) e as últimas linhas do log. Save antigo mostra só o log. Em tela estreita, uma coluna com rolagem. O ouro fica só no topo da tela. O painel lateral da dungeon continua compacto, sem o bloco Combate.
- **Config** guarda as opções do jogo (bip, idioma, **efeitos no mapa**: completo/sutil/desligado; sutil deixa a lamparina parada e só números e flash) e **Resetar save**, que pede confirmação, apaga o save e volta para a escolha de classe.
- **Novidades:** página na ajuda (`?`) com as notas das 10 últimas releases do GitHub (em inglês, como as notas). Abre sozinha uma vez depois de atualizar (`Game.SeenNews`); jogo novo e build `dev` não abrem. Sem internet, mostra um aviso e tenta de novo ao reabrir a página. Não virou aba porque a 6ª aba aperta o celular.

## Visual

- Mapa ASCII estilo Rogue; o jogador vê o `@` explorando sozinho.
- Andar de tamanho fixo (~80×21); a câmera segue o `@` em terminais menores.
- Névoa de guerra.
- **Impacto no combate:** o dano sobe flutuando em cima do alvo (~0,7s): claro no golpe do herói, laranja com `!` no golpe forte/crítico/furtivo, vermelho com `-` no dano recebido. O alvo pisca no golpe. Só golpes grandes (golpe forte, crítico, furtivo, ou dano recebido ≥ 1/4 do HP máximo) tremem o mapa 1 coluna por ~0,12s. O jogo só avisa a tela por `Game.Events`, fora do save.
- **Momentos:** monstro morto vira um `*` laranja que apaga em `·` (~0,3s; chefe estoura nas 8 casas em volta). Item raro ou melhor pego brilha nas diagonais do herói na cor da raridade (~0,6s; lendário também em cruz). Subir de nível solta um anel dourado que se expande do `@` (~0,7s).
- **Lamparina:** um raio de luz quente com cintilação em volta do `@` (mesmo degradê da fogueira). Ilumina só o que o herói vê e corredores conhecidos, nunca outra sala através da parede. Só visual: usa o relógio da tela, nunca o RNG do jogo. Monstros, itens e escadas mantêm a cor própria.
- **Linha de status** acima do log, em todos os tamanhos: o que o herói está fazendo (explorando, lutando com X, indo pegar item, indo para a escada, indo enfrentar o chefe, voltando para a cidade e o motivo).
- **Ajuda (`?`)** é um manual com páginas (←→ troca, ↑↓ rola): **Símbolos** (legenda de cada caractere), **Como funciona** (expedição, ordem das ações do herói, retirada, cidade, morte, chefes, auto, offline) e **Atributos** (FOR/DES/INT/CON, números do herói, raridade, classes).
- Monstros por letra (`r`, `g`, `o`...); chefes em letra maiúscula com cor própria. ~8–10 tipos na v1.
- **Tema único (clássico corrigido):** cada tipo de informação tem cor própria. Laranja só em abas, títulos e cursor; seções em branco negrito; linha de atributos em bege; herói `@` em amarelo forte e linha `▸` em amarelo claro (ouro é outro amarelo); XP em ciano; paredes mais escuras que o texto do log; lendário vermelho-alaranjado para não parecer título.
- Raridade de itens por cor: Comum (branco), Mágico (azul), Raro (verde), Épico (roxo), Lendário (laranja).
  - No mesmo andar, raridade maior nunca sai mais fraca que a menor (variação de ±10%; botas sem variação).
- **Efeitos:** raro tem 1 efeito, épico 2, lendário 3 (diferentes; a faixa de valor não depende da raridade, estilo Path of Exile simplificado): roubo de vida, crítico (dano dobrado), HP extra, ouro extra, cura de poção extra, XP extra. Valores em `EffectRange`; o mesmo efeito em itens diferentes soma até o teto `EffectCap` (sem teto, 5 peças davam 75% de roubo de vida). Itens antigos guardam os efeitos que já tinham.

## Expedições

- O herói entra na dungeon, sai quando quiser e guarda o loot.
- **Morte:** perde 75% do ouro (`DeathKeepGoldPct`) e os itens achados na expedição; mantém nível/XP, poções e o equipamento que saiu da cidade. Volta para a cidade.
- **Volta:** o herói sai pela escada `<` do andar atual. Na retirada (qualquer motivo) ele foge: ignora combate, só se cura (poção, magia de cura, segundo fôlego) e ataca apenas o monstro que bloqueia o único caminho; o ouro é guardado e a mochila é vendida automaticamente.
- **Retirada automática:** regras Recuar nas táticas (padrão: mochila cheia; sem poções com HP < 30%; "último andar" é uma regra "terminou o andar ≥ N → Recuar", que vale ao chegar na escada `>`).
- **Início:** o jogo começa na cidade; a expedição parte com `e` ou "Partir agora" na lista da cidade.
- **Auto-expedição:** liga/desliga (desligada por padrão). Ao terminar, inicia outra com a mesma configuração.
- Mandar o herói voltar manualmente desliga a auto-expedição; a retirada automática não.
- **Auto-equipar:** ligado por padrão. Desligado, o herói não troca equipamento na dungeon; o jogador equipa pelo baú.
- **Vender loot na volta:** ligado por padrão, independente do auto-equipar. Desligado, a mochila vai para o **baú** (sem limite por enquanto; um limite fica para o futuro). Na lista da cidade o baú mostra só os itens guardados (não os equipados): enter equipa (o item que estava equipado vai para o baú) e `x` vende. Saves antigos com auto-equipar desligado começam com a venda desligada (passo v5).
- **Item level:** todo item guarda o andar onde nasceu (`Item.Floor`; loja usa o andar mais fundo alcançado, chefe o andar dele) e mostra junto ao nome, `Brigandine +6 (iLvl 34)`, em todas as listas e no log. Explica por que um item de raridade menor e andar maior bate um de raridade maior e andar menor. Itens de saves antigos e a arma inicial não têm andar e aparecem sem o número.
- **Compra na loja:** o item comprado é equipado na hora e o que estava equipado vai para o baú (qualquer raridade, nunca é vendido).
- **Lendários nunca são vendidos sozinhos:** na volta vão sempre para o baú (mesmo com a venda ligada). Na dungeon segue a regra normal: com a mochila cheia, o item trocado fica para trás. Só saem vendidos com `x` e confirmação.
- **Ajuda (`?`) tem 5 páginas:** Símbolos, Como funciona, Táticas, Atributos e Classes. A página Atributos explica cada efeito de item épico/lendário (com o mesmo nome que aparece no item e a faixa de valores). A página Classes é montada da tabela de habilidades (nunca fica desatualizada): armas de cada classe, atributo principal e cada habilidade com nível, MP e descrição.
- **Ajuda (`?`) acompanha as regras:** toda regra nova que o jogador sente (combate, venda, mana, monstros) entra no manual.
- **Filtro de loot:** "Pegar itens" (comum+ por padrão) define a raridade mínima que vai para a mochila. Item que o herói vai equipar é pego sempre.
- **Itens de outra classe:** desligado por padrão; o herói só pega o que a classe dele usa.
- **Limite de mortes:** configurável (padrão sem limite). Depois de N mortes desde que a auto-expedição foi ligada, ela desliga e o herói fica na cidade. Religar zera a contagem.
- Herói novo começa com 5 poções. Na Loja (cidade) dá para comprar 1 por vez, até o limite de 10. Na Preparação, "Comprar automaticamente" define até quantas o herói completa sozinho antes de partir (se tiver ouro); o preço aparece só na Loja.
- **Poções:** por padrão, bebidas abaixo de 40% de HP (regra das táticas). O herói nunca sabe se um golpe vai matá-lo: não existe regra de poção antes de golpe letal. O herói guarda no máximo 10; o que passar disso é vendido ao chegar na cidade (metade do preço).

## Dungeon

- Andares infinitos, cada vez mais fundos. Chefe a cada 10 andares.
- Ouro: 1 a 3 pilhas por andar (+8% por andar), mais o XP ÷ 8 de cada monstro morto, que é a maior fonte. O dono achou o ouro fácil demais (2026-11-10): eram XP ÷ 2, +12% por andar e metade do ouro guardada na morte. Itens vendem por pilhas de ouro do andar deles (2026-11-11; antes era bônus × 4 × (raridade + 1) × 60%, que crescia devagar contra o ouro e deixava botas e efeitos quase sem valor): o andar sai do atributo principal comparado ao normal do tipo (`ItemDepth`), comum vale 1 pilha, a raridade multiplica (×1 a ×5) e cada efeito soma até +1 pilha conforme a força. Mais 20% do ouro gasto no ferreiro naquele item. A loja cobra 5× o valor (mesma proporção de antes, quando era 3× o valor cheio e a venda 60%).
- Monstros crescem de forma linear: +45% dos atributos-base por andar abaixo do primeiro andar em que aparecem (`MonsterScale = 1 + 0,45·n`). A curva exponencial anterior (15%/12% por andar) criava uma parede no andar 36–45; ver `docs/balance-audit/`.
- Geração estilo Rogue: salas retangulares + corredores, escada `>`.
- O chefe fica na sala da escada `>`; só dá para descer depois de derrotá-lo. Ele deixa cair um item raro (50%), épico (`BossEpicChance`, 47%) ou lendário (`BossLegendaryChance`, 3%). O jogo é hardcore: épicos e lendários são raros (chão: 1% e 0,2% por item; a loja nunca vende lendário).
- O chefe dorme até o herói encostar nele. O herói **não sabe** se vence o chefe: não há estimativa nem recuo por chefe forte; ele luta como contra qualquer monstro (só recua pelas regras normais, como HP baixo sem poções).
- O chefe fica mais forte a cada chefe vencido: o N-ésimo chefe tem HP ×(5 + N−1) e ataque ×(1,4 + 0,1·(N−1)) do monstro mais forte do andar. Os chefes são os marcos de dificuldade; os monstros comuns crescem de forma linear (ver `docs/balance-audit/data/calibration-bosses-R5.md`).
- Matar um chefe desbloqueia atalho para o andar seguinte (começar no andar 11, 21...).
- Comportamento por andar configurável: explorar tudo (padrão) ou descer direto ao achar `>`.
- Combate roguelike: esbarra no monstro e ataca, por turnos, automático.
- Combate em ritmo fixo (1 rodada/s); a primeira rodada sai no ritmo do passo, para o combate não parecer travado ao começar. A Velocidade decide quem ataca primeiro e dá ataques extras.
- Log com uma linha por rodada; linhas iguais seguidas viram `×N`.
- Itens: equipa sozinho se for melhor; mochila com limite.

## Herói

- Um herói, com **classe escolhida no jogo novo**: Guerreiro, Mago, Ladino ou Clérigo. Save antigo vira Guerreiro. Trocar de classe fica para o prestígio.
- HP base por classe: Guerreiro 30, Clérigo 28, Mago 26, Ladino 25. O mago não pode ser frágil a ponto de morrer nos primeiros andares da primeira expedição.
- Atributos: **FOR** (dano corpo a corpo), **DES** (velocidade), **INT** (magia e mana), **CON** (vida). Cada classe tem valores iniciais e ganho por nível próprios. O treino da cidade sobe esses quatro.
- **Precisão e crítico (herói e monstros, mesma regra):** relativos ao alvo, sem teto fixo (2026-11-13, ideia do amigo do dono). Com A = DES de quem ataca + 10 e D = DES do alvo + 10: erra 40% × D / (A + D) (20% com DES igual); crítico 30% × (A − D) / (A + D), só se quem ataca for mais ágil, dano dobrado, somado ao efeito de crítico dos itens. O painel do herói e a aba Nerd mostram as chances contra a DES média dos monstros do andar atual (na cidade, do mais fundo). Esquiva do ladino: 0,5% por DES, até 15% (era 3%/35%; somada à regra relativa ficaria forte demais). A bênção corta a chance de erro pela metade. Cada monstro tem DES própria (ágeis: morcego, lobo, lobisomem) que sobe 1 a cada 2 andares abaixo do primeiro em que aparece; chefes têm +5. Magias não erram.
- **Sorte (2026-11-11, pedido de um amigo do dono):** a partir de 3 erros seguidos (`LuckStreakMin`) a linha do log mostra a chance da sequência, `(1 em 64 · 1,6%)`. Duas sequências: ataques do herói que erram (acerto ou magia zera) e ataques contra o herói que erram, somando todos os monstros (erro do monstro ou esquiva; qualquer acerto zera). A aba Nerd tem a seção Sorte com o erro e a esquiva atuais e o recorde (sequência mais improvável) de cada uma. A comparação com 5★ de gacha ficou de fora.
- **Mana:** recupera 0,25 + 0,04 × INT por passo.
- Derivados: Ataque = arma + atributo de dano da classe; Defesa = armadura + capacete + escudo + nível; Velocidade = DES + botas (explorando, a DES conta só metade); Mana só para Mago e Clérigo, recupera um pouco por passo.
- Fontes de poder: nível/XP, equipamento (achado ou comprado), melhorias permanentes na cidade.
- Equipamento: arma, armadura, botas, capacete, escudo. Armas por classe (Guerreiro e Clérigo: espadas/machados e maças; Mago cajados; Ladino adagas). Armadura leve/média/pesada (mais pesada = mais Defesa).
  - Guerreiro: qualquer armadura e escudo. Mago: leve, sem escudo. Ladino: leve/média, sem escudo. Clérigo: média/pesada e escudo.
  - O herói começa com a arma básica comum da classe (espada curta, bordão, adaga, maça).
  - A dungeon e a loja geram itens de todas as classes. O que o herói não usa vai para a mochila e é vendido na volta; a loja não deixa comprar.
- Habilidades por nível (aparecem no log da rodada):
  - **Guerreiro:** golpe forte (dano dobrado a cada N golpes), segundo fôlego (cura uma vez por andar com vida baixa), golpe giratório (o golpe forte acerta todos em volta).
  - **Ladino:** ataque furtivo (dano triplo em monstro dormindo ou intacto), esquiva (cresce com DES), faro para tesouro (mais ouro e itens).
  - **Mago:** ataca de longe quando tem mana: míssil mágico, sono, bola de fogo, piscar (foge quando cercado e ferido), raio gélido (desacelera). Sem mana, luta com o cajado.
  - **Clérigo:** curar ferimentos (antes da poção), bênção (mais acerto e Defesa no começo da luta), expulsar mortos-vivos (dano triplo em zumbi, esqueleto e Lich).

## Cidade (v1)

- Loja: poções e equipamento básico.
- Ferreiro: melhora equipamento até +10 (`UpgradeMax`). Cada nível soma 10% da base do item **como se fosse comum** e o preço usa essa base multiplicada pela raridade (×1 comum a ×5 lendário, como o valor de venda): a raridade dá base e efeitos, o ferreiro dá a progressão do andar (o dono achou o upgrade forte demais em bases altas; sem limite, fundo o ouro cresce mais rápido que o preço).
- Mochila: começa com 8 vagas; cada upgrade dá +2, até 24. Fica na loja.
- Moeda única: ouro.

## Técnico

- Números de balanceamento em tabelas Go num lugar só (`balance.go`). Ritmo definido jogando.
- RNG com seed por expedição (testes e reprodução de bugs).
- Testes automatizados na lógica (geração, combate, simulação); sem testar a UI.
- Bip do terminal (morte/chefe) desligado por padrão, com opção para ligar.
- **Efeito CRT** desligado por padrão (opção na tela Config): camada estética de terminal antigo (Fallout), sem virar monocromático. As cores originais são mantidas, um pouco desbotadas e com tom verde-fósforo (o texto na cor padrão do terminal também ganha o tom); linhas alternadas ficam um pouco mais escuras (scanlines); de vez em quando os elementos cinzas (texto secundário, paredes, bordas) oscilam de brilho; raramente uma borda da interface aparece "rasgada" (`│` → `╎`) por um instante. Nada muda texto nem layout.
- **Aba Nerd** desligada por padrão (opção "Coisas de nerd" na Config): lista só de leitura com todos os números de balanceamento, agrupados por tema, e a descrição embaixo explicando o impacto de cada um. É a única fonte das explicações; `balance.go` não tem comentários por constante.
- **Idioma:** inglês por padrão, português como opção na tela Config. Fica no save. Ao trocar, nomes de itens mudam junto; linhas antigas do log ficam no idioma em que foram escritas.

## Marcos

1. `@` explorando um andar gerado, névoa, câmera, layout adaptável.
2. Monstros + combate + log + morte.
3. Andares, escada, loot, mochila, retirada automática.
4. Cidade (loja, ferreiro), auto-expedição, save.
5. Offline, chefes, atalhos, legenda.

## Táticas (gambits)

- **O que são:** a IA do herói é uma lista de regras na Preparação (até 16), sem progressão: tudo liberado desde o início. Cada regra é **quando** (até 2 condições, as duas precisam valer) → **fazer** (ação + alvo) → **até** (saída). Toda condição com número escolhe o operador: `<`, `>`, `≤` ou `≥` (HP%, mana%, inimigos à vista, inimigos ao lado, poções, andar, terminou o andar, melhor item na mochila). "Melhor item na mochila" compara a maior raridade da mochila da expedição (mochila vazia fica abaixo de Comum).
- **Ações** duram 1 passo: poção, segundo fôlego, cura, bênção, ataque, magias separadas e "Magias (auto)" (a escolha de magia de sempre). **Comportamentos** continuam até a saída: Fugir, Caçar e Recuar. Fugir usa um mapa de segurança (distância a pé dos monstros acordados à vista, invertida e relaxada, até 25 casas): o herói procura portas, corredores e outras salas, contornando o monstro se preciso, em vez de ir para o canto mais longe em linha reta (2026-11-11; antes ficava encurralado na sala).
- **Táticas não mudam as regras básicas do jogo:** cuidam de combate e do recuo, não de como o herói anda pela dungeon. Andar segue o **modo da expedição** (explorar tudo ou descer direto). Por isso não existem Explorar, Descer (brigavam com o modo) nem Descansar (com "sempre" travava a expedição). "Terminou o andar ≥ N" só existe com Recuar (com outra ação travava o herói na `>`). "Um golpe pode me matar" saiu: o herói não age sabendo que morreria.
- **Prioridade:** a cada passo a lista é lida de cima para baixo. Uma regra mais alta cuja entrada vale assume no lugar do comportamento ativo; uma mais baixa espera ele sair. Só as ações acima do comportamento ativo rodam (por isso, recuando, ele só se cura). Regra que não pode agir agora (sem mana, sem alvo, habilidade não aprendida) é pulada.
- **Saídas:** enquanto valer (padrão) · até uma condição · por N segundos · até terminar (nenhum monstro à vista). Depois de sair por tempo ou condição, a regra fica desarmada até a entrada deixar de valer, para não religar no passo seguinte. Recuar só termina na cidade.
- **Só o que a classe usa:** a lista padrão e o editor mostram só ações e condições da classe do herói (sem magias para o guerreiro, sem segundo fôlego para o mago, sem condições de mana para quem não tem mana). Regras de outra classe nunca agiam, então o jogo não muda.
- **Lista padrão:** segundo fôlego (HP < 30%), cura e poção (HP < 40%), recuar com mochila cheia, recuar sem poções com HP < 30%, magias (auto), bênção e ataque com inimigo ao lado. É o herói antigo sem a poção antes de golpe letal (o dono tirou essa regra); com ela, a lista foi conferida idêntica ao herói antigo em 56 simulações. "Restaurar táticas padrão" volta para ela.
- **Regras por nível:** a lista padrão só traz as habilidades já liberadas. Ao liberar uma habilidade ativa no nível, a regra padrão dela entra onde estaria na lista padrão, com uma linha no log; se já existe regra com a mesma ação, nada entra. Só acontece na hora de liberar: apagar a regra depois não a traz de volta. O mago continua com magias (auto), que já usa cada magia ao liberar.
- **Save v6:** as opções antigas (beber poção com HP <, recuar sem poção, recuar com mochila cheia, último andar) viram as regras padrão com os mesmos números.
- **Save v8:** o bônus do ferreiro fica guardado no item (`Item.Forged`); o passo converte os upgrades antigos com a regra antiga (10% da base por nível), então nada encolhe.
- **Save v7:** apaga as regras Explorar, Descer e Descansar, as de "terminou o andar" sem Recuar e as com "um golpe pode me matar". O herói padrão foi conferido idêntico em 56 simulações.
- **Sem regra que se aplique,** o herói anda pelo modo da expedição (nunca fica travado). O `r` do jogador continua acima de tudo.
- O herói **não recupera HP** na dungeon, só mana: por isso existe a saída por tempo.

## Playground

- **O que é:** programa separado (`go run ./cmd/playground`), **público** (decisão do dono, 2026-11-11): vai nas releases como `clidle-playground-linux-amd64` e `clidle-playground-windows-amd64.exe` e no Pages em `/playground/` (fora do cache offline do jogo, para quem só joga não baixar mais 13 MB), para testar mecânicas. Monta andares à mão, dá play e mostra o herói agindo com **as regras do jogo** (mesmo `step`, IA, táticas, magias e combate). Não lê nem grava o save. Abre em inglês; a aba Config troca o idioma (não fica salvo).
- **Layout só para desktop** (mínimo 136×30): o mapa e o log ficam **sempre visíveis** à direita; a coluna da esquerda muda com a aba (na aba Mapa, o painel do herói). Na aba Andar, a direita mostra o andar que está sendo editado, inteiro.
- **Teclas** em laranja no rodapé e nas abas: espaço play/pausa, `p` passo, `+/-` velocidade (0,25× a 50×), `r` reiniciar, `n` nova seed, `v` ver tudo, `1-6` telas, `q` sair. Nas listas: setas, enter, esc, `x` remove grupo de monstros, pgup/pgdn ±10.
- **Telas:** 1 Mapa, 2 Herói (classe, nível, poções, atributos extras como o treino), 3 Equipamento (item por vaga entre os que a classe usa, raridade, andar do item, ferreiro, efeitos na faixa do jogo e na quantidade da raridade), 4 Andar, 5 Táticas (o mesmo editor do jogo), 6 Config (idioma, efeitos do mapa).
- **Andares:** a aba Andar escolhe qual andar editar. Andar novo começa **sem salas**, e **andar sem salas não existe** para o jogo. A simulação começa no primeiro andar que existe; ao chegar na `>`, desce se o próximo andar existe, senão termina. Grade 3×3: setas andam pelas salas e enter **abre a sala** (a grade fica em cima, com a sala destacada); esc ou "Voltar à grade" voltam. Dentro da sala: Ligada, Entrada, Saída e monstros; a primeira sala ligada ganha `<` e `>`; desligar a sala da escada passa a escada para outra sala; as salas ligadas precisam se tocar. Cada sala: entrada, saída e grupos de monstros (tipo, quantidade, força em andares).
- **Seed é número, como no jogo** (a seed da expedição, um inteiro de 64 bits): enter na linha Seed e digita só dígitos; `n` sorteia pelo relógio, como um jogo novo. A luta usa a seed numa fila só, como a expedição. Exceção combinada com o dono: o desenho de cada andar usa a seed e o número do andar, para o andar 2 em diante ficar exatamente como foi montado, não importa o que aconteceu antes.
- **Mesmas regras, sem editar balanceamento:** item é o drop médio do andar escolhido; monstro é o do andar escolhido (nunca abaixo do primeiro andar dele); chefe só em andares múltiplos de 10, feito como no jogo (o mais forte de 20 monstros, por isso muda com a seed). Os corredores vêm do gerador do jogo, só entre as salas ligadas.
- **Abre cru:** andar 1 com só a sala do meio, `<` e `>` nela, sem monstros, guerreiro nível 1 com a arma inicial (pedido do dono: nada de exemplo pronto).
- **Sem loot no chão.** Chefe morto ainda dropa item.
- **Qualquer mudança reinicia a simulação** (pausada). Mesma seed = mesma luta.
- **Fim:** o herói morre, sai pela `<` (recuo) ou chega na `>` do último andar que existe. O log mostra tempo, passos, monstros mortos, dano dado/recebido e poções.
- "Ver tudo" só muda o desenho: o herói continua sem saber o que não viu.

## Para depois

- Grupo (vários `@` com cores diferentes).
- Prestígio (reset com bônus permanente).
- Itens lendários (únicos).
- Cavernas e outras variações de andar.
- Magias por loot (livros/pergaminhos).
- Materiais, outros prédios.
- ~~Regras de retirada mais elaboradas~~ (feito com as táticas).
