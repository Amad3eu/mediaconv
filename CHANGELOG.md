# Changelog

Todas as mudanças relevantes deste projeto serão documentadas neste arquivo.

O formato segue o [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/) e o
projeto usa [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## [Não lançado]

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

[Não lançado]: https://github.com/Amad3eu/mediaconv/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/Amad3eu/mediaconv/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/Amad3eu/mediaconv/compare/v0.1.3...v0.2.0
[0.1.3]: https://github.com/Amad3eu/mediaconv/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/Amad3eu/mediaconv/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/Amad3eu/mediaconv/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/Amad3eu/mediaconv/releases/tag/v0.1.0
