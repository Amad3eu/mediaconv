# Changelog

Todas as mudanças relevantes deste projeto serão documentadas neste arquivo.

O formato segue o [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/) e o
projeto usa [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## [Não lançado]

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

[Não lançado]: https://github.com/Amad3eu/mediaconv/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Amad3eu/mediaconv/releases/tag/v0.1.0
