# Como contribuir

Obrigado por considerar uma contribuição para o `mediaconv`. Mudanças pequenas,
focadas e acompanhadas por testes são mais fáceis de revisar e manter.

Ao participar, você concorda com o [Código de Conduta](CODE_OF_CONDUCT.md).

## Preparação do ambiente

Você precisará de:

- Go 1.26 ou superior;
- Git;
- FFmpeg e ffprobe no `PATH` para testes de integração e uso real.

Faça um fork, clone-o e crie uma branch a partir de `main`:

```bash
git clone https://github.com/SEU-USUARIO/mediaconv.git
cd mediaconv
git remote add upstream https://github.com/Amad3eu/mediaconv.git
git switch -c feat/minha-mudanca
```

Antes de iniciar uma mudança grande ou incompatível, abra uma issue para alinhar
a proposta. Correções pequenas e bem delimitadas podem ir diretamente para um
pull request.

## Desenvolvimento

Execute o CLI durante o desenvolvimento com:

```bash
go run ./cmd/mediaconv --help
```

Antes de enviar uma mudança, rode:

```bash
gofmt -w .
go mod tidy
go vet ./...
go test -race ./...
go build -trimpath ./cmd/mediaconv
```

Se alterar o empacotamento, valide também a configuração com GoReleaser. A geração
de SBOM exige o Syft instalado:

```bash
goreleaser check
goreleaser release --snapshot --clean --skip=publish,sign
```

Não inclua vídeos grandes, material protegido ou dados pessoais nos testes. Prefira
fixtures curtas, sintéticas e geradas de maneira reproduzível.

## Commits e pull requests

Use mensagens de commit curtas e no imperativo. O projeto adota prefixos no estilo
Conventional Commits, por exemplo:

```text
feat: add mp4 to webm profile
fix: preserve output when conversion is canceled
docs: explain ffmpeg installation
test: cover paths containing unicode
```

Um pull request deve:

- explicar o problema e a solução escolhida;
- manter uma única finalidade principal;
- adicionar ou atualizar testes para alterações de comportamento;
- atualizar documentação e `CHANGELOG.md` quando houver impacto para usuários;
- passar por todos os checks do GitHub Actions;
- evitar alterações de formatação ou refactors não relacionados.

Atualize sua branch sem criar merges desnecessários:

```bash
git fetch upstream
git rebase upstream/main
git push --force-with-lease
```

Use `--force-with-lease` somente na sua própria branch de contribuição. Nunca
reescreva `main` nem uma tag publicada.

## Compatibilidade

O código deve continuar compilando com a versão mínima de Go declarada em `go.mod`
e nas plataformas cobertas pela CI. Não presuma um shell Unix no código do CLI; os
argumentos do FFmpeg devem ser passados diretamente ao processo, sem concatenação
de comandos de shell.

Novos conversores devem ser implementados como perfis independentes, com validação
de entrada, testes de argumentos e mensagens de erro úteis. Mudanças incompatíveis
precisam de discussão prévia e documentação de migração.

## Licenças

Ao enviar uma contribuição, você concorda que ela seja distribuída sob a licença
do repositório. Não copie código ou mídias incompatíveis com essa licença. Se
adicionar uma dependência, registre qualquer aviso necessário em
`THIRD_PARTY_NOTICES.md`.
