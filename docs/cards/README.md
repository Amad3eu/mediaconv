# Cards de post

Cada imagem aqui é gerada a partir de um arquivo HTML versionado ao lado dela,
não desenhada à mão, pelo mesmo motivo que vale para os
[gráficos de desempenho](../benchmark/README.md): o card afirma coisas sobre o
repositório, e afirmação dentro de imagem ninguém consegue conferir depois.

| Imagem | Fonte | Versão |
| --- | --- | --- |
| [`site/card-verificacao-v070.png`](../../site/card-verificacao-v070.png) | [`verificacao-v070.html`](verificacao-v070.html) | v0.7.0 |

## Regenerar

Qualquer navegador headless serve. Com o Chrome ou Chromium:

```bash
chrome --headless --disable-gpu --hide-scrollbars \
  --window-size=1200,556 \
  --screenshot=site/card-verificacao-v070.png \
  "file://$PWD/docs/cards/verificacao-v070.html"
```

Confira contra o PNG commitado antes de aceitar um render feito em outra
máquina, porque fonte ausente muda o layout sem erro nenhum:

```bash
magick compare -metric RMSE site/card-verificacao-v070.png novo.png null:
```

## O sistema de design

Os tokens são os mesmos de [`site/styles.css`](../../site/styles.css), listados
em [BRAND.md](../BRAND.md), e a estrutura é a mesma de
[`docs/benchmark/v060.html`](../benchmark/v060.html): `eyebrow` com a versão em
teal maiúsculo, `h1`, `sub` discreta, o conteúdo, e um rodapé com a frase que
resume o card à esquerda e `github.com/Amad3eu/mediaconv` à direita.

O que muda entre um card e outro é só o miolo. No gráfico de desempenho ele são
barras, porque o dado é tempo. Aqui são duas strings, porque o dado é a
diferença entre o que o teste procurava e o que o código quebrado imprimia.

## De onde vem o que este card afirma

A asserção em questão verificava a saída de `mediaconv info` separando streams
de vídeo de streams de áudio. Ela procurava `"2 channel(s)"` na linha do stream
de vídeo para provar que a contagem de canais não vazava para lá.

O stream de vídeo do fixture tem `Channels: 0`, então a versão quebrada da
guarda imprimia `"0 channel(s)"`. A busca não batia, a asserção não falhava, e
o teste passava contra código quebrado.

Isso foi encontrado rodando a suíte contra mutações deliberadas do código, não
lendo o teste. A asserção foi reescrita para conferir o rótulo por linha em vez
da contagem, e revalidada contra três mutações. A asserção corrigida está em
`internal/cli/render_test.go`.
