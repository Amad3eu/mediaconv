# Política de segurança

## Versões suportadas

Enquanto o projeto estiver na série `0.x`, correções de segurança serão feitas na
versão estável mais recente. Versões anteriores podem deixar de receber correções.

| Versão | Suporte |
| --- | --- |
| Release mais recente | Sim |
| Versões anteriores | Não garantido |
| Branch `main` | Apenas para desenvolvimento |

## Como reportar uma vulnerabilidade

Não abra uma issue pública para uma vulnerabilidade ainda não corrigida. Envie um
relato privado por meio de um
[GitHub Security Advisory](https://github.com/Amad3eu/mediaconv/security/advisories/new).

Inclua, quando possível:

- versão do `mediaconv`, sistema operacional e arquitetura;
- versão e origem do FFmpeg/ffprobe instalados;
- descrição do impacto e das condições necessárias para reproduzi-lo;
- passos mínimos de reprodução ou prova de conceito sem dados pessoais;
- sugestões de mitigação, caso existam.

Evite anexar mídias privadas ou confidenciais. Use um arquivo sintético mínimo que
demonstre o problema.

O objetivo é confirmar o recebimento em até sete dias. Depois da validação, o
mantenedor coordenará a correção e a publicação do advisory antes da divulgação
pública. O prazo da correção depende da gravidade e da complexidade do problema.

## Escopo

O `mediaconv` executa os binários FFmpeg e ffprobe instalados pelo usuário; esses
binários não fazem parte do projeto. Vulnerabilidades que existam exclusivamente
no FFmpeg devem ser reportadas ao projeto FFmpeg. Problemas na forma como o
`mediaconv` valida caminhos, cria arquivos, monta argumentos, trata conteúdo não
confiável ou expõe informações pertencem a este projeto e devem ser reportados
aqui.
