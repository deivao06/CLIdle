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
- Telas: `1` Principal · `2` Preparação · `3` Herói · `4` Config · `?` legenda/ajuda · `q` sair.
- **Principal** muda sozinha: mostra a dungeon durante a expedição e a cidade quando o herói volta.
- **Preparação** fica sempre disponível: configura a próxima expedição e gasta o ouro guardado. Mudanças valem a partir da próxima expedição; compras de equipamento entram quando o herói voltar.
- **Config** guarda as opções do jogo (bip, idioma) e **Resetar save**, que pede confirmação, apaga o save e volta para a escolha de classe.

## Visual

- Mapa ASCII estilo Rogue; o jogador vê o `@` explorando sozinho.
- Andar de tamanho fixo (~80×21); a câmera segue o `@` em terminais menores.
- Névoa de guerra.
- Legenda explicando cada caractere de entidade.
- Monstros por letra (`r`, `g`, `o`...); chefes em letra maiúscula com cor própria. ~8–10 tipos na v1.
- Raridade de itens por cor: Comum (branco), Raro (azul), Mágico (verde), Épico (roxo), Lendário (laranja).
- **Efeitos:** épico tem 1 efeito, lendário 2 (diferentes): roubo de vida, crítico (dano dobrado), HP extra, ouro extra, cura de poção extra, XP extra. Valores em `EffectRange`.

## Expedições

- O herói entra na dungeon, sai quando quiser e guarda o loot.
- **Morte:** perde metade do ouro e os itens achados na expedição; mantém nível/XP, poções e o equipamento que saiu da cidade. Volta para a cidade.
- **Volta:** o herói sai pela escada `<` do andar atual; o ouro é guardado e a mochila é vendida automaticamente.
- **Retirada automática:** 2–3 regras simples configuráveis (ex: HP < X% sem poções, mochila cheia, chegou ao andar X).
- **Início:** o jogo começa na cidade; a expedição parte com `e` ou "Partir agora" na Preparação.
- **Auto-expedição:** liga/desliga (desligada por padrão). Ao terminar, inicia outra com a mesma configuração.
- Mandar o herói voltar manualmente desliga a auto-expedição; a retirada automática não.
- **Auto-equipar:** ligado por padrão. Desligado, o herói não troca equipamento na dungeon; na volta, a mochila vai para o **baú** (20 vagas, o que sobra é vendido). Na Preparação o jogador equipa (enter), desequipa e vende (`x`) itens do baú, só com o herói na cidade.
- **Filtro de loot:** "Pegar itens" (comum+ por padrão) define a raridade mínima que vai para a mochila. Item que o herói vai equipar é pego sempre.
- **Limite de mortes:** configurável (padrão sem limite). Depois de N mortes desde que a auto-expedição foi ligada, ela desliga e o herói fica na cidade. Religar zera a contagem.
- Antes de partir, o herói compra poções até completar o número configurado (se tiver ouro).
- **Poções:** bebidas automaticamente abaixo de um limite de HP configurável. O herói guarda no máximo 10; o que passar disso é vendido ao chegar na cidade (metade do preço).

## Dungeon

- Andares infinitos, cada vez mais fundos. Chefe a cada 10 andares.
- Geração estilo Rogue: salas retangulares + corredores, escada `>`.
- O chefe fica na sala da escada `>`; só dá para descer depois de derrotá-lo. Ele deixa cair um item épico.
- O chefe dorme até o herói chegar a 3 casas. Ao ver o chefe, o herói estima a luta (dano médio dos dois, HP e poções); se não vence, recua ("chefe forte demais").
- Matar um chefe desbloqueia atalho para o andar seguinte (começar no andar 11, 21...).
- Comportamento por andar configurável: explorar tudo (padrão) ou descer direto ao achar `>`.
- Combate roguelike: esbarra no monstro e ataca, por turnos, automático.
- Combate em ritmo fixo (1 rodada/s); a Velocidade decide quem ataca primeiro e dá ataques extras.
- Log com uma linha por rodada; linhas iguais seguidas viram `×N`.
- Itens: equipa sozinho se for melhor; mochila com limite.

