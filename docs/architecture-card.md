# Card de arquitetura

[`site/architecture-v070.png`](../site/architecture-v070.png) resume o projeto
em uma imagem: a pipeline, os perfis, o núcleo, os testes, o CI, a cadeia de
suprimento e a distribuição.

Ele é gerado a partir do texto abaixo, versionado aqui pelo mesmo motivo que o
HTML fica ao lado do gráfico em [`docs/benchmark`](benchmark/README.md): o card
afirma fatos sobre o repositório, e fato em imagem não é verificável por
ninguém depois. Guardando o prompt, ele pode ser refeito.

## O que torna este card falso

O card codifica dados que mudam. Quando qualquer um destes mudar, ele passa a
mentir e precisa ser refeito:

| O card afirma | Onde está a verdade |
| --- | --- |
| versão `v0.7.0` | a tag mais recente |
| seis perfis e seus codecs | `targets` em [`internal/profile/registry.go`](../internal/profile/registry.go) |
| quatro combinações de CI | a matriz em [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) |
| seis alvos de release, deb/rpm/apk | `builds` e `nfpms` em [`.goreleaser.yaml`](../.goreleaser.yaml) |
| Go 1.26+ | a diretiva `go` em [`go.mod`](../go.mod) |
| canais de distribuição | a seção de instalação dos READMEs |

Antes de regerar, confira cada linha. A versão anterior deste card mostrava
três perfis quando já existiam seis, e colocava `amd64`/`arm64` na faixa de CI,
onde arquitetura não é dimensão de teste.

Renomeie o arquivo ao mudar a versão (`architecture-v080.png`), para que uma
referência antiga nunca aponte silenciosamente para um conteúdo diferente.

## Idioma

Este card está em português e por isso aparece apenas no
[`README.pt-BR.md`](../README.pt-BR.md). O [`README.md`](../README.md) e a
landing page em [`site/index.html`](../site/index.html) estão em inglês e
precisam de uma versão equivalente, que ainda não existe. Para gerá-la, traduza
os rótulos do prompt abaixo e mantenha os nomes próprios de tecnologia.

## Prompt

```text
Card 1600x1000. Fundo #0b1220, tema escuro, tipografia monoespaçada,
cor de acento teal (#5eead4). TUDO EM PORTUGUÊS, exceto nomes próprios de
tecnologia (FFmpeg, Go, Cobra, GoReleaser, Sigstore, cosign, SBOM, CodeQL,
golangci-lint, govulncheck, GitHub Actions, Homebrew, Scoop).

Cabeçalho: logo (seta de play estilizada) + "MediaConv", e abaixo
"Conversor de mídia por linha de comando, sobre FFmpeg".
Canto inferior direito do card: "v0.7.0".

FAIXA 1 — PIPELINE. É a maior faixa, ocupa cerca de 30% da altura e é o
destaque visual do card. Nove caixas ligadas por setas, da esquerda para a
direita:
CLI → Validação → ffprobe → Perfil → Plano → FFmpeg → ffprobe → Verificação
→ Publicação
A caixa "Verificação" tem borda mais grossa e brilho suave.
Legenda abaixo da faixa, em corpo bem maior que o das caixas:
"o destino final só recebe o arquivo depois de verificado"

FAIXA 2 — PERFIS. Seis caixas, em duas linhas de três:
  web      MP4  · H.264 + AAC
  stream   WebM · VP9 + Opus
  music    MP3  · libmp3lame
  aac      M4A  · AAC
  master   WAV  · PCM s16le
  preview  GIF  · paleta própria, sem áudio

FAIXA 3 — NÚCLEO. Três caixas lado a lado:
  Go 1.26+      Cobra · binário único, sem runtime
  FFmpeg        binário do usuário · nunca por shell
  Arquitetura   registry de perfis · plano tipado
Abaixo, três pílulas:
  "staging privado" · "publicação atômica" · "sem overwrite implícito"

FAIXA 4 — TESTES. Compacta, três colunas:
  Unitários      tabela · stub de subprocesso
  Integração     conversões reais com FFmpeg
  Estáticos      golangci-lint · CodeQL · govulncheck
Duas pílulas à direita do título da faixa:
  "GitHub Actions" · "detector de corrida"

FAIXA 5 — CI. Uma linha de quatro pílulas, só as combinações reais:
  Linux Go 1.26 · Linux Go 1.27 · macOS Go 1.27 · Windows Go 1.27

FAIXA 6 — CADEIA DE SUPRIMENTO. Três caixas:
  GoReleaser     6 alvos: Linux/macOS/Windows × amd64/arm64 · deb · rpm · apk
  Sigstore       cosign · assinatura keyless
  Proveniência   SBOM · attestation

FAIXA 7 — DISTRIBUIÇÃO. Uma linha de pílulas:
  Homebrew · Scoop · .deb · .rpm · .apk · go install · script

Hierarquia: a faixa 1 domina claramente. As faixas 5 e 7 são as mais discretas
do card. O olho deve pousar na pipeline primeiro.
```

## Otimizar

A imagem que sai do gerador tem cerca de 1,5 MB, pesado demais para um README.
O card é quase todo cor chapada sobre fundo escuro, então uma paleta de 256
cores reduz para cerca de 386 KB sem diferença visível — o único gradiente é o
brilho ao redor de "Verificação", e ele sobrevive sem bandeamento.

```bash
magick entrada.png -strip -colors 256 -define png:compression-level=9 \
  site/architecture-v070.png
```

Confira o brilho ampliado antes de aceitar o resultado:

```bash
magick site/architecture-v070.png -crop 420x130+1150+120 +repage -resize 300% \
  /tmp/glow.png
```
