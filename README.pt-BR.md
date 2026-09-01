# MediaConv

[English](README.md) · [Português (Brasil)](README.pt-BR.md)

[![CI](https://github.com/Amad3eu/mediaconv/actions/workflows/ci.yml/badge.svg)](https://github.com/Amad3eu/mediaconv/actions/workflows/ci.yml)
[![CodeQL](https://github.com/Amad3eu/mediaconv/actions/workflows/codeql.yml/badge.svg)](https://github.com/Amad3eu/mediaconv/actions/workflows/codeql.yml)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)](go.mod)
[![Licença: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Site](https://img.shields.io/badge/Site-GitHub%20Pages-222222?logo=githubpages)](https://amad3eu.github.io/mediaconv/)

MediaConv é um conversor de mídia por linha de comando, seguro e amigável a
automações, baseado no FFmpeg.

Ele começa com uma tarefa bem resolvida: converter vídeos WebM em MP4 amplamente
compatível, usando vídeo H.264 e áudio AAC. O MediaConv valida a entrada,
converte em uma área temporária privada, verifica o resultado e somente então
publica a saída.

> [!NOTE]
> O MediaConv está em desenvolvimento inicial. Até a v1.0, comandos e opções
> podem mudar entre versões menores.

## Por que usar o MediaConv?

O FFmpeg é poderoso, mas a linha de comando para uma conversão WebM para MP4
segura é fácil de errar. O MediaConv empacota esse fluxo em um CLI pequeno, com
padrões previsíveis, diagnóstico claro, saída estruturada para automações e uma
arquitetura pronta para receber mais conversores no futuro.

Use o MediaConv quando você quiser:

- um comando curto em vez de decorar flags do FFmpeg;
- saída verificada antes de criar ou substituir o arquivo final;
- erros claros para codecs ausentes, entrada corrompida ou conflito de saída;
- um CLI que funciona em scripts com JSON e códigos de saída tipados;
- uma base pronta para crescer com novos perfis de conversão.

## Recursos

- Conversão local de WebM para MP4 com foco em compatibilidade.
- Progresso interativo no terminal e saída limpa para scripts.
- Nenhuma sobrescrita sem a opção explícita `--overwrite`.
- Limpeza de arquivos temporários após falha ou interrupção.
- Caminhos com espaços e Unicode enviados diretamente ao FFmpeg, sem shell.
- Saída legível para pessoas e no formato JSON.
- Diagnóstico de dependências e codecs com `mediaconv doctor`.
- Binários nativos para Linux, macOS e Windows em AMD64 e ARM64.

## Início rápido

Instale o FFmpeg primeiro, depois instale o MediaConv pela última release ou com
Go.

```bash
# Verifique o FFmpeg e os codecs necessários.
mediaconv doctor

# Inspecione o arquivo de entrada.
mediaconv inspect "gravacao.webm"

# Crie gravacao.mp4 ao lado da entrada.
mediaconv convert "gravacao.webm"

# Escolha a saída e permita explicitamente sua substituição.
mediaconv convert "gravacao.webm" \
  --output "exportados/gravacao.mp4" \
  --overwrite
```

Site do projeto: <https://amad3eu.github.io/mediaconv/>

Última release: <https://github.com/Amad3eu/mediaconv/releases/latest>

## Requisitos

O MediaConv não inclui nem baixa o FFmpeg. Instale `ffmpeg` e `ffprobe` antes de
usá-lo. O perfil inicial `web` também exige o encoder de vídeo `libx264`, o encoder
de áudio AAC e o muxer MP4.

Comandos comuns de instalação:

```bash
# Debian / Ubuntu
sudo apt update && sudo apt install ffmpeg

# macOS com Homebrew
brew install ffmpeg

# Arch Linux
sudo pacman -S ffmpeg
```

No Windows, uma opção referenciada pela
[página oficial de downloads do FFmpeg](https://ffmpeg.org/download.html) é:

```powershell
winget install --id Gyan.FFmpeg --exact --source winget
```

As compilações do FFmpeg variam. Execute `mediaconv doctor` em vez de presumir que
um pacote possui todos os codecs.

## Instalação do MediaConv

### Arquivo de release

Baixe o arquivo correspondente ao seu sistema em
[GitHub Releases](https://github.com/Amad3eu/mediaconv/releases/latest), confira-o
com o checksum publicado, extraia-o e coloque `mediaconv` em um diretório do
`PATH`.

### Com a toolchain Go

```bash
go install github.com/Amad3eu/mediaconv/cmd/mediaconv@latest
```

Instalar o MediaConv com Go não instala o FFmpeg.

### Compilar o código-fonte

```bash
git clone https://github.com/Amad3eu/mediaconv.git
cd mediaconv
go build -trimpath -o ./bin/mediaconv ./cmd/mediaconv
```

O desenvolvimento exige Go 1.26 ou mais recente.

## Comandos

```text
mediaconv convert INPUT [--to mp4] [-o SAÍDA] [--preset web] [--overwrite]
mediaconv inspect INPUT
mediaconv doctor
mediaconv formats
mediaconv version
mediaconv completion bash|zsh|fish|powershell
```

Use `mediaconv COMANDO --help` para ver todas as opções e exemplos. As opções
globais incluem `--json`, `--verbose`, `--ffmpeg-path` e `--ffprobe-path`.

### JSON e códigos de saída

Use `--json` em automações. Resultados bem-sucedidos são escritos em stdout;
progresso e diagnósticos usam stderr. O progresso interativo é desativado
automaticamente quando stderr não é um terminal.

| Código | Significado |
| ---: | --- |
| 0 | Sucesso |
| 1 | Erro interno inesperado |
| 2 | Comando, opção ou argumento inválido |
| 3 | Dependência ou capacidade do FFmpeg ausente |
| 4 | Entrada inválida, corrompida ou não suportada |
| 5 | Conflito de saída ou falha na publicação |
| 6 | Falha na conversão ou verificação da saída |
| 130 | Interrompido pelo usuário |

## Conversões suportadas

| Entrada | Saída | Perfil | Vídeo | Áudio | Situação |
| --- | --- | --- | --- | --- | --- |
| WebM | MP4 | `web` | H.264 (`libx264`, CRF 23) | AAC 192 kbit/s | Inicial |

O perfil `web` converte o primeiro vídeo e o primeiro áudio opcional. Ele gera
`yuv420p`, preserva metadados compatíveis, descarta capítulos e legendas, ajusta
dimensões ímpares para valores pares e ativa fast start no MP4. O CLI avisa quando
streams extras, transparência, capítulos, legendas ou HDR podem ser perdidos.

## Segurança e privacidade

- Somente arquivos locais regulares são aceitos. URLs, dispositivos e pipes não são suportados.
- O FFmpeg recebe uma lista de argumentos e nunca é executado por `sh`, `cmd.exe` ou outro shell.
- A conversão acontece em uma área temporária privada no mesmo sistema de arquivos da saída.
- Saídas existentes e links simbólicos são rejeitados, exceto quando um arquivo regular é substituído explicitamente.
- A saída verificada é publicada de forma atômica nos sistemas de arquivos compatíveis.
- Os arquivos são processados localmente e nunca enviados pelo MediaConv.
- Não existe telemetria.

Sem `--overwrite`, a publicação utiliza um hard link para impedir que outro processo
faça o MediaConv substituir um destino criado durante a conversão. O sistema de
arquivos da saída precisa oferecer suporte a hard links. Isso é comum em NTFS,
APFS, ext4 e sistemas locais semelhantes, mas pode não estar disponível em alguns
discos removíveis ou compartilhamentos de rede.

## Próximos passos

- Perfis adicionais, como MP4 para WebM e MOV/MKV para MP4.
- Extração de áudio para MP3, AAC e WAV.
- Conversão em lote com controle conservador de concorrência.
- Distribuição por gerenciadores de pacotes depois que a interface estabilizar.
- Aceleração por hardware após a criação de testes específicos por capacidade.

Plugins dinâmicos e binários do FFmpeg incluídos estão propositalmente fora do
escopo inicial. Consulte [a arquitetura](docs/ARCHITECTURE.md) para entender os
limites do projeto.

## Contribuição e segurança

Leia [CONTRIBUTING.md](CONTRIBUTING.md) antes de abrir um pull request. Relate
problemas de segurança privadamente seguindo [SECURITY.md](SECURITY.md). Os
mantenedores também devem aplicar as configurações descritas em
[docs/REPOSITORY_SETUP.md](docs/REPOSITORY_SETUP.md).

## Licença e FFmpeg

O MediaConv está disponível sob a [Licença MIT](LICENSE). O FFmpeg é um projeto
separado cuja licença depende de sua configuração de compilação. O MediaConv chama
os executáveis instalados pelo usuário e não os redistribui. Consulte
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) para mais detalhes.