## Herói

- Um herói, com **classe escolhida no jogo novo**: Guerreiro, Mago, Ladino ou Clérigo. Save antigo vira Guerreiro. Trocar de classe fica para o prestígio.
- Atributos: **FOR** (dano corpo a corpo), **DES** (velocidade), **INT** (magia e mana), **CON** (vida). Cada classe tem valores iniciais e ganho por nível próprios. O treino da cidade sobe esses quatro.
- Derivados: Ataque = arma + atributo de dano da classe; Defesa = armadura + capacete + escudo + nível; Velocidade = DES + botas (explorando, a DES conta só metade); Mana só para Mago e Clérigo, recupera um pouco por passo.
- Fontes de poder: nível/XP, equipamento (achado ou comprado), melhorias permanentes na cidade.
- Equipamento: arma, armadura, botas, capacete, escudo. Armas por classe (Guerreiro espadas/machados, Mago cajados, Ladino adagas, Clérigo maças). Armadura leve/média/pesada (mais pesada = mais Defesa).
  - Guerreiro: qualquer armadura e escudo. Mago: leve, sem escudo. Ladino: leve/média, sem escudo. Clérigo: média/pesada e escudo.
  - A dungeon e a loja geram itens de todas as classes. O que o herói não usa vai para a mochila e é vendido na volta; a loja não deixa comprar.
- Habilidades por nível (aparecem no log da rodada):
  - **Guerreiro:** golpe forte (dano dobrado a cada N golpes), segundo fôlego (cura uma vez por andar com vida baixa), golpe giratório (o golpe forte acerta todos em volta).
  - **Ladino:** ataque furtivo (dano triplo em monstro dormindo ou intacto), esquiva (cresce com DES), faro para tesouro (mais ouro e itens).
  - **Mago:** ataca de longe quando tem mana: míssil mágico, sono, bola de fogo, piscar (foge quando cercado e ferido), raio gélido (desacelera). Sem mana, luta com o cajado.
  - **Clérigo:** curar ferimentos (antes da poção), bênção (mais acerto e Defesa no começo da luta), expulsar mortos-vivos (dano triplo em zumbi, esqueleto e Lich).

## Cidade (v1)

- Loja: poções e equipamento básico.
- Ferreiro: melhora equipamento.
- Mochila: começa com 8 vagas; cada upgrade dá +2, até 24. Com o herói fora, vai para a fila e vale na volta, como o ferreiro.
- Moeda única: ouro.

## Técnico

- Números de balanceamento em tabelas Go num lugar só (`balance.go`). Ritmo definido jogando.
- RNG com seed por expedição (testes e reprodução de bugs).
- Testes automatizados na lógica (geração, combate, simulação); sem testar a UI.
- Bip do terminal (morte/chefe) desligado por padrão, com opção para ligar.
- **Idioma:** inglês por padrão, português como opção na tela Config. Fica no save. Ao trocar, nomes de itens mudam junto; linhas antigas do log ficam no idioma em que foram escritas.

## Marcos

1. `@` explorando um andar gerado, névoa, câmera, layout adaptável.
2. Monstros + combate + log + morte.
3. Andares, escada, loot, mochila, retirada automática.
4. Cidade (loja, ferreiro), auto-expedição, save.
5. Offline, chefes, atalhos, legenda.

## Para depois

- Grupo (vários `@` com cores diferentes).
- Prestígio (reset com bônus permanente).
- Itens lendários (únicos).
- Cavernas e outras variações de andar.
- Magias por loot (livros/pergaminhos).
- Materiais, outros prédios.
- Herói autônomo o bastante para explorar sem preocupação (regras de retirada mais elaboradas).
