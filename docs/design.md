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
- Telas: `1` Principal · `2` Preparação · `3` Herói · `?` legenda/ajuda · `q` sair.
- **Principal** muda sozinha: mostra a dungeon durante a expedição e a cidade quando o herói volta.
- **Preparação** fica sempre disponível: configura a próxima expedição e gasta o ouro guardado. Mudanças valem a partir da próxima expedição; compras de equipamento entram quando o herói voltar.

## Visual

- Mapa ASCII estilo Rogue; o jogador vê o `@` explorando sozinho.
- Andar de tamanho fixo (~80×21); a câmera segue o `@` em terminais menores.
- Névoa de guerra.
- Legenda explicando cada caractere de entidade.
- Monstros por letra (`r`, `g`, `o`...); chefes em letra maiúscula com cor própria. ~8–10 tipos na v1.
- Raridade de itens por cor: Comum (branco), Raro (azul), Épico (roxo).

## Expedições

- O herói entra na dungeon, sai quando quiser e guarda o loot.
- **Morte:** perde ouro e itens achados na expedição; mantém nível/XP, poções e o equipamento que saiu da cidade. Volta para a cidade.
- **Volta:** o herói sai pela escada `<` do andar atual; o ouro é guardado e a mochila é vendida automaticamente.
- **Retirada automática:** 2–3 regras simples configuráveis (ex: HP < X% sem poções, mochila cheia, chegou ao andar X).
- **Início:** o jogo começa na cidade; a expedição parte com `e` ou "Partir agora" na Preparação.
- **Auto-expedição:** liga/desliga (desligada por padrão). Ao terminar, inicia outra com a mesma configuração.
- Mandar o herói voltar manualmente desliga a auto-expedição; a retirada automática não.
- Antes de partir, o herói compra poções até completar o número configurado (se tiver ouro).
- **Poções:** bebidas automaticamente abaixo de um limite de HP configurável.

## Dungeon

- Andares infinitos, cada vez mais fundos. Chefe a cada 10 andares.
- Geração estilo Rogue: salas retangulares + corredores, escada `>`.
- O chefe fica na sala da escada `>`; só dá para descer depois de derrotá-lo. Ele deixa cair um item épico.
- Matar um chefe desbloqueia atalho para o andar seguinte (começar no andar 11, 21...).
- Comportamento por andar configurável: explorar tudo (padrão) ou descer direto ao achar `>`.
- Combate roguelike: esbarra no monstro e ataca, por turnos, automático.
- Combate em ritmo fixo (1 rodada/s); a Velocidade decide quem ataca primeiro e dá ataques extras.
- Log com uma linha por rodada; linhas iguais seguidas viram `×N`.
- Itens: equipa sozinho se for melhor; mochila com limite.

## Herói

- Um herói na v1.
- Atributos: HP, Ataque, Defesa, Velocidade (passos por segundo + iniciativa).
- Velocidade de exploração vem de atributo/equipamento.
- Fontes de poder: nível/XP, equipamento (achado ou comprado), melhorias permanentes na cidade.

## Cidade (v1)

- Loja: poções e equipamento básico.
- Ferreiro: melhora equipamento.
- Moeda única: ouro.

## Técnico

- Números de balanceamento em tabelas Go num lugar só (`balance.go`). Ritmo definido jogando.
- RNG com seed por expedição (testes e reprodução de bugs).
- Testes automatizados na lógica (geração, combate, simulação); sem testar a UI.
- Bip do terminal (morte/chefe) desligado por padrão, com opção para ligar.

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
- Mais atributos (crítico, sorte...), materiais, outros prédios.
- Herói autônomo o bastante para explorar sem preocupação (regras de retirada mais elaboradas).
