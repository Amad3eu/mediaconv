# Marca

O símbolo funde o botão de play (`▶`) com o chevron de prompt de terminal
(`❯`): uma forma só, com dupla leitura, dizendo mídia e linha de comando ao
mesmo tempo.

## Paleta

| Uso | Cor |
| --- | --- |
| Símbolo | `#0f766e` |
| Fundo escuro do banner e do painel de terminal | `#0b1220` |
| Texto sobre fundo claro | `#172026` |

São as mesmas variáveis de [`site/styles.css`](../site/styles.css), então a
marca, o site e a gravação do terminal usam um único conjunto de cores.

## Arquivos

Todos vivem em `site/`, que é o diretório publicado no GitHub Pages, então cada
um tem uma URL estável em `https://amad3eu.github.io/mediaconv/`.

| Arquivo | Tamanho | Para que serve |
| --- | --- | --- |
| [`site/brand/banner.png`](../site/brand/banner.png) | 1280×640 | Título dos READMEs e social preview do repositório |
| [`site/brand/mark.png`](../site/brand/mark.png) | 512×512 | Símbolo isolado em teal, fundo transparente |
| [`site/brand/mark-ink.png`](../site/brand/mark-ink.png) | 512×512 | Símbolo monocromático escuro, para quando o teal não serve |
| [`site/brand/wordmark.png`](../site/brand/wordmark.png) | 1200×239 | Símbolo com o nome ao lado, para fundos claros |
| [`site/favicon.png`](../site/favicon.png) | 32×32 | Ícone da aba do navegador |
| [`site/apple-touch-icon.png`](../site/apple-touch-icon.png) | 180×180 | Ícone de tela inicial no iOS |

O `wordmark.png` tem o texto em `#172026` sobre fundo transparente, então ele
**some em fundo escuro**. Em contexto que possa ser claro ou escuro, use o
`banner.png`, que carrega o próprio fundo.

## Social preview do repositório

O banner tem exatamente a proporção 2:1 que o GitHub espera. Ele não pode ser
configurado pela API: envie `site/brand/banner.png` em
*Settings > General > Social preview*.

## Regenerar

Os arquivos acima são derivados de artes maiores. Se você substituir a arte,
recorte o símbolo pela caixa delimitadora do canal alfa antes de redimensionar:
as artes originais deixavam o símbolo ocupando pouco mais de 60% do quadrado e
fora do centro, o que desperdiça pixels justamente em 16×16, onde eles fazem
mais falta.
