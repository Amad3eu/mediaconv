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
- exija todos os checks do workflow `CI`;
- exija histórico linear e resolução de todas as conversas;
- bloqueie force push e exclusão da branch;
- permita bypass apenas ao mantenedor para emergências documentadas.

Para um projeto mantido inicialmente por uma pessoa, não exija aprovação de outro
revisor, pois isso impediria releases. Ative uma aprovação obrigatória quando houver
outro mantenedor ativo.

## Segurança

- Habilite Dependency Graph, Dependabot alerts e Dependabot security updates.
- Habilite Private Vulnerability Reporting.
- Habilite code scanning, secret scanning e push protection, quando disponíveis.
- Use permissões de workflow somente leitura por padrão; permita escrita apenas ao
  workflow de release.
- Proteja tags que correspondam a `v*` contra atualização e exclusão.

## Labels e automação

Crie as labels `bug`, `enhancement`, `triage`, `dependencies`, `go` e
`github-actions`, referenciadas pelos formulários e pelo Dependabot. O Dependabot
agrupa atualizações de módulos Go e Actions em pull requests semanais separados.

## Primeira release

Antes de enviar a primeira tag, confirme que o repositório público contém a licença,
o README, esta documentação e uma execução bem-sucedida da CI. Em seguida siga
[RELEASING.md](RELEASING.md).
