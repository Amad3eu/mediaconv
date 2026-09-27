# Changelog

Todas as mudanças relevantes deste projeto serão documentadas neste arquivo.

O formato segue o [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/) e o
projeto usa [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## [Não lançado]

### Alterado

- A gravação do terminal passa a mostrar o `doctor` com as verificações de VP9,
  Opus e WebM, além de `--jobs` e de uma conversão para WebM, refletindo o que
  a v0.6.0 faz.

## [0.6.0] - 2026-09-27

### Adicionado

- Perfil `stream`, que converte vídeo para WebM com VP9 e Opus. Use
  `--to webm`.
- Verificações de `libvpx-vp9`, `libopus` e do muxer WebM no `mediaconv
  doctor`.
- Abas por plataforma e botão de copiar nos comandos do site, para que cada
  pessoa veja apenas o caminho que lhe interessa.

### Alterado

- A lista de formatos de saída passa a viver apenas no registry de perfis, em
  vez de duplicada em `internal/app`, como previa a ADR 0003.
- A ajuda de `--to` e `--preset` passa a ser gerada a partir do registry, então
  um formato novo aparece no `--help` sem edição manual.
- Os comandos do site quebram em várias linhas em vez de serem cortados na
  borda, e a gravação do terminal na página inicial ficou maior.

### Corrigido

- Comandos longos no site quebravam no meio de uma palavra, o que fazia uma URL
  de instalação parecer três comandos distintos.

## [0.5.0] - 2026-09-26

### Adicionado

- Identidade visual: símbolo, banner, favicon e ícone de tela inicial, com as
  cores documentadas em `docs/BRAND.md`.
- Metatags Open Graph e Twitter no site, para que links compartilhados exibam o
  banner.
- Opção `--jobs` (`-j`) no comando `batch`, para converter vários arquivos ao
  mesmo tempo. Os resultados continuam sendo reportados na ordem da varredura,
  independentemente da concorrência.
- Saída colorida no terminal: rótulos de status em verde, falhas em vermelho,
  avisos em amarelo e dicas esmaecidas.
- Opção global `--color` com os valores `auto`, `always` e `never`, além do
  suporte à variável de ambiente `NO_COLOR`.
- Demonstração em GIF do CLI no README, no README em português e no site,
  gerada de forma reproduzível pelo VHS a partir de `docs/demo/`.

### Alterado

- As capacidades do FFmpeg passam a ser detectadas uma única vez por execução,
  e não uma vez por arquivo. Em um lote de 20 arquivos curtos isso eliminou 76
  dos 80 processos de detecção e reduziu o tempo total pela metade.
- O site passa a exibir a gravação real do terminal no lugar do bloco de
  exemplo estático, que já não refletia a saída atual do CLI.

## [0.4.0] - 2026-09-08

### Adicionado

- Comando `batch` para converter arquivos suportados em uma pasta.
- Opções `--output-dir`, `--recursive`, `--overwrite`, `--to` e `--preset` no
  fluxo de lote.
- Saída JSON para conversões em lote com resumo e itens por arquivo.
- Lista de formatos ampliada com aliases comuns como QT, M4V, M4B, OGA e OPUS.

## [0.3.0] - 2026-09-08

### Adicionado

- Perfil `music` para converter WAV, FLAC, M4A, AAC, OGG e MP3 para MP3 com
  `libmp3lame`.
- Seleção automática de perfil: `web` para MP4 e `music` para MP3.
- Validação específica para saídas MP3 antes da publicação do arquivo final.
- Teste de integração real para conversão WAV para MP3.

## [0.2.0] - 2026-09-08

### Adicionado

- Suporte do perfil `web` para converter entradas MOV, MKV, AVI e MP4 para MP4
  H.264/AAC, além de WebM.
- Teste de integração real para conversão MOV para MP4.
- Matriz de formatos atualizada no README, README em português e site.

## [0.1.3] - 2026-09-07

### Adicionado

- Script de instalação para Windows via PowerShell publicado no GitHub Pages.
- Documentação do instalador Windows no README, README em português e site do
  projeto.

## [0.1.2] - 2026-09-07

### Adicionado

- Script universal de instalação para Linux e macOS publicado no GitHub Pages.
- Documentação do instalador no README, README em português e site do projeto.

## [0.1.1] - 2026-09-07

### Adicionado

- Pacotes Linux `.deb`, `.rpm` e `.apk` gerados pelo GoReleaser.
- Publicação automática do cask no repositório `Amad3eu/homebrew-tap`.
- Publicação automática do manifesto Scoop no repositório
  `Amad3eu/scoop-bucket`.
- Documentação de instalação via Homebrew, Scoop e pacotes Linux.

## [0.1.0] - 2026-08-29

Primeira versão pública. O FFmpeg continua sendo uma dependência externa e precisa
ser instalado separadamente; use `mediaconv doctor` para conferir a instalação.

### Adicionado

- Comandos `convert`, `inspect`, `doctor`, `formats`, `version` e `completion`.
- Perfil `web`, que converte WebM (VP8/VP9 com Vorbis/Opus) para MP4 amplamente
  compatível usando libx264 com CRF 23, AAC, `yuv420p` e `+faststart`.
- Ajuste automático de dimensões ímpares para pares antes da codificação.
- Escrita em arquivo temporário ao lado do destino, com verificação por ffprobe
  antes da publicação do resultado.
- Proteção contra sobrescrita acidental, exigindo `--overwrite` explícito, e
  recusa de caminhos de saída que sejam links simbólicos.
- Cancelamento por `Ctrl+C` sem deixar arquivos parciais ou temporários.
- Saída legível e `--json` para automação, com códigos de saída documentados.
- CI multiplataforma, CodeQL, análise de vulnerabilidades e revisão de dependências.
- Releases automatizadas com binários, checksums, SBOMs, assinatura keyless e
  atestados de proveniência.

[Não lançado]: https://github.com/Amad3eu/mediaconv/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/Amad3eu/mediaconv/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/Amad3eu/mediaconv/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/Amad3eu/mediaconv/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/Amad3eu/mediaconv/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/Amad3eu/mediaconv/compare/v0.1.3...v0.2.0
[0.1.3]: https://github.com/Amad3eu/mediaconv/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/Amad3eu/mediaconv/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/Amad3eu/mediaconv/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/Amad3eu/mediaconv/releases/tag/v0.1.0
