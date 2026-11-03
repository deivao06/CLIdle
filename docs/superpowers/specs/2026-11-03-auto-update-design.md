# Releases e auto-update — design

## Intenção

Hoje o CLIdle só é distribuído compilando o código. O dono quer **releases** no GitHub e que o jogo **descubra sozinho** quando há versão nova, **avise** e, com a confirmação do jogador, **se atualize e reinicie**.

## Objetivos

1. **Releases** no GitHub (`deivao06/CLIdle`, repositório público) com binários prontos para **Linux** e **Windows** e uma página com **o que mudou**.
2. O jogo **checa** se há versão nova **quando o herói volta à cidade**.
3. Encontrando, **pergunta** ao jogador; confirmando, **baixa, troca o binário, salva e reinicia**.
4. A **versão** da build aparece sempre no **canto inferior direito da tela 4** (Config).

## Não-objetivos

- macOS e outras arquiteturas além de amd64.
- Atualizar sem confirmação.
- Opção na Config para desligar a checagem.
- Rollback para versões antigas.

## Anti-objetivos

- Tocar na lógica do jogo, no determinismo ou no progresso offline (`internal/game` não muda).
- Travar a UI esperando a rede.
- Corromper o binário ou o save se a rede cair no meio, ou se o jogador recusar.
- Estourar o limite da API do GitHub sem login (60 consultas/hora por IP).

## Como uma release nasce

- O dono cria e envia uma tag semver: `git tag v0.1.0 && git push --tags`. A primeira versão é **v0.1.0**.
- Um workflow do GitHub Actions (`.github/workflows/release.yml`), disparado por tags `v*`:
  1. compila, com `CGO_ENABLED=0`, `-ldflags "-s -w -X <pacote>.Version=<tag>"`:
     - `clidle-linux-amd64`
     - `clidle-windows-amd64.exe`
  2. gera `SHA256SUMS` dos dois arquivos;
  3. monta as **notas** a partir dos commits entre a tag anterior e a nova (`git log`), agrupando pelo prefixo:
     - `feat:` → **Novidades**
     - `fix:` → **Correções**
     - os demais prefixos ficam de fora;
     - sem tag anterior, usa todos os commits;
  4. cria a release com os 3 arquivos e as notas (`gh release create`).
- Versões são comparadas como semver `vMAJOR.MINOR.PATCH`.

## Versão no jogo

- Uma variável `Version`, padrão `"dev"`, preenchida pelo `-ldflags` no workflow.
- Tela 4: a versão em cinza apagado no **canto inferior direito** do corpo da tela, ex.: `v0.1.0` ou `dev`. *(O canto esquerdo já tem as teclas no rodapé.)*
- Build `dev` (compilada localmente) **nunca checa** atualização.

## Checagem

- Gatilho: o herói **chega na cidade** (volta ou morte) durante o jogo ao vivo, e também ao abrir o jogo com o herói já na cidade.
- **No máximo 1 consulta por hora** (controle em memória). Nunca durante o progresso offline (`CatchUp`).
- Depois que encontra uma versão nova, **para de consultar**: o aviso fica até o jogador agir.
- A consulta roda **em segundo plano** (comando do Bubble Tea), com timeout curto (~10 s). Falha de rede, limite da API ou resposta estranha: **silêncio**, tenta de novo na próxima janela.
- Fonte: `GET https://api.github.com/repos/deivao06/CLIdle/releases/latest`. Usa a tag, as notas e as URLs dos assets do sistema atual (`runtime.GOOS`/`GOARCH`) e do `SHA256SUMS`. Sem asset para o sistema atual: não oferece.

## Aviso e confirmação

- Achando versão nova, aparece um **aviso fixo no cabeçalho**: "Nova versão vX.Y.Z · u atualizar". Fica visível em todas as telas e durante as expedições, até o jogador atualizar.
- **`u`** abre a confirmação a qualquer momento (inclusive no meio da expedição: o save preserva a simulação).
- Com a auto-expedição **desligada**, a confirmação também **abre sozinha** ao chegar na cidade.
- A confirmação usa a caixa que já existe, mas **não fecha** quando o herói muda de lugar.
- Texto: "Nova versão vX.Y.Z disponível (atual vA.B.C). Atualizar agora?" e, abaixo, até ~5 itens das notas da release. *(Mostrar as notas é decisão minha.)*
- **Não** (n/esc): fecha; o aviso continua.
- **Sim**: mostra "Atualizando..." e, em segundo plano:
  1. consulta a release mais recente **uma vez** (pode ter saído outra desde o aviso) e usa ela;
  2. baixa o binário para um arquivo temporário **no mesmo diretório** do executável atual;
  3. confere o SHA-256 contra o `SHA256SUMS` *(decisão minha)*;
  4. troca o executável:
     - **Linux:** `rename` atômico por cima do atual (permissão 0755);
     - **Windows:** renomeia o atual para `.old` e o novo para o nome original;
  5. salva o jogo e encerra o Bubble Tea, restaurando o terminal;
  6. reinicia: **Linux** com `exec` no mesmo processo; **Windows** iniciando o novo executável no mesmo terminal e **esperando ele terminar** (repassando o código de saída). O `.old` fica travado enquanto isso; o jogo tenta apagá-lo a cada início, em silêncio. *(Decisão minha.)*
- Erro em 1–4 (rede, checksum, sem permissão de escrita): apaga o temporário, não troca nada e mostra "Falha ao atualizar: ..." na **barra inferior** por alguns segundos (como o erro de save; não vai para o log do jogo, que é salvo). *(Decisão minha.)* O aviso continua.

## README

- O aviso de IA continua no topo.
- Nova seção **Instalação**: baixar o binário da página de Releases (Linux: `chmod +x`).
- Seção curta **Releases**: como criar uma (tag `vX.Y.Z` + push) e que o jogo avisa sozinho.

## Fica a critério do builder

- Nome e lugar do pacote (ex.: `internal/update`) e da variável `Version`.
- Como detectar a "chegada na cidade" na UI (ex.: comparar `InTown()` entre ticks).
- Formato exato das notas e do script do workflow.
- Testes: comparação de versões, escolha do asset, montagem das notas, troca do binário com arquivos temporários, e a consulta usando um servidor HTTP de teste (URL base configurável).

## Pronto quando

- Push de uma tag gera uma release com os 2 binários, `SHA256SUMS` e notas agrupadas.
- Um binário da release mostra a versão na tela 4; uma build local mostra `dev` e não consulta a rede.
- Com uma release mais nova publicada, ao voltar à cidade o jogo pergunta; "sim" atualiza e reinicia já na versão nova com o save intacto; "não" segue o jogo.
- `CGO_ENABLED=0 go build` continua funcionando; `gofmt -l .` vazio, `go vet ./...` e `go test ./...` passando.
