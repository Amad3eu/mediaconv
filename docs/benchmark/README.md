# Gráficos de desempenho

Cada imagem aqui é gerada a partir de um arquivo HTML versionado ao lado dela,
não desenhada à mão, para que os números tenham procedência e o gráfico possa
ser refeito.

| Imagem | Fonte | Versão |
| --- | --- | --- |
| [`site/benchmark-v060.png`](../../site/benchmark-v060.png) | [`v060.html`](v060.html) | v0.6.0 |

## Regenerar

Qualquer navegador headless serve. Com o Chrome ou Chromium:

```bash
chrome --headless --disable-gpu --hide-scrollbars \
  --window-size=1200,556 \
  --screenshot=site/benchmark-v060.png \
  "file://$PWD/docs/benchmark/v060.html"
```

## De onde vêm os números da v0.6.0

Vinte arquivos WAV de um segundo convertidos para MP3, medidos em uma máquina de
vinte núcleos, com o FFmpeg do sistema. As três barras são o mesmo trabalho em
três versões do código:

| Versão | Tempo | Processos do FFmpeg |
| --- | ---: | ---: |
| Antes | 11,83s | 140, dos quais 80 só detectando capabilities |
| Detecção memoizada | 5,51s | 64, dos quais 4 de detecção |
| Com `--jobs 8` | 0,97s | 64 |

A contagem de processos foi obtida substituindo `ffmpeg` e `ffprobe` por um
wrapper que registra cada invocação antes de repassar para o binário real. Os
tempos são de execução única, então tratam-se de ordens de grandeza, não de uma
média estatística.

## A cor

A barra usa `#0d9488`, não o `#0f766e` da marca. O teal da identidade tem croma
abaixo do piso recomendado para marca de gráfico e lê como cinza quando vira uma
área preenchida. O `#0d9488` é o tom saturado mais próximo que passa nas
verificações de luminosidade, croma e contraste sobre o fundo escuro. Veja
[BRAND.md](../BRAND.md) para a paleta da identidade.
