# Classes — Design

Data: 2026-10-28. Leia junto com `AGENTS.md` e `docs/design.md`.

## Por quê

Hoje o herói é um guerreiro implícito: corpo a corpo, Ataque/Defesa/Velocidade. Antes de balancear poções, raridades e equipamentos, o dono quer classes, porque elas mudam o que cada um desses números significa (mago não usa escudo, clérigo depende menos de poção etc.).

No idle, o jogador não controla o combate. Então **a classe precisa aparecer no que o herói faz sozinho**: como ele luta, o que usa, o que aparece no log.

## Decisões do dono

- **Um herói só**, com uma classe. Grupo de heróis fica para depois.
- **4 classes:** Guerreiro, Mago, Ladino, Clérigo.
- **Classe escolhida no início do jogo.** Trocar de classe vem depois, junto com o prestígio. O save atual vira Guerreiro.
- **Atributos estilo D&D simplificado:** Força (FOR), Destreza (DES), Inteligência (INT), Constituição (CON). Sem carisma/sabedoria.
- **Mago ataca à distância e usa magias** parecidas com as de roguelike clássico.
- **Magias por nível** (cada uma libera num nível fixo). Magias vindas de loot ficam para depois.
- **Capacete e escudo** entram como tipos novos de equipamento.

## Objetivos

- Cada classe joga de um jeito visivelmente diferente no mapa e no log.
- Guerreiro continua sendo a classe simples e segura (é o herói de hoje).
- Saves existentes continuam abrindo.

## Não-objetivos (agora)

- Grupo de heróis, troca de classe, prestígio.
- Magias por loot (livros/pergaminhos), poção de mana.
- Monstros que atacam à distância; herói que recua para manter distância ("kiting").
- Equipar/desequipar manualmente, baú na cidade (etapa própria, depois).
- Ajustes de poção, item garantido por andar, raridades novas (etapa de balanceamento, depois desta).

## Anti-objetivos

- Classes que só diferem em números.
- Quebrar as invariantes do `AGENTS.md`: determinismo, `Advance` independente do fatiamento, offline = mesma lógica, save/load idêntico, desempenho de 12h offline.

## Atributos

| Atributo | Efeito |
|---|---|
| **FOR** | dano corpo a corpo |
| **DES** | velocidade (exploração e iniciativa), acerto/crítico |
| **INT** | dano das magias, mana máxima (mago e clérigo) |
| **CON** | vida máxima |

Valores derivados (fórmulas em `balance.go`):

- **Ataque** = dano da arma + atributo de dano da classe (FOR para guerreiro/clérigo, DES para ladino, INT para magias do mago).
- **Defesa** = soma de armadura + capacete + escudo, mais um pouco por nível como hoje (não vem de atributo).
- **Velocidade** = DES + botas.
- **Mana** = INT (só mago e clérigo). Recupera um pouco a cada passo.
- **HP** = CON + nível.

**Decisão minha:** "mesmo comportamento" no marco 1 quer dizer **jogar parecido**, não números idênticos. O Guerreiro deve ter uma curva de HP, Ataque e Defesa próxima da de hoje (o nível continua contando), para o save atual não ficar mais fraco de repente. O ajuste fino vem na etapa de balanceamento.

Cada classe tem atributos iniciais e ganho por nível próprios, com o atributo principal crescendo mais. **Treino da cidade** passa a ser FOR / DES / INT / CON.

## Classes

Lista inicial de habilidades. **Decisão minha** (o dono aprovou as classes e "magias por nível", não esta lista); números ficam para o construtor e para o balanceamento jogando.

### Guerreiro (FOR)
- Corpo a corpo. Mais vida. Usa qualquer armadura, escudo.
- Nv 1 **Golpe forte:** de tempos em tempos, um ataque com dano dobrado.
- Nv 5 **Segundo fôlego:** uma vez por andar, recupera parte da vida quando ela fica baixa.
- Nv 10 **Golpe giratório:** acerta todos os monstros adjacentes.

### Mago (INT)
- Ataca à distância: se há monstro visível dentro do alcance e mana suficiente, conjura em vez de andar até ele. Sem mana, luta corpo a corpo com o cajado (fraco).
- Pouca vida. Só armadura leve, sem escudo.
- Nv 1 **Míssil mágico:** dano à distância, alvo único.
- Nv 4 **Sono:** o alvo dorme algumas rodadas (chefe resiste ou dorme menos).
- Nv 7 **Bola de fogo:** dano no alvo e em quem estiver em volta dele.
- Nv 10 **Piscar:** quando cercado e com vida baixa, teletransporta para um ponto visível longe.
- Nv 14 **Raio gélido:** dano e deixa o alvo mais lento.

