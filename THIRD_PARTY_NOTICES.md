# Avisos de terceiros

Este arquivo resume componentes de terceiros relevantes ao `mediaconv`. As versões
efetivamente usadas em cada release estão registradas em `go.mod`, `go.sum` e nos
arquivos SBOM anexados à release.

## Dependências Go

- [Cobra](https://github.com/spf13/cobra), sob a licença Apache-2.0, fornece a
  estrutura de comandos do CLI.
- [pflag](https://github.com/spf13/pflag), sob a licença BSD-3-Clause, fornece o
  processamento de flags usado pelo Cobra.
- [mousetrap](https://github.com/inconshreveable/mousetrap), sob a licença
  BSD-3-Clause, pode ser usado pelo Cobra em builds para Windows.
- [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys), sob uma licença
  BSD-3-Clause, fornece interfaces de baixo nível específicas de cada sistema.
- A biblioteca padrão do [Go](https://go.dev/LICENSE) é distribuída sob uma licença
  BSD de três cláusulas.

Consulte os arquivos de licença dos respectivos projetos para os termos completos.
Dependências indiretas podem mudar entre versões; o SBOM da release é a fonte mais
precisa para uma versão publicada.

## FFmpeg e ffprobe

O `mediaconv` chama executáveis FFmpeg e ffprobe fornecidos pelo ambiente do usuário.
Eles não são incorporados nem redistribuídos nos arquivos oficiais do `mediaconv`.

FFmpeg é um projeto separado e pode ser distribuído sob LGPL ou GPL, dependendo da
configuração usada para compilar o binário. Em especial, builds que habilitam certos
codecs e bibliotecas podem estar sob GPL; isso inclui normalmente builds com
`libx264`, usado pelo perfil inicial. Consulte os [avisos legais oficiais do
FFmpeg](https://ffmpeg.org/legal.html) e o fornecedor do seu binário para conhecer
os termos aplicáveis.

As marcas e nomes de projetos de terceiros pertencem aos seus respectivos titulares.
