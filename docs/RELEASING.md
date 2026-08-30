# Processo de release

As releases são publicadas pelo workflow `.github/workflows/release.yml` quando uma
tag SemVer com prefixo `v` é enviada ao GitHub. O GoReleaser compila os binários,
gera arquivos compactados, checksum SHA-256, SBOMs e uma assinatura keyless do
arquivo de checksums. O GitHub também registra um atestado de proveniência para os
artefatos publicados.

## Antes de criar a tag

1. Confirme que a CI da branch `main` está verde e que não há mudanças locais.
2. Atualize a seção `Não lançado` do `CHANGELOG.md` para a nova versão e abra outra
   seção `Não lançado` vazia.
3. Confirme a versão localmente com `go test ./...` e um snapshot do GoReleaser.
4. Integre a mudança de release em `main` antes de criar a tag.

```bash
git switch main
git pull --ff-only origin main
git status --short
go test ./...
goreleaser release --snapshot --clean --skip=publish,sign
```

## Criar e publicar

Use uma tag anotada e assinada sempre que houver uma chave Git configurada:

```bash
git tag -s v0.1.0 -m "mediaconv v0.1.0"
git push origin v0.1.0
```

Se assinatura Git ainda não estiver configurada, uma tag anotada é aceitável:

```bash
git tag -a v0.1.0 -m "mediaconv v0.1.0"
git push origin v0.1.0
```

O workflow aceita versões como `v1.2.3` e pré-releases como `v1.2.3-rc.1`. Não
execute o GoReleaser manualmente com um token de produção e não mova uma tag depois
que ela tiver sido publicada.

## Validar a publicação

Depois que o workflow terminar:

1. confira as notas e todos os alvos na página da release;
2. baixe o arquivo da sua plataforma, o checksum e o bundle de assinatura;
3. valide a identidade Sigstore e depois o checksum;
4. valide o atestado de proveniência com o GitHub CLI;
5. execute `mediaconv version` e uma conversão curta em pelo menos uma plataforma.

Exemplo para `v0.1.0`, ajustando os nomes dos arquivos baixados:

```bash
repository='github.com/Amad3eu/mediaconv'
workflow='.github/workflows/release.yml@refs/tags/v0.1.0'
certificate_identity="https://${repository}/${workflow}"

cosign verify-blob \
  --bundle mediaconv_0.1.0_checksums.txt.bundle \
  --certificate-identity "${certificate_identity}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  mediaconv_0.1.0_checksums.txt

sha256sum --ignore-missing -c mediaconv_0.1.0_checksums.txt

gh attestation verify mediaconv_0.1.0_linux_amd64.tar.gz \
  --repo Amad3eu/mediaconv
```

No macOS, use `shasum -a 256` para conferir manualmente o valor quando
`sha256sum` não estiver disponível. No Windows, use `Get-FileHash -Algorithm SHA256`.

## Se uma release apresentar problema

Não apague nem reutilize a tag. Marque a release como descontinuada, documente o
problema e publique uma nova versão patch. Para uma vulnerabilidade, siga
`SECURITY.md` e coordene a divulgação por um GitHub Security Advisory.