### Ladino (DES)
- Corpo a corpo rápido. Armadura leve ou média, sem escudo.
- Nv 1 **Ataque furtivo:** crítico grande contra monstro dormindo ou no primeiro golpe.
- Nv 5 **Esquiva:** chance de evitar um golpe, crescendo com DES.
- Nv 8 **Faro para tesouro:** mais ouro nas pilhas e mais chance de item.

### Clérigo (FOR, mana por INT)
- Corpo a corpo médio. Armadura média ou pesada, escudo.
- Nv 1 **Curar ferimentos:** usa mana para se curar; tenta isso antes de beber poção.
- Nv 4 **Bênção:** por algumas rodadas, mais acerto e Defesa.
- Nv 8 **Expulsar mortos-vivos:** dano alto em zumbi, esqueleto e Lich.

## Equipamento

- Espaços: **arma, armadura, capacete, escudo, botas**.
- **Arma depende da classe** (decisão minha, nomes por profundidade como hoje):
  - Guerreiro: espadas e machados.
  - Mago: cajados.
  - Ladino: adagas.
  - Clérigo: maças.
- **Armaduras têm peso** (leve / média / pesada) e cada classe só usa algumas (ver classes).
- Capacete: todas as classes. Escudo: guerreiro e clérigo.
- **Decisão do dono:** a dungeon e a loja geram itens de **todas** as classes. Itens que a classe não usa vão para a mochila e, por enquanto, são vendidos na volta. Depois, um **baú (stash) na cidade** vai guardá-los para outros heróis do grupo.
- A loja pode mostrar itens que o herói não usa; a compra deles fica bloqueada até existir o baú (decisão minha).
- Auto-equipar continua como hoje nesta etapa.

## Escolha da classe

- **Jogo novo:** antes da cidade, uma tela simples para escolher a classe, com uma linha descrevendo cada uma.
- **Save antigo:** sem classe salva → Guerreiro (valor zero do campo). O treino antigo é convertido: Vigor → CON, Força → FOR, Agilidade → DES. Resistência deixa de existir (Defesa agora é só equipamento); **decisão minha:** os níveis dela viram CON.
- **Decisão minha:** a conversão é a primeira migração de verdade: `saveVersion` sobe para 2 e um passo de migração roda uma vez ao carregar um save versão 1. Valor zero não basta, porque Guerreiro é a classe zero e o treino é salvo por índice. Atualizar a regra de compatibilidade no `AGENTS.md`.
- Itens antigos (Adaga, Maça...) continuam existindo; se a classe não puder usar, o herói mantém o que já está equipado até trocar.

## Log e texto

- Habilidades e magias aparecem no log da mesma rodada, como os ataques hoje (`você lança míssil mágico no rato (7)`).
- Todo texto novo vai em `internal/lang/en.json` e `pt.json`.
- Painel do herói mostra classe, FOR/DES/INT/CON, Mana (quando houver) e magias liberadas.

## Restrições técnicas

- Toda aleatoriedade via `Expedition.RNG` / `Game.RNG`.
- Alcance das magias usa a visibilidade que já existe (`Level.Visible`); nada de BFS extra por passo.
- Campos novos de save precisam funcionar com valor zero (sem migração).
- Números só em `balance.go`.

## Marcos sugeridos

Parar ao fim de cada um para o dono testar.

1. **Atributos + classe:** FOR/DES/INT/CON, valores derivados, campo de classe, tela de escolha, save antigo vira Guerreiro com o mesmo comportamento de hoje.
2. **Equipamento:** capacete, escudo, peso de armadura, armas por classe, geração filtrada pela classe.
3. **Guerreiro e Ladino:** habilidades.
4. **Mana, Mago e Clérigo:** mana, ataque à distância, magias.

## Fica para o construtor

- Todas as fórmulas e números (atributos iniciais, ganho por nível, custos, alcance, mana, duração de efeitos).
- Nomes dos itens novos (cajados, capacetes, escudos, armaduras leves/médias).
- Como a IA decide entre magias quando várias estão disponíveis.
