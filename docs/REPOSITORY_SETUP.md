# Configuração recomendada do repositório

Estas configurações são feitas uma única vez na interface do GitHub e não podem ser
aplicadas somente pelos arquivos versionados.

## Geral

- Defina `main` como branch padrão.
- Habilite squash merge e use o título do pull request como mensagem do commit.
- Desabilite merge commits para manter histórico linear.
- Habilite a exclusão automática de branches depois do merge.
- Adicione os tópicos `cli`, `go`, `ffmpeg`, `video-converter`, `webm` e `mp4`.

## Ruleset de `main`

Crie uma ruleset que:

- exija pull request antes do merge;
- exija que a branch esteja atualizada;
- exija os checks obrigatórios listados abaixo;
- exija histórico linear e resolução de todas as conversas;
- bloqueie force push e exclusão da branch;
- permita bypass apenas ao mantenedor para emergências documentadas.

### Checks obrigatórios

> [!IMPORTANT]
> O GitHub Actions **não cria um check com o nome do workflow**. Ele cria um
> check por *job*. Não existe um contexto chamado `CI`; exigir esse nome deixa
> a ruleset esperando para sempre por um status que nunca é reportado, e
> nenhum pull request consegue mergear.

Exija os nomes dos jobs, exatamente como aparecem no PR:

```text
Quality checks
Test (ubuntu-latest, Go 1.26.x)
Test (ubuntu-latest, Go 1.27.x)
Test (macos-latest, Go 1.27.x)
Test (windows-latest, Go 1.27.x)
FFmpeg integration
Vulnerability scan
Validate release
Analyze Go
```

Dois cuidados ao alterar essa lista:

- Um job só pode virar obrigatório **depois** que ele existe na `main`. Exigir
  um job novo antes disso trava os pull requests abertos, que ainda não o
  possuem.
- Os nomes dos jobs da matriz mudam junto com a matriz. Se você alterar as
  versões de Go ou os sistemas operacionais em `ci.yml`, atualize a ruleset na
  mesma mudança.

O job `Dependency review` só funciona com o Dependency graph habilitado (veja
[Segurança](#segurança)). Só o torne obrigatório depois de habilitá-lo.

Para um projeto mantido inicialmente por uma pessoa, não exija aprovação de outro
revisor, pois isso impediria releases. Ative uma aprovação obrigatória quando houver
outro mantenedor ativo.

## Segurança

- Habilite Dependency Graph, Dependabot alerts e Dependabot security updates.
  O Dependency graph precisa ser ligado pela interface, em
  *Settings > Code security*: a API de repositório aceita o `PATCH` e ignora
  esse campo em silêncio. Sem ele, o job `Dependency review` falha com
  "Dependency review is not supported on this repository".
- Habilite Private Vulnerability Reporting.
- Habilite code scanning, secret scanning e push protection, quando disponíveis.
- Use permissões de workflow somente leitura por padrão; permita escrita apenas ao
  workflow de release.
- Proteja tags que correspondam a `v*` contra atualização e exclusão.

## Labels e automação

Crie as labels `bug`, `enhancement`, `triage`, `dependencies`, `go` e
`github-actions`, referenciadas pelos formulários e pelo Dependabot. O Dependabot
agrupa atualizações de módulos Go e Actions em pull requests semanais separados.

## Verificação

Este documento descreve estado que vive no GitHub, não no repositório, então
ele envelhece sem que nada quebre visivelmente. Confira periodicamente se o que
está configurado ainda corresponde ao que está escrito aqui:

```bash
# Merge, branches e segurança.
gh api repos/Amad3eu/mediaconv --jq '{
  allow_merge_commit, allow_rebase_merge, allow_squash_merge,
  delete_branch_on_merge, use_squash_pr_title_as_default,
  security: .security_and_analysis
}'

# Checks obrigatórios efetivamente exigidos.
gh api repos/Amad3eu/mediaconv/rulesets \
  --jq '.[] | select(.name == "Protect main") | .id' \
  | xargs -I{} gh api repos/Amad3eu/mediaconv/rulesets/{} \
    --jq '.rules[] | select(.type == "required_status_checks")
          | .parameters.required_status_checks[].context'

# Nomes reais dos checks em um pull request aberto, para comparar com a lista
# acima. Substitua NUMERO pelo número do pull request.
gh api "repos/Amad3eu/mediaconv/commits/$(
  gh api repos/Amad3eu/mediaconv/pulls/NUMERO --jq .head.sha
)/check-runs" --jq '.check_runs[].name' | sort -u
```

Um contexto exigido que não apareça na última lista nunca será reportado, e
todos os pull requests ficarão parados em *Expected — Waiting for status to be
reported*.

## Primeira release

Antes de enviar a primeira tag, confirme que o repositório público contém a licença,
o README, esta documentação e uma execução bem-sucedida da CI. Em seguida siga
[RELEASING.md](RELEASING.md).
