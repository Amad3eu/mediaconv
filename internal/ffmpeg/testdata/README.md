# Fixtures de detecção

Saída literal de um FFmpeg real, usada por `capabilities_test.go` para exercitar
o parser contra o formato genuíno em vez de um inventado no teste. Assim o
pacote é testado em máquinas e runners sem FFmpeg instalado, e o formato tem
procedência, como os números em [docs/benchmark](../../../docs/benchmark).

| Arquivo | Comando |
| --- | --- |
| `ffmpeg-version.txt` | `ffmpeg -version` |
| `ffmpeg-encoders.txt` | `ffmpeg -hide_banner -encoders` |
| `ffmpeg-muxers.txt` | `ffmpeg -hide_banner -muxers` |

Gravados com `ffmpeg version N-126313-g1ae4048218-20260829`.

Para regravar com outro build, rode os três comandos acima redirecionando para
os arquivos correspondentes. Os testes conferem nomes de encoders e muxers que
qualquer build completo tem, então um build enxuto vai falhar — o que é o ponto:
o fixture precisa representar um FFmpeg que sirva para os seis perfis.
